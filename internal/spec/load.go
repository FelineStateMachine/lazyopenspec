package spec

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// DirName is the name of the OpenSpec directory at a repository root.
const DirName = "openspec"

var archivePrefixRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-(.+)$`)

// FindRoot walks up from start looking for a directory that contains
// openspec/. It returns start itself when no ancestor has one.
func FindRoot(start string) (string, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	// Resolve symlinks so paths match what git reports (e.g. /tmp on macOS).
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		abs = real
	}
	for dir := abs; ; {
		if fi, err := os.Stat(filepath.Join(dir, DirName)); err == nil && fi.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return abs, nil
		}
		dir = parent
	}
}

// Load reads the OpenSpec tree rooted at root/openspec.
func Load(root string) (*Snapshot, error) {
	s := &Snapshot{Root: root, Dir: filepath.Join(root, DirName)}
	fi, err := os.Stat(s.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, errors.New(s.Dir + " is not a directory")
	}
	s.Exists = true

	for _, name := range []string{"config.yaml", "project.md"} {
		if p := filepath.Join(s.Dir, name); fileExists(p) {
			s.Config = p
			break
		}
	}

	s.Specs, err = loadSpecs(filepath.Join(s.Dir, "specs"))
	if err != nil {
		return nil, err
	}

	changesDir := filepath.Join(s.Dir, "changes")
	s.Changes, err = loadChanges(changesDir, false)
	if err != nil {
		return nil, err
	}
	s.Archive, err = loadChanges(filepath.Join(changesDir, "archive"), true)
	if err != nil {
		return nil, err
	}
	sort.Slice(s.Archive, func(i, j int) bool { return s.Archive[i].ID > s.Archive[j].ID })
	return s, nil
}

func loadSpecs(dir string) ([]Spec, error) {
	var specs []Spec
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() || d.Name() != "spec.md" {
			return nil
		}
		rel, err := filepath.Rel(dir, filepath.Dir(path))
		if err != nil {
			return err
		}
		src, mod, err := readFile(path)
		if err != nil {
			return nil // vanished mid-walk; the next scan will catch up
		}
		meta := ParseSpec(src)
		specs = append(specs, Spec{
			ID:           filepath.ToSlash(rel),
			Path:         path,
			Title:        meta.Title,
			Purpose:      meta.Purpose,
			Requirements: meta.Requirements,
			Scenarios:    meta.Scenarios,
			ModTime:      mod,
			Content:      src,
		})
		return nil
	})
	sort.Slice(specs, func(i, j int) bool { return specs[i].ID < specs[j].ID })
	return specs, err
}

func loadChanges(dir string, archived bool) ([]Change, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var changes []Change
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if !archived && e.Name() == "archive" {
			continue
		}
		c, err := loadChange(filepath.Join(dir, e.Name()), archived)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].ID < changes[j].ID })
	return changes, nil
}

func loadChange(dir string, archived bool) (Change, error) {
	c := Change{ID: filepath.Base(dir), Dir: dir, Archived: archived}
	c.Name = c.ID
	if archived {
		if m := archivePrefixRe.FindStringSubmatch(c.ID); m != nil {
			c.ArchivedOn, c.Name = m[1], m[2]
		}
	}
	if fi, err := os.Stat(dir); err == nil {
		c.ModTime = fi.ModTime()
	}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		src, mod, err := readFile(path)
		if err != nil {
			return nil
		}
		if mod.After(c.ModTime) {
			c.ModTime = mod
		}
		if rel == ".openspec.yaml" {
			c.Schema = ParseSchema(src)
			return nil
		}
		if !strings.HasSuffix(rel, ".md") {
			return nil
		}
		a := Artifact{Kind: classify(rel), Rel: rel, Path: path, ModTime: mod, Content: src}
		switch a.Kind {
		case KindProposal:
			c.Summary = ParseSummary(src)
		case KindTasks:
			c.Tasks = ParseTasks(src)
		case KindDelta:
			c.Deltas = c.Deltas.add(ParseDelta(src))
		}
		c.Artifacts = append(c.Artifacts, a)
		return nil
	})
	sort.SliceStable(c.Artifacts, func(i, j int) bool {
		a, b := c.Artifacts[i], c.Artifacts[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Rel < b.Rel
	})
	return c, err
}

func classify(rel string) ArtifactKind {
	switch rel {
	case "proposal.md":
		return KindProposal
	case "design.md":
		return KindDesign
	case "tasks.md":
		return KindTasks
	}
	// The CLI reads deltas only from specs/<capability>/spec.md.
	if dir, ok := strings.CutPrefix(rel, "specs/"); ok && strings.Count(dir, "/") == 1 && strings.HasSuffix(dir, "/spec.md") {
		return KindDelta
	}
	return KindOther
}

func readFile(path string) (string, time.Time, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", time.Time{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", time.Time{}, err
	}
	return string(b), fi.ModTime(), nil
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}
