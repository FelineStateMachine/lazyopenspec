package spec

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

var fixtureDir = filepath.Join("..", "..", "testdata", "fixture")

// copyFixture copies testdata/fixture into a fresh temp dir and returns the
// repo root (the directory containing openspec/).
func copyFixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "repo")
	if err := os.CopyFS(root, os.DirFS(fixtureDir)); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	return root
}

func mustLoad(t *testing.T, root string) *Snapshot {
	t.Helper()
	s, err := Load(root)
	if err != nil {
		t.Fatalf("Load(%s): %v", root, err)
	}
	return s
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func changeIDs(cs []Change) []string {
	var ids []string
	for _, c := range cs {
		ids = append(ids, c.ID)
	}
	return ids
}

func TestLoadFixtureCLIParity(t *testing.T) {
	s := mustLoad(t, fixtureDir)

	if !s.Exists {
		t.Fatal("Exists = false, want true")
	}
	if s.Root != fixtureDir || s.Dir != filepath.Join(fixtureDir, "openspec") {
		t.Errorf("Root, Dir = %q, %q", s.Root, s.Dir)
	}
	if want := filepath.Join(fixtureDir, "openspec", "config.yaml"); s.Config != want {
		t.Errorf("Config = %q, want %q", s.Config, want)
	}

	// `openspec list --json`: 5 active changes (add-workspace-invites has only
	// .openspec.yaml but is still a change), sorted by ID here.
	wantChanges := []string{"add-csv-export-filters", "add-due-dates", "add-task-search", "add-workspace-invites", "harden-session-security"}
	if got := changeIDs(s.Changes); !reflect.DeepEqual(got, wantChanges) {
		t.Fatalf("Changes = %v, want %v", got, wantChanges)
	}
	wantTasks := map[string]TaskStats{
		"add-csv-export-filters":  {Done: 8, Total: 8},
		"add-due-dates":           {},
		"add-task-search":         {},
		"add-workspace-invites":   {},
		"harden-session-security": {Done: 4, Total: 10},
	}
	for _, c := range s.Changes {
		if c.Tasks != wantTasks[c.ID] {
			t.Errorf("%s: Tasks = %+v, want %+v", c.ID, c.Tasks, wantTasks[c.ID])
		}
		if c.Archived || c.ArchivedOn != "" || c.Name != c.ID {
			t.Errorf("%s: Archived=%v ArchivedOn=%q Name=%q for an active change", c.ID, c.Archived, c.ArchivedOn, c.Name)
		}
		if c.Schema != "spec-driven" {
			t.Errorf("%s: Schema = %q, want spec-driven", c.ID, c.Schema)
		}
		if c.Dir != filepath.Join(s.Dir, "changes", c.ID) {
			t.Errorf("%s: Dir = %q", c.ID, c.Dir)
		}
	}

	// `openspec show <change> --json` deltaCount.
	wantDeltas := map[string]DeltaStats{
		"add-csv-export-filters":  {Added: 1, Modified: 1},
		"add-task-search":         {Added: 2, Modified: 1},
		"harden-session-security": {Added: 1, Modified: 1, Removed: 1, Renamed: 1},
	}
	for _, c := range s.Changes {
		if c.Deltas != wantDeltas[c.ID] {
			t.Errorf("%s: Deltas = %+v, want %+v", c.ID, c.Deltas, wantDeltas[c.ID])
		}
	}

	// `openspec spec list --json` requirementCount, sorted by ID.
	type specCount struct {
		ID                      string
		Requirements, Scenarios int
	}
	var gotSpecs []specCount
	for _, sp := range s.Specs {
		gotSpecs = append(gotSpecs, specCount{sp.ID, sp.Requirements, sp.Scenarios})
	}
	wantSpecs := []specCount{{"data-export", 3, 4}, {"notifications", 2, 4}, {"task-management", 4, 10}, {"user-auth", 4, 7}}
	if !reflect.DeepEqual(gotSpecs, wantSpecs) {
		t.Errorf("Specs = %+v, want %+v", gotSpecs, wantSpecs)
	}

	// Archive: sorted newest-first with the date prefix split off.
	type arch struct{ ID, On, Name string }
	var gotArch []arch
	for _, c := range s.Archive {
		gotArch = append(gotArch, arch{c.ID, c.ArchivedOn, c.Name})
		if !c.Archived {
			t.Errorf("%s: Archived = false", c.ID)
		}
		if c.Dir != filepath.Join(s.Dir, "changes", "archive", c.ID) {
			t.Errorf("%s: Dir = %q", c.ID, c.Dir)
		}
	}
	wantArch := []arch{
		{"2026-09-23-add-email-notifications", "2026-09-23", "add-email-notifications"},
		{"2026-08-19-add-user-auth", "2026-08-19", "add-user-auth"},
	}
	if !reflect.DeepEqual(gotArch, wantArch) {
		t.Errorf("Archive = %+v, want %+v", gotArch, wantArch)
	}
	if got := s.Archive[0].Tasks; got != (TaskStats{Done: 7, Total: 7}) {
		t.Errorf("add-email-notifications Tasks = %+v, want 7/7", got)
	}
	if got := s.Archive[1].Tasks; got != (TaskStats{Done: 10, Total: 10}) {
		t.Errorf("add-user-auth Tasks = %+v, want 10/10", got)
	}
	if got := s.Archive[0].Deltas; got != (DeltaStats{Added: 2, Modified: 1}) {
		t.Errorf("add-email-notifications Deltas = %+v", got)
	}
}

func TestLoadFixtureArtifacts(t *testing.T) {
	s := mustLoad(t, fixtureDir)

	type art struct {
		Kind ArtifactKind
		Rel  string
	}
	want := map[string][]art{
		"add-csv-export-filters": {{KindProposal, "proposal.md"}, {KindTasks, "tasks.md"}, {KindDelta, "specs/data-export/spec.md"}},
		"add-due-dates":          {{KindProposal, "proposal.md"}, {KindOther, "README.md"}},
		"add-task-search": {{KindProposal, "proposal.md"},
			{KindDelta, "specs/search/spec.md"}, {KindDelta, "specs/task-management/spec.md"}},
		"add-workspace-invites": nil, // .openspec.yaml is metadata, not an artifact
		"harden-session-security": {{KindProposal, "proposal.md"}, {KindDesign, "design.md"},
			{KindTasks, "tasks.md"}, {KindDelta, "specs/user-auth/spec.md"}},
		"2026-08-19-add-user-auth": {{KindProposal, "proposal.md"}, {KindDesign, "design.md"},
			{KindTasks, "tasks.md"}, {KindDelta, "specs/user-auth/spec.md"}},
		"2026-09-23-add-email-notifications": {{KindProposal, "proposal.md"}, {KindTasks, "tasks.md"},
			{KindDelta, "specs/notifications/spec.md"}, {KindDelta, "specs/task-management/spec.md"}},
	}
	for _, c := range append(append([]Change{}, s.Changes...), s.Archive...) {
		var got []art
		for _, a := range c.Artifacts {
			got = append(got, art{a.Kind, a.Rel})
			if a.Path != filepath.Join(c.Dir, filepath.FromSlash(a.Rel)) {
				t.Errorf("%s/%s: Path = %q", c.ID, a.Rel, a.Path)
			}
			if a.Content == "" || a.ModTime.IsZero() {
				t.Errorf("%s/%s: empty Content or zero ModTime", c.ID, a.Rel)
			}
			if a.ModTime.After(c.ModTime) {
				t.Errorf("%s/%s: artifact ModTime %v after change ModTime %v", c.ID, a.Rel, a.ModTime, c.ModTime)
			}
		}
		if !reflect.DeepEqual(got, want[c.ID]) {
			t.Errorf("%s artifacts = %v, want %v", c.ID, got, want[c.ID])
		}
	}

	harden, ok := s.FindChange("harden-session-security")
	if !ok {
		t.Fatal("FindChange(harden-session-security) not found")
	}
	if !harden.Has(KindDesign) || !harden.Has(KindDelta) {
		t.Error("harden-session-security should have design and delta artifacts")
	}
	if d, _ := harden.Artifact(KindDelta); d.Label() != "Δ user-auth" || d.Capability() != "user-auth" {
		t.Errorf("delta Label, Capability = %q, %q", d.Label(), d.Capability())
	}
	if p, _ := harden.Artifact(KindProposal); p.Label() != "proposal.md" {
		t.Errorf("proposal Label = %q", p.Label())
	}
	csv, _ := s.FindChange("add-csv-export-filters")
	if csv.Has(KindDesign) {
		t.Error("add-csv-export-filters has no design.md")
	}
	if _, ok := csv.Artifact(KindDesign); ok {
		t.Error("Artifact(KindDesign) ok = true for add-csv-export-filters")
	}
	if _, ok := s.FindChange("2026-08-19-add-user-auth"); !ok {
		t.Error("FindChange should find archived changes by ID")
	}
	if _, ok := s.FindChange("nope"); ok {
		t.Error("FindChange(nope) ok = true")
	}

	kinds := map[ArtifactKind]string{KindProposal: "proposal", KindDesign: "design", KindTasks: "tasks", KindDelta: "delta", KindOther: "other"}
	for k, want := range kinds {
		if k.String() != want {
			t.Errorf("ArtifactKind(%d).String() = %q, want %q", k, k.String(), want)
		}
	}
}

func TestLoadFixtureSummariesAndSpecs(t *testing.T) {
	s := mustLoad(t, fixtureDir)

	wantSummary := map[string]string{
		"add-csv-export-filters":  "Project owners export to CSV mainly",
		"add-due-dates":           "Teams use Tally for release checklists",
		"add-task-search":         "Projects with more than a few hundred tasks", // H1 "Change:" title is skipped
		"add-workspace-invites":   "",
		"harden-session-security": "A September penetration test flagged",
	}
	for _, c := range s.Changes {
		want := wantSummary[c.ID]
		if (want == "") != (c.Summary == "") || !strings.HasPrefix(c.Summary, want) {
			t.Errorf("%s: Summary = %q, want prefix %q", c.ID, c.Summary, want)
		}
	}

	notif, ok := s.FindSpec("notifications")
	if !ok {
		t.Fatal("FindSpec(notifications) not found")
	}
	if notif.Title != "notifications Specification" ||
		notif.Purpose != "TBD - created by archiving change add-email-notifications. Update Purpose after archive." {
		t.Errorf("notifications Title, Purpose = %q, %q", notif.Title, notif.Purpose)
	}
	export, _ := s.FindSpec("data-export")
	if export.Title != "Data Export Specification" || !strings.HasPrefix(export.Purpose, "Lets project owners take their data") {
		t.Errorf("data-export Title, Purpose = %q, %q", export.Title, export.Purpose)
	}
	if export.Path != filepath.Join(s.Dir, "specs", "data-export", "spec.md") || export.Content == "" || export.ModTime.IsZero() {
		t.Errorf("data-export Path/Content/ModTime not populated: %q", export.Path)
	}
	if _, ok := s.FindSpec("search"); ok {
		t.Error("FindSpec(search) ok = true; search only exists as a delta")
	}
}

func TestLoadChangeModTimeIsNewestFile(t *testing.T) {
	root := copyFixture(t)
	before, _ := mustLoad(t, root).FindChange("add-task-search")
	newest := before.ModTime.Add(time.Hour).Truncate(time.Second)
	nested := filepath.Join(before.Dir, "specs", "search", "spec.md")
	if err := os.Chtimes(nested, newest, newest); err != nil {
		t.Fatal(err)
	}
	c, _ := mustLoad(t, root).FindChange("add-task-search")
	if !c.ModTime.Equal(newest) {
		t.Errorf("ModTime = %v, want nested delta mtime %v", c.ModTime, newest)
	}
}

func TestLoadWithoutOpenspecDir(t *testing.T) {
	root := t.TempDir()
	s, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Exists || s.Root != root || s.Dir != filepath.Join(root, "openspec") {
		t.Errorf("got Exists=%v Root=%q Dir=%q", s.Exists, s.Root, s.Dir)
	}
	if len(s.Changes)+len(s.Archive)+len(s.Specs) != 0 || s.Config != "" {
		t.Errorf("expected an empty snapshot, got %+v", s)
	}
}

func TestLoadOpenspecIsAFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "openspec"), "not a dir")
	if _, err := Load(root); err == nil {
		t.Error("Load succeeded when openspec is a file")
	}
}

// A fresh clone has no empty dirs: openspec/ may lack specs/, changes/ and
// changes/archive/.
func TestLoadMissingSubdirs(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "openspec"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := mustLoad(t, root)
	if !s.Exists || len(s.Changes)+len(s.Archive)+len(s.Specs) != 0 || s.Config != "" {
		t.Errorf("got %+v", s)
	}

	// Legacy project.md is used when config.yaml is absent; changes without
	// an archive dir still load.
	writeFile(t, filepath.Join(root, "openspec", "project.md"), "# Project\n")
	writeFile(t, filepath.Join(root, "openspec", "changes", "solo", "proposal.md"), "## Why\nSolo.\n")
	s = mustLoad(t, root)
	if s.Config != filepath.Join(root, "openspec", "project.md") {
		t.Errorf("Config = %q, want project.md", s.Config)
	}
	if got := changeIDs(s.Changes); !reflect.DeepEqual(got, []string{"solo"}) || s.Changes[0].Schema != "" {
		t.Errorf("Changes = %v (schema %q)", got, s.Changes[0].Schema)
	}
}

func TestLoadSkipsDotDirsAndArchiveWithoutDatePrefix(t *testing.T) {
	root := t.TempDir()
	changes := filepath.Join(root, "openspec", "changes")
	writeFile(t, filepath.Join(changes, ".scratch", "proposal.md"), "## Why\nx\n")
	writeFile(t, filepath.Join(changes, "stray.md"), "not a change dir\n")
	writeFile(t, filepath.Join(changes, "archive", "manually-moved", "proposal.md"), "## Why\nx\n")
	s := mustLoad(t, root)
	if len(s.Changes) != 0 {
		t.Errorf("Changes = %v, want none (dot-dirs and files skipped)", changeIDs(s.Changes))
	}
	if len(s.Archive) != 1 || s.Archive[0].Name != "manually-moved" || s.Archive[0].ArchivedOn != "" {
		t.Errorf("Archive = %+v", s.Archive)
	}
}

// The CLI only reads deltas from specs/<capability>/spec.md (change-parser.js
// parseDeltaSpecs iterates directories under specs/). Verified:
// `openspec show <change> --json` reports deltaCount 0 for this layout.
func TestLoadDeltaNeedsCapabilityDir(t *testing.T) {
	t.Log("was BUG: load.go:217 classify treats \"specs/spec.md\" as KindDelta because it has the \"specs/\" prefix " +
		"and \"/spec.md\" suffix; the artifact gets Capability \"spec.md\" (label \"Δ spec.md\") and its " +
		"requirements are counted in Deltas, while the CLI ignores files without a capability dir")
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "openspec", "changes", "loose", "specs", "spec.md"),
		"## ADDED Requirements\n### Requirement: Loose\nThe system SHALL x.\n")
	c := mustLoad(t, root).Changes[0]
	if a := c.Artifacts[0]; a.Kind == KindDelta {
		t.Errorf("specs/spec.md classified as delta (Capability %q)", a.Capability())
	}
	if c.Deltas.Total() != 0 {
		t.Errorf("Deltas = %+v, want none", c.Deltas)
	}
}

func TestFindRoot(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir()) // /var -> /private/var on macOS
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "repo")
	if err := os.CopyFS(root, os.DirFS(fixtureDir)); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, "src", "pkg", "deep")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(base, "elsewhere", "nested")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, start, want string
	}{
		{"root itself", root, root},
		{"nested subdir", deep, root},
		{"inside openspec/changes/<id>", filepath.Join(root, "openspec", "changes", "add-due-dates"), root},
		{"relative path with dots", filepath.Join(deep, "..", ".."), root},
		{"no openspec anywhere returns start", outside, outside},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindRoot(tt.start)
			if err != nil {
				t.Fatalf("FindRoot: %v", err)
			}
			if got != tt.want {
				t.Errorf("FindRoot(%q) = %q, want %q", tt.start, got, tt.want)
			}
		})
	}

	t.Run("relative to cwd", func(t *testing.T) {
		t.Chdir(deep)
		got, err := FindRoot(".")
		if err != nil || got != root {
			t.Errorf("FindRoot(.) = %q, %v; want %q", got, err, root)
		}
	})

	t.Run("symlinked start resolves to real path", func(t *testing.T) {
		link := filepath.Join(base, "link")
		if err := os.Symlink(deep, link); err != nil {
			t.Skip("symlinks unsupported:", err)
		}
		if got, err := FindRoot(link); err != nil || got != root {
			t.Errorf("FindRoot(link) = %q, %v; want %q", got, err, root)
		}
	})
}
