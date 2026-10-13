package spec

import (
	"fmt"
	"sort"
	"time"
)

// EventKind describes what happened to an item between two snapshots.
type EventKind int

const (
	EventCreated EventKind = iota
	EventEdited
	EventProgress
	EventArchived
	EventRemoved
)

// ItemKind identifies which list an event's item lives in.
type ItemKind int

const (
	ItemChange ItemKind = iota
	ItemSpec
	ItemArchive
)

// Event is one line in the activity feed.
type Event struct {
	Time   time.Time
	Kind   EventKind
	Item   ItemKind
	ID     string // change or spec ID
	Rel    string // artifact path within the change, or "spec.md"
	Detail string
}

// Diff describes what changed from prev to next, oldest-first within a call.
func Diff(prev, next *Snapshot, now time.Time) []Event {
	if prev == nil || next == nil {
		return nil
	}
	var events []Event
	emit := func(kind EventKind, item ItemKind, id, rel, format string, args ...any) {
		events = append(events, Event{
			Time: now, Kind: kind, Item: item, ID: id, Rel: rel,
			Detail: fmt.Sprintf(format, args...),
		})
	}

	prevChanges := indexChanges(prev.Changes)
	nextChanges := indexChanges(next.Changes)
	// Only archive entries that are new in this snapshot can explain a change
	// disappearing; older archives may reuse the same name.
	prevArchive := indexChanges(prev.Archive)
	archivedNames := map[string]string{}
	for _, c := range next.Archive {
		if _, old := prevArchive[c.ID]; !old {
			archivedNames[c.Name] = c.ID
		}
	}

	for _, id := range sortedKeys(nextChanges) {
		c := nextChanges[id]
		old, ok := prevChanges[id]
		if !ok {
			emit(EventCreated, ItemChange, id, "", "new change")
			continue
		}
		diffArtifacts(old, c, emit)
	}
	for _, id := range sortedKeys(prevChanges) {
		if _, ok := nextChanges[id]; ok {
			continue
		}
		if archID, ok := archivedNames[id]; ok {
			emit(EventArchived, ItemArchive, archID, "", "archived")
			continue
		}
		emit(EventRemoved, ItemChange, id, "", "change removed")
	}

	prevSpecs := map[string]Spec{}
	for _, s := range prev.Specs {
		prevSpecs[s.ID] = s
	}
	nextSpecs := map[string]bool{}
	for _, s := range next.Specs {
		nextSpecs[s.ID] = true
		old, ok := prevSpecs[s.ID]
		switch {
		case !ok:
			emit(EventCreated, ItemSpec, s.ID, "spec.md", "new spec, %d requirements", s.Requirements)
		case old.Requirements != s.Requirements:
			emit(EventEdited, ItemSpec, s.ID, "spec.md", "requirements %d → %d", old.Requirements, s.Requirements)
		case !old.ModTime.Equal(s.ModTime):
			emit(EventEdited, ItemSpec, s.ID, "spec.md", "spec.md")
		}
	}
	for _, s := range prev.Specs {
		if !nextSpecs[s.ID] {
			emit(EventRemoved, ItemSpec, s.ID, "", "spec removed")
		}
	}
	return events
}

func diffArtifacts(old, c Change, emit func(EventKind, ItemKind, string, string, string, ...any)) {
	oldArts := map[string]Artifact{}
	for _, a := range old.Artifacts {
		oldArts[a.Rel] = a
	}
	seen := map[string]bool{}
	for _, a := range c.Artifacts {
		seen[a.Rel] = true
		prevArt, existed := oldArts[a.Rel]
		switch {
		case !existed:
			emit(EventCreated, ItemChange, c.ID, a.Rel, "%s", a.Rel)
		case a.Kind == KindTasks && old.Tasks != c.Tasks:
			emit(EventProgress, ItemChange, c.ID, a.Rel, "tasks %d/%d → %d/%d",
				old.Tasks.Done, old.Tasks.Total, c.Tasks.Done, c.Tasks.Total)
		case !prevArt.ModTime.Equal(a.ModTime):
			emit(EventEdited, ItemChange, c.ID, a.Rel, "%s", a.Rel)
		}
	}
	for _, a := range old.Artifacts {
		if !seen[a.Rel] {
			emit(EventRemoved, ItemChange, c.ID, "", "%s", a.Rel)
		}
	}
}

func indexChanges(cs []Change) map[string]Change {
	m := make(map[string]Change, len(cs))
	for _, c := range cs {
		m[c.ID] = c
	}
	return m
}

func sortedKeys(m map[string]Change) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
