package ui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/dami/lazyopenspec/internal/spec"
)

func fixtureModel(t *testing.T, w, h int) Model {
	t.Helper()
	root, err := filepath.Abs("../../testdata/fixture")
	if err != nil {
		t.Fatal(err)
	}
	snap, err := spec.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	m := New(Options{Root: root, Interval: time.Hour})
	return step(t, step(t, m, tea.WindowSizeMsg{Width: w, Height: h}), loadedMsg{snap: snap})
}

func step(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	next, _ := m.Update(msg)
	return next.(Model)
}

func key(s string) tea.KeyPressMsg {
	switch s {
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	}
	return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
}

// Every frame must be exactly the terminal size, at any size, in any panel.
func TestViewFitsTerminal(t *testing.T) {
	for _, size := range [][2]int{{160, 45}, {80, 24}, {40, 10}, {20, 5}, {3, 2}, {1, 1}} {
		m := fixtureModel(t, size[0], size[1])
		for _, k := range []string{"", "tab", "j", "]", "z", "?", "x", "2", "enter", "d"} {
			if k != "" {
				m = step(t, m, key(k))
			}
			lines := strings.Split(m.View().Content, "\n")
			if len(lines) != size[1] {
				t.Fatalf("%dx%d after %q: %d lines, want %d", size[0], size[1], k, len(lines), size[1])
			}
			for i, l := range lines {
				if w := ansi.StringWidth(l); w > size[0] {
					t.Fatalf("%dx%d after %q: line %d is %d wide", size[0], size[1], k, i, w)
				}
			}
		}
	}
}

func TestSelectionFollowsIDAcrossReloads(t *testing.T) {
	m := fixtureModel(t, 120, 40)
	m = step(t, m, key("j")) // add-due-dates
	if got := m.selected[panelChanges]; got != "add-due-dates" {
		t.Fatalf("selected %q", got)
	}
	// A change sorting before it appears; the cursor should stay on add-due-dates.
	next := *m.snap
	next.Changes = append([]spec.Change{{ID: "aaa-new", Name: "aaa-new"}}, next.Changes...)
	m = step(t, m, loadedMsg{snap: &next})
	if got := m.idAt(panelChanges, m.cursors[panelChanges]); got != "add-due-dates" {
		t.Errorf("after reload cursor on %q, want add-due-dates", got)
	}
	if len(m.events) != 1 || m.events[0].ID != "aaa-new" {
		t.Errorf("events = %+v", m.events)
	}
}

func TestResizeKeepsCursorVisible(t *testing.T) {
	m := fixtureModel(t, 120, 60)
	for range 4 {
		m = step(t, m, key("j"))
	}
	m = step(t, m, tea.WindowSizeMsg{Width: 120, Height: 16})
	h := m.geometry().panels[panelChanges].h - 2
	c, off := m.cursors[panelChanges], m.offsets[panelChanges]
	if c < off || c >= off+h {
		t.Errorf("cursor %d not within visible rows [%d,%d)", c, off, off+h)
	}
}

func TestAllocateFillsTotal(t *testing.T) {
	share := [numLists]int{35, 25, 15, 25}
	for _, total := range []int{4, 10, 37, 100} {
		rows := allocate([numLists]int{50, 3, 2, 1}, share, total, panelChanges)
		sum := 0
		for _, r := range rows {
			if r < 1 {
				t.Errorf("total %d: panel got %d rows", total, r)
			}
			sum += r
		}
		if sum != total {
			t.Errorf("total %d: allocated %d (%v)", total, sum, rows)
		}
	}
}
