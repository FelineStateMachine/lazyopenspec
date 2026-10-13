package ui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/FelineStateMachine/lazyopenspec/internal/spec"
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
	for _, size := range [][2]int{{160, 45}, {80, 24}, {84, 46}, {60, 90}, {40, 10}, {20, 5}, {3, 2}, {1, 1}} {
		m := fixtureModel(t, size[0], size[1])
		for _, k := range []string{"", "tab", "j", "]", "z", "?", "x", "2", "enter", "d", "v", "tab", "v"} {
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

func TestPortraitAuto(t *testing.T) {
	// lazygit's defaults: portrait when width <= 84 and height >= 46.
	tests := []struct {
		w, h int
		want bool
	}{
		{84, 46, true},
		{60, 120, true},
		{85, 46, false},
		{84, 45, false},
		{160, 45, false},
	}
	for _, tt := range tests {
		if got := fixtureModel(t, tt.w, tt.h).geometry().portrait; got != tt.want {
			t.Errorf("%dx%d: portrait = %v, want %v", tt.w, tt.h, got, tt.want)
		}
	}
	m := fixtureModel(t, 160, 45)
	m.portraitMode = PortraitAlways
	if !m.geometry().portrait {
		t.Error("always: not portrait at 160x45")
	}
	m = fixtureModel(t, 60, 120)
	m.portraitMode = PortraitNever
	if m.geometry().portrait {
		t.Error("never: portrait at 60x120")
	}
}

func TestPortraitStacksPanelsAboveMain(t *testing.T) {
	m := fixtureModel(t, 80, 62) // body 60: side 20, main 40
	g := m.geometry()
	if g.main.x != 0 || g.main.w != 80 || g.main.y != headerRows+20 || g.main.h != 40 {
		t.Errorf("main = %+v, want full width below a 20-row side section", g.main)
	}
	y := headerRows
	for p, r := range g.panels {
		if r.x != 0 || r.w != 80 || r.y != y {
			t.Errorf("panel %d = %+v, want full width at y=%d", p, r, y)
		}
		y += r.h
	}
	if y != g.main.y {
		t.Errorf("panels end at y=%d, main starts at %d", y, g.main.y)
	}
}

func TestSquashedSidePanels(t *testing.T) {
	// A 20-row side section is below the normal-layout threshold (22) but
	// tall enough (>= 16) for 3-row squashed panels, like lazygit.
	m := fixtureModel(t, 80, 62)
	if got := m.sideHeights(20); got != [numLists]int{11, 3, 3, 3} {
		t.Errorf("focused Changes: heights %v", got)
	}
	m = step(t, m, key("2"))
	if got := m.sideHeights(20); got != [numLists]int{3, 11, 3, 3} {
		t.Errorf("focused Specs: heights %v", got)
	}
	m = step(t, m, key("0")) // main focused: Specs still drives it and stays open
	if got := m.sideHeights(20); got != [numLists]int{3, 11, 3, 3} {
		t.Errorf("main focused: heights %v", got)
	}
	if got := m.sideHeights(12); got != [numLists]int{1, 9, 1, 1} {
		t.Errorf("short section: heights %v, want title-only squashed panels", got)
	}
	if got := m.sideHeights(40); got[0] < 4 || got[1] < 4 {
		t.Errorf("tall section should not squash: %v", got)
	}
}
