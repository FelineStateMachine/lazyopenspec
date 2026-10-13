package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/FelineStateMachine/lazyopenspec/internal/spec"
)

func (m Model) render() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	g := m.geometry()
	var cols []string
	if m.showHelp {
		cols = append(cols, box(m.theme, m.theme.title.Render("Keys"), "", m.helpText(m.width-2), m.width, g.main.h, true))
	} else if !m.zoom {
		panels := make([]string, 0, numLists)
		for p := range panelID(numLists) {
			panels = append(panels, m.renderPanel(p, g.panels[p]))
		}
		cols = append(cols, lipgloss.JoinVertical(lipgloss.Left, panels...))
	}
	if !m.showHelp {
		cols = append(cols, m.renderMain(g.main))
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, cols...)

	lines := strings.Split(m.renderHeader()+"\n"+body, "\n")
	if len(lines) > m.height-footerRows {
		lines = lines[:max(m.height-footerRows, 0)]
	}
	for len(lines) < m.height-footerRows {
		lines = append(lines, "")
	}
	lines = append(lines, m.renderFooter())
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, m.width, "")
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderHeader() string {
	th := m.theme
	sep := th.subtle.Render("  │  ")
	left := []string{th.title.Render(" lazyopenspec"), th.bold.Render(tildePath(m.opts.Root))}
	if m.git.Branch != "" {
		branch := " " + m.git.Branch
		if n := len(m.git.Dirty); n > 0 {
			branch += th.modSt.Render(fmt.Sprintf(" +%d uncommitted", n))
		}
		left = append(left, th.renSt.Render(branch))
	}
	if m.snap != nil && m.snap.Exists {
		active, done := 0, 0
		for _, c := range m.snap.Changes {
			active++
			if c.Tasks.Complete() {
				done++
			}
		}
		stats := fmt.Sprintf("%d active", active)
		if done > 0 {
			stats += fmt.Sprintf(", %d ready to archive", done)
		}
		left = append(left, th.muted.Render(stats))
	}

	var right []string
	if m.follow {
		right = append(right, th.key.Render("FOLLOW"))
	}
	live := th.subtle.Render("●")
	switch {
	case m.loading:
		live = th.modSt.Render("●")
	case !m.lastSeen.IsZero() && m.now.Sub(m.lastSeen) < freshWindow:
		live = th.freshSt.Render("●")
	}
	status := "watching"
	if !m.lastSeen.IsZero() {
		status = "last edit " + ago(m.now, m.lastSeen)
	}
	right = append(right, live+" "+th.muted.Render(status)+" ")
	return spread(strings.Join(left, sep), strings.Join(right, sep), m.width)
}

func (m Model) renderFooter() string {
	th := m.theme
	if m.err != nil {
		return th.remSt.Render(" error: " + m.err.Error())
	}
	hint := func(k, d string) string { return th.key.Render(k) + " " + th.keyDesc.Render(d) }
	var hints []string
	switch m.focus {
	case panelMain:
		hints = []string{hint("j/k", "scroll"), hint("[ ]", "tab"), hint("d", "diff"), hint("e", "edit"), hint("esc", "back")}
	case panelActivity:
		hints = []string{hint("j/k", "move"), hint("enter", "jump"), hint("c", "clear"), hint("[ ]", "tab")}
	default:
		hints = []string{hint("j/k", "move"), hint("enter", "read"), hint("[ ]", "tab"), hint("d", "diff"), hint("e", "edit")}
		if m.focus == panelChanges {
			hints = append(hints, hint("s", "sort"))
		}
	}
	hints = append(hints, hint("f", "follow"), hint("z", "zoom"), hint("?", "help"), hint("q", "quit"))
	return ansi.Truncate(" "+strings.Join(hints, "  "), m.width, "…")
}

func (m Model) helpText(w int) string {
	th := m.theme
	rows := [][2]string{
		{"tab / shift+tab", "cycle panels"},
		{"1 2 3 4 / 0", "jump to Changes, Specs, Archive, Activity / main"},
		{"j k / up down", "move selection, or scroll when main is focused"},
		{"g G", "top / bottom"},
		{"enter / l", "focus main pane (on Activity: jump to item)"},
		{"esc / h", "back to list"},
		{"J K / ctrl+d ctrl+u", "scroll main pane by half a page"},
		{"[ ]", "previous / next file tab"},
		{"d", "toggle git diff vs HEAD for the current file"},
		{"e", "open the current file in $VISUAL / $EDITOR"},
		{"f", "follow mode: jump to whatever the agents touch next"},
		{"s", "sort changes by name / most recently edited"},
		{"z", "zoom main pane"},
		{"c", "clear activity log"},
		{"r", "force reload"},
		{"q", "quit"},
	}
	var sb strings.Builder
	sb.WriteString("\n")
	for _, r := range rows {
		sb.WriteString("  " + th.key.Render(fmt.Sprintf("%-22s", r[0])) + th.keyDesc.Render(r[1]) + "\n")
	}
	sb.WriteString("\n  " + th.muted.Render("Legend: ") +
		th.freshSt.Render("●") + th.muted.Render(" edited <15s  ") +
		th.warmSt.Render("●") + th.muted.Render(" edited <2m  ") +
		th.modSt.Render("M") + th.muted.Render(" uncommitted  ") +
		th.muted.Render("P D S T = proposal, design, spec deltas, tasks"))
	sb.WriteString("\n\n  " + th.subtle.Render("press any key to close"))
	return sb.String()
}

func (m Model) renderPanel(p panelID, r rect) string {
	th := m.theme
	focused := m.focus == p
	n := m.count(p)
	title := fmt.Sprintf("[%d] %s", p+1, panelTitles[p])
	if p == panelChanges && m.sortRecent {
		title += " (recent)"
	}
	if focused {
		title = th.title.Render(title)
	} else {
		title = th.muted.Render(title)
	}
	footer := ""
	if n > 0 {
		footer = th.muted.Render(fmt.Sprintf("%d of %d", m.cursors[p]+1, n))
	}

	w, h := r.w-2, r.h-2
	var rows []string
	if n == 0 {
		rows = append(rows, " "+th.subtle.Render(emptyText(p, m.snap)))
	}
	for i := m.offsets[p]; i < min(n, m.offsets[p]+h); i++ {
		sel := i == m.cursors[p]
		active := sel && (focused || (p == m.lastList && m.focus == panelMain))
		rows = append(rows, m.renderRow(p, i, w, sel, active))
	}
	return box(th, title, footer, strings.Join(rows, "\n"), r.w, r.h, focused)
}

func emptyText(p panelID, snap *spec.Snapshot) string {
	if snap == nil {
		return "loading…"
	}
	switch p {
	case panelChanges:
		return "no active changes"
	case panelSpecs:
		return "no specs yet"
	case panelArchive:
		return "nothing archived"
	default:
		return "watching for edits…"
	}
}

// painter renders styled cells, adding the selection background when the row
// is selected so inner style resets don't punch holes in the highlight.
type painter struct {
	bg  bool
	th  theme
	w   int
	out strings.Builder
}

func (p *painter) put(st lipgloss.Style, s string) {
	if p.bg {
		st = st.Background(p.th.selBg)
	}
	p.out.WriteString(st.Render(s))
}

func (p *painter) String() string {
	s := ansi.Truncate(p.out.String(), p.w, "…")
	if pad := p.w - lipgloss.Width(s); pad > 0 {
		st := lipgloss.NewStyle()
		if p.bg {
			st = st.Background(p.th.selBg)
		}
		s += st.Render(strings.Repeat(" ", pad))
	}
	return s
}

func (m Model) renderRow(p panelID, i, w int, sel, active bool) string {
	th := m.theme
	pt := &painter{bg: sel, th: th, w: w}
	nameSt := lipgloss.NewStyle().Foreground(th.text)
	if active {
		nameSt = nameSt.Bold(true).Foreground(th.accent)
	}
	cursor := " "
	if active {
		cursor = "▌"
	}
	pt.put(lipgloss.NewStyle().Foreground(th.accent), cursor)

	switch p {
	case panelChanges, panelArchive:
		var c spec.Change
		if p == panelChanges {
			c = m.changes()[i]
		} else {
			c = m.snap.Archive[i]
		}
		label, used := c.ID, 3
		if p == panelArchive {
			label = c.Name
			if c.ArchivedOn != "" {
				pt.put(th.muted, c.ArchivedOn[5:]+" ")
				used += 6
			}
		}
		pt.put(m.freshStyle(c.ModTime), m.freshDot(c.ModTime))
		putLabeled(pt, w, used, label, nameSt, m.dirMark(c.Dir), m.dirMarkStyle(c.Dir), m.changeRowRight(c, w))

	case panelSpecs:
		s := m.snap.Specs[i]
		pt.put(m.freshStyle(s.ModTime), m.freshDot(s.ModTime))
		var right segments
		if pending := len(m.pendingDeltas(s.ID)); pending > 0 {
			right = append(right, segment{th.modSt, fmt.Sprintf("Δ%d ", pending)})
		}
		right = append(right, segment{th.muted, fmt.Sprintf("%3d req", s.Requirements)})
		putLabeled(pt, w, 3, s.ID, nameSt, m.fileMark(s.Path), th.modSt, right)

	case panelActivity:
		e := m.events[i]
		ts := th.muted
		if m.now.Sub(e.Time) < freshWindow {
			ts = th.freshSt
		}
		pt.put(ts, e.Time.Format("15:04:05")+" ")
		icon, st := eventIcon(th, e.Kind)
		pt.put(st, icon+" ")
		idW := clamp(w/3, 8, 24)
		pt.put(nameSt, fmt.Sprintf("%-*s", idW, ansi.Truncate(e.ID, idW, "…"))+" ")
		pt.put(th.muted, e.Detail)
	}
	return pt.String()
}

// putLabeled writes a left-aligned label and right-aligned segments, with an
// optional git mark just before the segments. used is the width already
// written on the row.
func putLabeled(pt *painter, w, used int, label string, nameSt lipgloss.Style, mark string, markSt lipgloss.Style, right segments) {
	if mark != "" {
		mark += " "
	}
	rightW := right.Width() + lipgloss.Width(mark)
	label = ansi.Truncate(label, max(w-used-rightW-1, 1), "…")
	pt.put(nameSt, label)
	pt.put(lipgloss.NewStyle(), strings.Repeat(" ", max(w-used-lipgloss.Width(label)-rightW, 1)))
	if mark != "" {
		pt.put(markSt, mark)
	}
	for _, seg := range right {
		pt.put(seg.st, seg.s)
	}
}

type segment struct {
	st lipgloss.Style
	s  string
}

// changeRowRight is the "PDST ▰▰▱ 3/5" block on a change row.
func (m Model) changeRowRight(c spec.Change, w int) segments {
	th := m.theme
	var segs segments
	for _, k := range []struct {
		kind   spec.ArtifactKind
		letter string
	}{{spec.KindProposal, "P"}, {spec.KindDesign, "D"}, {spec.KindDelta, "S"}, {spec.KindTasks, "T"}} {
		st := th.subtle
		if c.Has(k.kind) {
			st = th.title
		}
		segs = append(segs, segment{st, k.letter})
	}
	if w >= 36 {
		segs = append(segs, segment{lipgloss.NewStyle(), " "})
		bar := 5
		filled := 0
		if c.Tasks.Total > 0 {
			filled = c.Tasks.Done * bar / c.Tasks.Total
		}
		fill := th.title
		if c.Tasks.Complete() {
			fill = th.addSt
		}
		segs = append(segs, segment{fill, strings.Repeat("▰", filled)}, segment{th.subtle, strings.Repeat("▱", bar-filled)})
	}
	count := ""
	if c.Tasks.Total > 0 {
		count = fmt.Sprintf("%d/%d", c.Tasks.Done, c.Tasks.Total)
	}
	st := th.muted
	if c.Tasks.Complete() {
		st = th.addSt
	}
	segs = append(segs, segment{st, fmt.Sprintf(" %5s", count)})
	return segs
}

type segments []segment

func (s segments) Width() int {
	n := 0
	for _, seg := range s {
		n += lipgloss.Width(seg.s)
	}
	return n
}

func eventIcon(th theme, k spec.EventKind) (string, lipgloss.Style) {
	switch k {
	case spec.EventCreated:
		return "+", th.addSt
	case spec.EventProgress:
		return "▸", th.title
	case spec.EventArchived:
		return "✓", th.renSt
	case spec.EventRemoved:
		return "-", th.remSt
	default:
		return "~", th.modSt
	}
}

func (m Model) freshDot(t time.Time) string {
	if m.now.Sub(t) < warmWindow {
		return "● "
	}
	return "  "
}

func (m Model) freshStyle(t time.Time) lipgloss.Style {
	if m.now.Sub(t) < freshWindow {
		return m.theme.freshSt
	}
	return m.theme.warmSt
}

// dirMark summarizes git status for files under a change dir.
func (m Model) dirMark(dir string) string {
	prefix := dir + string(filepath.Separator)
	untracked, modified := false, false
	for path, code := range m.git.Dirty {
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		if code == "??" {
			untracked = true
		} else {
			modified = true
		}
	}
	switch {
	case modified:
		return "M"
	case untracked:
		return "+"
	}
	return ""
}

func (m Model) dirMarkStyle(dir string) lipgloss.Style {
	if m.dirMark(dir) == "+" {
		return m.theme.addSt
	}
	return m.theme.modSt
}

func (m Model) fileMark(path string) string {
	switch code, ok := m.git.Dirty[path]; {
	case !ok:
		return ""
	case code == "??":
		return "+"
	default:
		return "M"
	}
}

// spread places left and right on one line of width w.
func spread(left, right string, w int) string {
	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return ansi.Truncate(left, max(w-lipgloss.Width(right)-1, 0), "…") + " " + right
	}
	return left + strings.Repeat(" ", gap) + right
}

func ago(now, t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := now.Sub(t)
	switch {
	case d < 2*time.Second:
		return "just now"
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return t.Format("2006-01-02")
	}
}

func tildePath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if rel, err := filepath.Rel(home, p); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.Join("~", rel)
		}
	}
	return p
}
