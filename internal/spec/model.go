// Package spec reads an OpenSpec tree from disk into an immutable Snapshot.
package spec

import (
	"strings"
	"time"
)

// ArtifactKind classifies a file inside a change directory.
type ArtifactKind int

const (
	KindProposal ArtifactKind = iota
	KindDesign
	KindTasks
	KindDelta
	KindOther
)

func (k ArtifactKind) String() string {
	switch k {
	case KindProposal:
		return "proposal"
	case KindDesign:
		return "design"
	case KindTasks:
		return "tasks"
	case KindDelta:
		return "delta"
	default:
		return "other"
	}
}

// Artifact is a single markdown file belonging to a change.
type Artifact struct {
	Kind    ArtifactKind
	Rel     string // path relative to the change dir, e.g. "specs/auth/spec.md"
	Path    string // absolute path
	ModTime time.Time
	Content string
}

// Label is the short display name of the artifact.
func (a Artifact) Label() string {
	if a.Kind == KindDelta {
		return "Δ " + a.Capability()
	}
	return a.Rel
}

// Capability is the spec ID a delta artifact targets: specs/<id>/spec.md -> <id>.
func (a Artifact) Capability() string {
	return strings.TrimSuffix(strings.TrimPrefix(a.Rel, "specs/"), "/spec.md")
}

// TaskStats counts markdown checkboxes in tasks.md.
type TaskStats struct {
	Done  int
	Total int
}

// Complete reports whether there is at least one task and all are checked.
func (t TaskStats) Complete() bool { return t.Total > 0 && t.Done == t.Total }

// DeltaStats counts requirement operations across a change's delta specs.
type DeltaStats struct {
	Added    int
	Modified int
	Removed  int
	Renamed  int
}

// Total is the number of requirement operations.
func (d DeltaStats) Total() int { return d.Added + d.Modified + d.Removed + d.Renamed }

func (d DeltaStats) add(o DeltaStats) DeltaStats {
	return DeltaStats{
		Added:    d.Added + o.Added,
		Modified: d.Modified + o.Modified,
		Removed:  d.Removed + o.Removed,
		Renamed:  d.Renamed + o.Renamed,
	}
}

// Change is an active or archived change directory.
type Change struct {
	ID         string // directory name
	Dir        string
	Archived   bool
	ArchivedOn string // YYYY-MM-DD prefix for archived changes
	Name       string // ID without the archive date prefix
	Schema     string // from .openspec.yaml, if present
	Summary    string // first prose line under "## Why" in proposal.md
	Artifacts  []Artifact
	Tasks      TaskStats
	Deltas     DeltaStats
	ModTime    time.Time // newest artifact mtime
}

// Artifact returns the first artifact of the given kind.
func (c Change) Artifact(kind ArtifactKind) (Artifact, bool) {
	for _, a := range c.Artifacts {
		if a.Kind == kind {
			return a, true
		}
	}
	return Artifact{}, false
}

// Has reports whether the change has an artifact of the given kind.
func (c Change) Has(kind ArtifactKind) bool {
	_, ok := c.Artifact(kind)
	return ok
}

// Spec is a main capability spec under openspec/specs.
type Spec struct {
	ID           string // capability directory name
	Path         string // absolute path to spec.md
	Title        string
	Purpose      string
	Requirements int
	Scenarios    int
	ModTime      time.Time
	Content      string
}

// Snapshot is a point-in-time read of an OpenSpec tree.
type Snapshot struct {
	Root    string // repository root (parent of openspec/)
	Dir     string // the openspec/ directory
	Exists  bool   // whether Dir exists
	Changes []Change
	Archive []Change
	Specs   []Spec
	Config  string // path to config.yaml or project.md, if any
}

// FindChange looks up an active or archived change by ID.
func (s *Snapshot) FindChange(id string) (Change, bool) {
	for _, c := range s.Changes {
		if c.ID == id {
			return c, true
		}
	}
	for _, c := range s.Archive {
		if c.ID == id {
			return c, true
		}
	}
	return Change{}, false
}

// FindSpec looks up a main spec by capability ID.
func (s *Snapshot) FindSpec(id string) (Spec, bool) {
	for _, sp := range s.Specs {
		if sp.ID == id {
			return sp, true
		}
	}
	return Spec{}, false
}
