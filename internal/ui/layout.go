package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type rect struct{ x, y, w, h int }

func (r rect) contains(x, y int) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

// geometry is the screen layout for one frame. Panel rects include borders.
type geometry struct {
	panels [numLists]rect
	main   rect
}

const (
	headerRows = 1
	footerRows = 1
)

func (m Model) geometry() geometry {
	var g geometry
	bodyH := max(m.height-headerRows-footerRows, 0)
	leftW := clamp(m.width*38/100, 34, 60)
	if m.width < 80 {
		leftW = m.width / 2
	}
	if m.zoom {
		leftW = 0
	}

	// Content rows available to the four list panels after their borders.
	avail := max(bodyH-2*numLists, numLists)
	want := [numLists]int{
		max(m.count(panelChanges), 1),
		max(m.count(panelSpecs), 1),
		max(m.count(panelArchive), 1),
		max(m.count(panelActivity), 1),
	}
	share := [numLists]int{35, 25, 15, 25}
	rows := allocate(want, share, avail, m.focus)

	y := headerRows
	for i := range numLists {
		h := rows[i] + 2
		g.panels[i] = rect{0, y, leftW, h}
		y += h
	}
	g.main = rect{leftW, headerRows, m.width - leftW, bodyH}
	return g
}

// allocate splits total rows between panels: each gets its percentage share,
// unused share flows to panels that want more, and any slack goes to the
// focused panel.
func allocate(want, share [numLists]int, total int, focus panelID) [numLists]int {
	var rows [numLists]int
	left := total
	for i := range rows {
		rows[i] = min(max(total*share[i]/100, 1), want[i])
		left -= rows[i]
	}
	for left > 0 {
		best, gap := -1, 0
		for i := range rows {
			if g := want[i] - rows[i]; g > gap {
				best, gap = i, g
			}
		}
		if best < 0 {
			break
		}
		rows[best]++
		left--
	}
	if left > 0 && int(focus) < numLists {
		rows[focus] += left
	} else if left > 0 {
		rows[panelActivity] += left
	}
	return rows
}

// box draws a rounded border around body with a title embedded in the top
// edge and an optional footer in the bottom edge, lazygit-style.
func box(t theme, title, footer, body string, w, h int, focused bool) string {
	if w < 2 || h < 2 {
		return ""
	}
	border := t.subtle
	if focused {
		border = lipgloss.NewStyle().Foreground(t.accent)
	}
	b := lipgloss.RoundedBorder()
	innerW := w - 2

	edge := func(left, right, label string) string {
		label = ansi.Truncate(label, max(innerW-2, 0), "…")
		fill := innerW - lipgloss.Width(label)
		if label != "" {
			fill -= 2
			label = border.Render(b.Top) + label + border.Render(b.Top)
		}
		return border.Render(left) + label + border.Render(strings.Repeat(b.Top, max(fill, 0))) + border.Render(right)
	}

	var sb strings.Builder
	sb.WriteString(edge(b.TopLeft, b.TopRight, title))
	lines := strings.Split(body, "\n")
	for i := range h - 2 {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		sb.WriteString("\n")
		sb.WriteString(border.Render(b.Left))
		sb.WriteString(fit(line, innerW))
		sb.WriteString(border.Render(b.Right))
	}
	sb.WriteString("\n")
	bottom := edge(b.BottomLeft, b.BottomRight, footer)
	sb.WriteString(bottom)
	return sb.String()
}

// fit truncates or right-pads s to exactly w cells.
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = ansi.Truncate(s, w, "…")
	if pad := w - lipgloss.Width(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

func clamp(v, lo, hi int) int {
	return max(lo, min(v, hi))
}
