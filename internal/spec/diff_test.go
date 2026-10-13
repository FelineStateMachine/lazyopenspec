package spec

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

// ev is the comparable part of an Event (Time is checked separately).
type ev struct {
	Kind   EventKind
	Item   ItemKind
	ID     string
	Rel    string
	Detail string
}

func strip(t *testing.T, events []Event, now time.Time) []ev {
	t.Helper()
	var out []ev
	for _, e := range events {
		if !e.Time.Equal(now) {
			t.Errorf("event %+v: Time = %v, want %v", e, e.Time, now)
		}
		out = append(out, ev{e.Kind, e.Item, e.ID, e.Rel, e.Detail})
	}
	return out
}

func assertEvents(t *testing.T, got []Event, now time.Time, want []ev) {
	t.Helper()
	if g := strip(t, got, now); !reflect.DeepEqual(g, want) {
		t.Errorf("Diff events mismatch\n got: %s\nwant: %s", fmtEvents(g), fmtEvents(want))
	}
}

func fmtEvents(es []ev) string {
	var b strings.Builder
	for _, e := range es {
		b.WriteString("\n  ")
		b.WriteString(strings.Join([]string{
			[]string{"Created", "Edited", "Progress", "Archived", "Removed"}[e.Kind],
			[]string{"Change", "Spec", "Archive"}[e.Item], e.ID, e.Rel, e.Detail,
		}, " | "))
	}
	return b.String()
}

// touch rewrites a file (or creates it) and pushes its mtime forward so the
// change is visible even on filesystems with coarse timestamps.
func touch(t *testing.T, path, content string) {
	t.Helper()
	writeFile(t, path, content)
	bump(t, path)
}

func bump(t *testing.T, path string) {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	later := fi.ModTime().Add(time.Hour)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}
}

func appendTo(t *testing.T, path, extra string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	touch(t, path, string(b)+extra)
}

const newRequirement = `
### Requirement: Excel export
The system SHALL export a project's tasks as an XLSX workbook.

#### Scenario: Export as XLSX
- **WHEN** a project owner requests an XLSX export
- **THEN** the system returns a workbook with one row per task
`

func TestDiffAgentEdits(t *testing.T) {
	root := copyFixture(t)
	changes := filepath.Join(root, "openspec", "changes")
	prev := mustLoad(t, root)

	// 1. Check task 2.2 (the first unchecked top-level task).
	tasks := filepath.Join(changes, "harden-session-security", "tasks.md")
	b, err := os.ReadFile(tasks)
	if err != nil {
		t.Fatal(err)
	}
	checked := strings.Replace(string(b), "- [ ] 2.2", "- [x] 2.2", 1)
	if checked == string(b) {
		t.Fatal("task 2.2 not found")
	}
	touch(t, tasks, checked)

	// 2. A new change dir with a proposal.
	touch(t, filepath.Join(changes, "add-task-comments", ".openspec.yaml"), "schema: spec-driven\ncreated: 2026-10-13\n")
	touch(t, filepath.Join(changes, "add-task-comments", "proposal.md"), "## Why\n\nMembers discuss tasks in Slack.\n")

	// 3. design.md added to an existing change.
	touch(t, filepath.Join(changes, "add-task-search", "design.md"), "## Context\n\nPostgres FTS.\n")

	// 4. A main spec gains a requirement.
	appendTo(t, filepath.Join(root, "openspec", "specs", "data-export", "spec.md"), newRequirement)

	// 5. Archive the completed change.
	if err := os.Rename(filepath.Join(changes, "add-csv-export-filters"),
		filepath.Join(changes, "archive", "2026-10-13-add-csv-export-filters")); err != nil {
		t.Fatal(err)
	}

	// 6. Delete an artifact.
	if err := os.Remove(filepath.Join(changes, "harden-session-security", "design.md")); err != nil {
		t.Fatal(err)
	}

	next := mustLoad(t, root)
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	assertEvents(t, Diff(prev, next, now), now, []ev{
		{EventCreated, ItemChange, "add-task-comments", "", "new change"},
		{EventCreated, ItemChange, "add-task-search", "design.md", "design.md"},
		{EventProgress, ItemChange, "harden-session-security", "tasks.md", "tasks 4/10 → 5/10"},
		{EventRemoved, ItemChange, "harden-session-security", "", "design.md"},
		{EventArchived, ItemArchive, "2026-10-13-add-csv-export-filters", "", "archived"},
		{EventEdited, ItemSpec, "data-export", "spec.md", "requirements 3 → 4"},
	})

	// Sanity: the snapshot itself reflects the mutations.
	if h, _ := next.FindChange("harden-session-security"); h.Tasks != (TaskStats{Done: 5, Total: 10}) || h.Has(KindDesign) {
		t.Errorf("harden-session-security after edits: Tasks=%+v Has(design)=%v", h.Tasks, h.Has(KindDesign))
	}
	if a, ok := next.FindChange("2026-10-13-add-csv-export-filters"); !ok || !a.Archived || a.Name != "add-csv-export-filters" {
		t.Errorf("archived change = %+v, %v", a, ok)
	}
	if len(next.Changes) != 5 || len(next.Archive) != 3 || next.Archive[0].ID != "2026-10-13-add-csv-export-filters" {
		t.Errorf("Changes=%v Archive=%v", changeIDs(next.Changes), changeIDs(next.Archive))
	}

	// Reloading without further edits yields no events.
	again := mustLoad(t, root)
	if evs := Diff(next, again, now); len(evs) != 0 {
		t.Errorf("Diff of unchanged reload = %s", fmtEvents(strip(t, evs, now)))
	}
}

func TestDiffEditsAndRemovals(t *testing.T) {
	root := copyFixture(t)
	osDir := filepath.Join(root, "openspec")
	changes := filepath.Join(osDir, "changes")
	prev := mustLoad(t, root)

	// Content edit with no structural change.
	appendTo(t, filepath.Join(changes, "harden-session-security", "design.md"), "\n- Decided: confirm before revoking the current session.\n")
	// tasks.md rewritten without changing progress.
	appendTo(t, filepath.Join(changes, "add-csv-export-filters", "tasks.md"), "\nNotes: shipped behind a flag.\n")
	// A delta spec gains a requirement: reported as an artifact edit.
	appendTo(t, filepath.Join(changes, "add-task-search", "specs", "search", "spec.md"), newRequirement)
	// Metadata-only change: .openspec.yaml is not an artifact, so no event.
	touch(t, filepath.Join(changes, "add-due-dates", ".openspec.yaml"), "schema: spec-driven\ncreated: 2026-10-11\n")
	// A whole change deleted (not archived).
	if err := os.RemoveAll(filepath.Join(changes, "add-workspace-invites")); err != nil {
		t.Fatal(err)
	}
	// A main spec touched without changing its requirement count.
	bump(t, filepath.Join(osDir, "specs", "user-auth", "spec.md"))
	// A new main spec and a removed one.
	touch(t, filepath.Join(osDir, "specs", "search", "spec.md"),
		"# search Specification\n\n## Purpose\nSearch.\n## Requirements\n### Requirement: A\n### Requirement: B\n")
	if err := os.RemoveAll(filepath.Join(osDir, "specs", "notifications")); err != nil {
		t.Fatal(err)
	}

	next := mustLoad(t, root)
	now := time.Date(2026, 10, 13, 12, 0, 1, 0, time.UTC)
	assertEvents(t, Diff(prev, next, now), now, []ev{
		{EventEdited, ItemChange, "add-csv-export-filters", "tasks.md", "tasks.md"},
		{EventEdited, ItemChange, "add-task-search", "specs/search/spec.md", "specs/search/spec.md"},
		{EventEdited, ItemChange, "harden-session-security", "design.md", "design.md"},
		{EventRemoved, ItemChange, "add-workspace-invites", "", "change removed"},
		{EventCreated, ItemSpec, "search", "spec.md", "new spec, 2 requirements"},
		{EventEdited, ItemSpec, "user-auth", "spec.md", "spec.md"},
		{EventRemoved, ItemSpec, "notifications", "", "spec removed"},
	})
}

func TestDiffNilAndIdentical(t *testing.T) {
	s := mustLoad(t, fixtureDir)
	now := time.Now()
	if got := Diff(nil, s, now); got != nil {
		t.Errorf("Diff(nil, s) = %v, want nil", got)
	}
	if got := Diff(s, nil, now); got != nil {
		t.Errorf("Diff(s, nil) = %v, want nil", got)
	}
	if got := Diff(s, s, now); len(got) != 0 {
		t.Errorf("Diff(s, s) = %v, want none", got)
	}
	// openspec/ appearing from nothing: every change and spec is new.
	empty := mustLoad(t, t.TempDir())
	got := strip(t, Diff(empty, s, now), now)
	if len(got) != len(s.Changes)+len(s.Specs) {
		t.Errorf("Diff(empty, fixture) = %d events, want %d: %s", len(got), len(s.Changes)+len(s.Specs), fmtEvents(got))
	}
	if len(got) > 0 && got[0] != (ev{EventCreated, ItemChange, "add-csv-export-filters", "", "new change"}) {
		t.Errorf("first event = %+v", got[0])
	}
}

// Re-using a change name after an earlier archive is allowed by the CLI
// (`new change` only checks changes/<name>, and archive dirs differ by date).
func TestDiffArchiveWithReusedName(t *testing.T) {
	root := copyFixture(t)
	changes := filepath.Join(root, "openspec", "changes")
	touch(t, filepath.Join(changes, "archive", "2026-08-01-add-due-dates", "proposal.md"), "## Why\nFirst attempt.\n")
	prev := mustLoad(t, root)

	if err := os.Rename(filepath.Join(changes, "add-due-dates"), filepath.Join(changes, "archive", "2026-10-13-add-due-dates")); err != nil {
		t.Fatal(err)
	}
	next := mustLoad(t, root)
	now := time.Now()
	assertEvents(t, Diff(prev, next, now), now, []ev{
		{EventArchived, ItemArchive, "2026-10-13-add-due-dates", "", "archived"},
	})
}

func TestFingerprintChanged(t *testing.T) {
	root := copyFixture(t)
	dir := filepath.Join(root, "openspec")

	fp := Scan(dir)
	proposal := filepath.Join(dir, "changes", "add-due-dates", "proposal.md")
	yaml := filepath.Join(dir, "changes", "add-workspace-invites", ".openspec.yaml")
	for _, p := range []string{proposal, yaml, filepath.Join(dir, "config.yaml")} {
		if _, ok := fp[p]; !ok {
			t.Errorf("Scan missing file %s", p)
		}
	}
	for _, p := range []string{dir, filepath.Join(dir, "changes"), filepath.Join(dir, "changes", "archive")} {
		if _, ok := fp[p]; ok {
			t.Errorf("Scan includes directory %s", p)
		}
	}
	if got := fp.Changed(Scan(dir)); len(got) != 0 {
		t.Errorf("Changed with no edits = %v", got)
	}

	tests := []struct {
		name   string
		mutate func(t *testing.T) []string
	}{
		{"add", func(t *testing.T) []string {
			p := filepath.Join(dir, "changes", "add-due-dates", "design.md")
			writeFile(t, p, "## Context\n")
			return []string{p}
		}},
		{"modify content", func(t *testing.T) []string {
			appendTo(t, proposal, "\nMore.\n")
			return []string{proposal}
		}},
		{"modify mtime only", func(t *testing.T) []string {
			bump(t, yaml)
			return []string{yaml}
		}},
		{"modify size only", func(t *testing.T) []string {
			fi, _ := os.Stat(yaml)
			writeFile(t, yaml, "schema: x\n")
			if err := os.Chtimes(yaml, fi.ModTime(), fi.ModTime()); err != nil {
				t.Fatal(err)
			}
			return []string{yaml}
		}},
		{"delete", func(t *testing.T) []string {
			p := filepath.Join(dir, "specs", "notifications", "spec.md")
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
			return []string{p}
		}},
		{"rename dir (archive)", func(t *testing.T) []string {
			from := filepath.Join(dir, "changes", "add-task-search")
			to := filepath.Join(dir, "changes", "archive", "2026-10-13-add-task-search")
			if err := os.Rename(from, to); err != nil {
				t.Fatal(err)
			}
			var want []string
			for _, rel := range []string{".openspec.yaml", "proposal.md", "specs/search/spec.md", "specs/task-management/spec.md"} {
				want = append(want, filepath.Join(from, rel), filepath.Join(to, rel))
			}
			return want
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := Scan(dir)
			want := tt.mutate(t)
			got := before.Changed(Scan(dir))
			slices.Sort(got)
			slices.Sort(want)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Changed() = %v, want %v", got, want)
			}
		})
	}
}

func TestScanMissingDir(t *testing.T) {
	fp := Scan(filepath.Join(t.TempDir(), "nope"))
	if fp == nil || len(fp) != 0 {
		t.Errorf("Scan(missing) = %v, want empty non-nil", fp)
	}
	full := Scan(filepath.Join(fixtureDir, "openspec"))
	got := fp.Changed(full)
	if len(got) != len(full) || len(got) == 0 {
		t.Errorf("Changed(empty -> fixture) = %d paths, want %d", len(got), len(full))
	}
	if back := full.Changed(fp); len(back) != len(full) {
		t.Errorf("Changed(fixture -> empty) = %d paths, want %d", len(back), len(full))
	}
}
