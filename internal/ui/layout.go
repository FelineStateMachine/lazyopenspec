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
	body     rect // everything between the header and footer
	panels   [numLists]rect
	main     rect
	portrait bool // side panels stacked above the main pane
}

const (
	headerRows = 1
	footerRows = 1
)

// PortraitMode mirrors lazygit's gui.portraitMode: whether to stack the side
// panels on top of the main pane instead of beside it.
type PortraitMode string

const (
	PortraitAuto   PortraitMode = "auto"
	PortraitAlways PortraitMode = "always"
	PortraitNever  PortraitMode = "never"
)

// lazygit's defaults for gui.portraitModeAutoMaxWidth / portraitModeAutoMinHeight.
const (
	portraitAutoMaxWidth  = 84
	portraitAutoMinHeight = 46
)

// ParsePortraitMode validates a --portrait flag value.
func ParsePortraitMode(s string) (PortraitMode, bool) {
	switch p := PortraitMode(s); p {
	case PortraitAuto, PortraitAlways, PortraitNever:
		return p, true
	}
	return "", false
}

func (m Model) portrait() bool {
	switch m.portraitMode {
	case PortraitAlways:
		return true
	case PortraitNever:
		return false
	default:
		return m.width <= portraitAutoMaxWidth && m.height >= portraitAutoMinHeight
	}
}

func (m Model) geometry() geometry {
	g := geometry{portrait: m.portrait()}
	bodyH := max(m.height-headerRows-footerRows, 0)
	g.body = rect{0, headerRows, m.width, bodyH}

	// Side section beside the main pane, or (portrait) above it taking a
	// third of the height, like lazygit's default sidePanelWidth of 0.3333.
	side := rect{0, headerRows, clamp(m.width*38/100, 34, 60), bodyH}
	if m.width < 80 {
		side.w = m.width / 2
	}
	g.main = rect{side.w, headerRows, m.width - side.w, bodyH}
	if g.portrait {
		side = rect{0, headerRows, m.width, bodyH / 3}
		g.main = rect{0, headerRows + side.h, m.width, bodyH - side.h}
	}

	y := side.y
	for i, h := range m.sideHeights(side.h) {
		g.panels[i] = rect{side.x, y, side.w, h}
		y += h
	}
	if m.zoom {
		g.main = g.body
	}
	return g
}

// sideFocus is the side panel lazygit would treat as current: the focused
// list, or the one driving the main pane while the main pane has focus.
func (m Model) sideFocus() panelID {
	if m.focus < numLists {
		return m.focus
	}
	return m.lastList
}

// sideHeights splits the side section between the list panels (heights
// include borders). With enough room each panel gets a share of the height;
// otherwise, as in lazygit, the focused panel takes the space and the others
// squash to a single row, or just their title when space is very tight.
func (m Model) sideHeights(total int) [numLists]int {
	// lazygit's thresholds (28 and 21 rows for five panels), scaled to four.
	const (
		minNormal     = 28 * numLists / 5
		minTallSquash = 21 * numLists / 5
		squashedTall  = 3 // border, selected row, border
		squashedShort = 1 // title only
	)
	var heights [numLists]int
	if total >= minNormal {
		want := [numLists]int{
			max(m.count(panelChanges), 1),
			max(m.count(panelSpecs), 1),
			max(m.count(panelArchive), 1),
			max(m.count(panelActivity), 1),
		}
		share := [numLists]int{35, 25, 15, 25}
		for i, rows := range allocate(want, share, total-2*numLists, m.focus) {
			heights[i] = rows + 2
		}
		return heights
	}
	squashed := squashedShort
	if total >= minTallSquash {
		squashed = squashedTall
	}
	focused := m.sideFocus()
	for i := range heights {
		heights[i] = squashed
	}
	heights[focused] = max(total-squashed*(numLists-1), 0)
	return heights
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
	if w < 2 || h < 1 {
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
	if h == 1 {
		return sb.String() // squashed panel: title only
	}
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
