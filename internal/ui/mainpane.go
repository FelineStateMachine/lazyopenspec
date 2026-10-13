package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"

	"github.com/dami/lazyopenspec/internal/gitx"
	"github.com/dami/lazyopenspec/internal/spec"
)

// target is the item shown in the main pane.
type target struct {
	panel  panelID
	change spec.Change
	spec   spec.Spec
	ok     bool
}

func (t target) key() string {
	switch t.panel {
	case panelSpecs:
		return targetKey(t.panel, t.spec.ID)
	default:
		return targetKey(t.panel, t.change.ID)
	}
}

func targetKey(p panelID, id string) string { return fmt.Sprintf("%d:%s", p, id) }

// tab is one file viewable for a target.
type tab struct {
	key     string
	label   string
	path    string
	mod     time.Time
	content string
}

func (m Model) target() target {
	if m.snap == nil {
		return target{}
	}
	p := m.lastList
	i := m.cursors[p]
	switch p {
	case panelChanges:
		if cs := m.changes(); i < len(cs) {
			return target{panel: p, change: cs[i], ok: true}
		}
	case panelSpecs:
		if i < len(m.snap.Specs) {
			return target{panel: p, spec: m.snap.Specs[i], ok: true}
		}
	case panelArchive:
		if i < len(m.snap.Archive) {
			return target{panel: p, change: m.snap.Archive[i], ok: true}
		}
	}
	return target{}
}

// pendingDeltas lists active-change delta artifacts that target a spec.
func (m Model) pendingDeltas(specID string) []tab {
	if m.snap == nil {
		return nil
	}
	var tabs []tab
	for _, c := range m.snap.Changes {
		for _, a := range c.Artifacts {
			if a.Kind == spec.KindDelta && a.Capability() == specID {
				tabs = append(tabs, tab{
					key: "Δ " + c.ID, label: "Δ " + c.ID,
					path: a.Path, mod: a.ModTime, content: a.Content,
				})
			}
		}
	}
	return tabs
}

func (m Model) tabsFor(t target) []tab {
	if !t.ok {
		return nil
	}
	if t.panel == panelSpecs {
		tabs := []tab{{key: "spec.md", label: "spec.md", path: t.spec.Path, mod: t.spec.ModTime, content: t.spec.Content}}
		return append(tabs, m.pendingDeltas(t.spec.ID)...)
	}
	tabs := make([]tab, 0, len(t.change.Artifacts))
	for _, a := range t.change.Artifacts {
		tabs = append(tabs, tab{key: a.Rel, label: a.Label(), path: a.Path, mod: a.ModTime, content: a.Content})
	}
	return tabs
}

func (m Model) currentTab(t target) (tab, bool) {
	tabs := m.tabsFor(t)
	if len(tabs) == 0 {
		return tab{}, false
	}
	want := m.tabs[t.key()]
	for _, tb := range tabs {
		if tb.key == want {
			return tb, true
		}
	}
	return tabs[0], true
}

func (m *Model) cycleTab(delta int) {
	t := m.target()
	tabs := m.tabsFor(t)
	if len(tabs) == 0 {
		return
	}
	cur, _ := m.currentTab(t)
	i := 0
	for j, tb := range tabs {
		if tb.key == cur.key {
			i = j
		}
	}
	i = (i + delta + len(tabs)) % len(tabs)
	m.tabs[t.key()] = tabs[i].key
}

const mainChromeRows = 5 // 3 header rows, tab bar, rule

func (m Model) mainInner() (w, h int) {
	g := m.geometry()
	return max(g.main.w-2, 0), max(g.main.h-2, 0)
}

// syncMain refreshes the viewport when the visible content's identity
// changes. Scroll position is kept per target/tab so live edits to the file
// being read don't yank the view back to the top.
func (m *Model) syncMain() tea.Cmd {
	w, h := m.mainInner()
	vpH := max(h-mainChromeRows, 0)
	t := m.target()
	tb, hasTab := m.currentTab(t)
	dirty := m.git.Dirty[tb.path]

	viewKey := fmt.Sprintf("%s|%s|%t", t.key(), tb.key, m.diffMode)
	contentKey := fmt.Sprintf("%s|%d|%s|%dx%d|%t", viewKey, tb.mod.UnixNano(), dirty, w, vpH, m.theme.isDark)
	if contentKey == m.mainKey && m.vp.Width() == w && m.vp.Height() == vpH {
		return nil
	}
	if m.viewKey != "" {
		m.scroll[m.viewKey] = m.vp.YOffset()
	}
	m.mainKey, m.viewKey = contentKey, viewKey

	m.vp.SetWidth(w)
	m.vp.SetHeight(vpH)

	var cmd tea.Cmd
	switch {
	case m.snap != nil && !m.snap.Exists:
		m.vp.SetContent(m.theme.muted.Render(fmt.Sprintf(
			"\n  No %s/ directory in %s yet.\n  Waiting for one to appear (run `openspec init`).",
			spec.DirName, m.opts.Root)))
	case !t.ok || !hasTab:
		m.vp.SetContent(m.theme.muted.Render("\n  Nothing selected."))
	case m.diffMode:
		var content string
		content, cmd = m.diffContent(tb, dirty)
		m.vp.SetContent(content)
	default:
		m.vp.SetContent(m.renderer.render(tb.path, tb.mod, tb.content, w, m.theme.glamourStyle()))
	}
	m.vp.SetYOffset(m.scroll[viewKey])
	return cmd
}

func (m *Model) diffContent(tb tab, dirty string) (string, tea.Cmd) {
	if !m.git.IsRepo {
		return m.theme.muted.Render("\n  Not a git repository; diff unavailable."), nil
	}
	if dirty == "" {
		return m.theme.muted.Render("\n  No uncommitted changes to " + m.relPath(tb.path) + "."), nil
	}
	key := fmt.Sprintf("%s|%d|%s", tb.path, tb.mod.UnixNano(), dirty)
	if d, ok := m.diffs[key]; ok {
		if d.err != nil {
			return m.theme.remSt.Render("\n  git diff failed: " + d.err.Error()), nil
		}
		return m.colorDiff(d.text), nil
	}
	if m.diffReq[key] {
		return m.theme.muted.Render("\n  Loading diff…"), nil
	}
	m.diffReq[key] = true
	root, path := m.opts.Root, tb.path
	untracked := dirty == "??"
	return m.theme.muted.Render("\n  Loading diff…"), func() tea.Msg {
		text, err := gitx.Diff(root, path, untracked)
		return diffMsg{key: key, text: text, err: err}
	}
}

func (m Model) colorDiff(diff string) string {
	t := m.theme
	var sb strings.Builder
	for line := range strings.SplitSeq(strings.TrimRight(diff, "\n"), "\n") {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"),
			strings.HasPrefix(line, "diff "), strings.HasPrefix(line, "index "),
			strings.HasPrefix(line, "new file"), strings.HasPrefix(line, "deleted file"):
			line = t.muted.Render(line)
		case strings.HasPrefix(line, "@@"):
			line = t.diffHunk.Render(line)
		case strings.HasPrefix(line, "+"):
			line = t.diffAdd.Render(line)
		case strings.HasPrefix(line, "-"):
			line = t.diffDel.Render(line)
		}
		sb.WriteString(" ")
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return sb.String()
}

func (m Model) relPath(path string) string {
	if rel, err := filepath.Rel(m.opts.Root, path); err == nil {
		return rel
	}
	return path
}

// renderMain draws the main pane: item header, tab bar, and viewport.
func (m Model) renderMain(r rect) string {
	w := max(r.w-2, 0)
	t := m.target()
	focused := m.focus == panelMain

	if m.showHelp {
		return box(m.theme, m.theme.title.Render("Keys"), "", m.helpText(w), r.w, r.h, true)
	}

	var title string
	var header []string
	switch {
	case !t.ok:
		title = "lazyopenspec"
		header = []string{"", "", ""}
	case t.panel == panelSpecs:
		title = "Spec " + m.theme.bold.Render(t.spec.ID)
		header = m.specHeader(t.spec, w)
	case t.change.Archived:
		title = "Archived " + m.theme.bold.Render(t.change.Name)
		header = m.changeHeader(t.change, w)
	default:
		title = "Change " + m.theme.bold.Render(t.change.ID)
		header = m.changeHeader(t.change, w)
	}

	lines := append(header, m.tabBar(t, w), m.theme.subtle.Render(strings.Repeat("─", w)))
	body := strings.Join(lines, "\n") + "\n" + m.vp.View()

	footer := ""
	if m.vp.TotalLineCount() > m.vp.Height() {
		footer = m.theme.muted.Render(fmt.Sprintf("%3.f%%", m.vp.ScrollPercent()*100))
	}
	return box(m.theme, title, footer, body, r.w, r.h, focused)
}

func (m Model) changeHeader(c spec.Change, w int) []string {
	th := m.theme
	right := ""
	if c.Archived {
		right = "archived " + c.ArchivedOn
	} else {
		right = "edited " + ago(m.now, c.ModTime)
	}
	if c.Schema != "" {
		right = c.Schema + "  " + right
	}
	line1 := spread(" "+th.bold.Render(c.Name), th.muted.Render(right)+" ", w)

	summary := c.Summary
	if summary == "" {
		summary = "(no proposal yet)"
	}
	line2 := " " + th.muted.Render(summary)

	chip := func(label string, ok bool) string {
		if ok {
			return th.addSt.Render("✓ " + label)
		}
		return th.subtle.Render("○ " + label)
	}
	parts := []string{
		chip("proposal", c.Has(spec.KindProposal)),
		chip("design", c.Has(spec.KindDesign)),
		chip("specs", c.Has(spec.KindDelta)),
		chip("tasks", c.Has(spec.KindTasks)),
	}
	if c.Tasks.Total > 0 {
		parts = append(parts, m.progressBar(c.Tasks, 12)+" "+m.taskCount(c.Tasks))
	}
	if d := c.Deltas; d.Total() > 0 {
		parts = append(parts, m.deltaSummary(d))
	}
	line3 := " " + strings.Join(parts, "  ")
	return []string{line1, line2, line3}
}

func (m Model) specHeader(s spec.Spec, w int) []string {
	th := m.theme
	title := s.Title
	if title == "" {
		title = s.ID
	}
	line1 := spread(" "+th.bold.Render(title), th.muted.Render("edited "+ago(m.now, s.ModTime))+" ", w)
	purpose := s.Purpose
	if purpose == "" {
		purpose = "(no purpose)"
	}
	line2 := " " + th.muted.Render(purpose)
	stats := fmt.Sprintf("%d requirements  %d scenarios", s.Requirements, s.Scenarios)
	line3 := " " + th.muted.Render(stats)
	if pending := m.pendingDeltas(s.ID); len(pending) > 0 {
		ids := make([]string, len(pending))
		for i, p := range pending {
			ids[i] = strings.TrimPrefix(p.label, "Δ ")
		}
		line3 += "  " + th.modSt.Render("Δ pending: "+strings.Join(ids, ", "))
	}
	return []string{line1, line2, line3}
}

func (m Model) tabBar(t target, w int) string {
	th := m.theme
	tabs := m.tabsFor(t)
	cur, _ := m.currentTab(t)
	var parts []string
	for _, tb := range tabs {
		// Style each piece separately: underlined styles re-wrap every rune
		// and would mangle an already-styled substring.
		st := th.tabInactive
		if tb.key == cur.key {
			st = th.tabActive
		}
		part := st.Render(tb.label)
		if m.now.Sub(tb.mod) < freshWindow {
			part = th.freshSt.Render("● ") + part
		}
		if mark := m.gitMark(tb.path); mark != "" {
			part += " " + mark
		}
		parts = append(parts, part)
	}
	left := " " + strings.Join(parts, th.subtle.Render(" │ "))
	right := ""
	if m.diffMode {
		right = th.modSt.Render("[diff vs HEAD]") + " "
	}
	return spread(left, right, w)
}

func (m Model) gitMark(path string) string {
	code, ok := m.git.Dirty[path]
	if !ok {
		return ""
	}
	switch strings.TrimSpace(code) {
	case "??", "A":
		return m.theme.addSt.Render("+")
	case "D":
		return m.theme.remSt.Render("-")
	default:
		return m.theme.modSt.Render("M")
	}
}

func (m Model) progressBar(ts spec.TaskStats, width int) string {
	if ts.Total == 0 || width <= 0 {
		return m.theme.subtle.Render(strings.Repeat("▱", max(width, 0)))
	}
	filled := ts.Done * width / ts.Total
	st := m.theme.title
	if ts.Complete() {
		st = m.theme.addSt
	}
	return st.Render(strings.Repeat("▰", filled)) + m.theme.subtle.Render(strings.Repeat("▱", width-filled))
}

func (m Model) taskCount(ts spec.TaskStats) string {
	s := fmt.Sprintf("%d/%d", ts.Done, ts.Total)
	if ts.Complete() {
		return m.theme.addSt.Render(s)
	}
	return m.theme.muted.Render(s)
}

func (m Model) deltaSummary(d spec.DeltaStats) string {
	th := m.theme
	var parts []string
	if d.Added > 0 {
		parts = append(parts, th.addSt.Render(fmt.Sprintf("+%d", d.Added)))
	}
	if d.Modified > 0 {
		parts = append(parts, th.modSt.Render(fmt.Sprintf("~%d", d.Modified)))
	}
	if d.Removed > 0 {
		parts = append(parts, th.remSt.Render(fmt.Sprintf("-%d", d.Removed)))
	}
	if d.Renamed > 0 {
		parts = append(parts, th.renSt.Render(fmt.Sprintf("→%d", d.Renamed)))
	}
	return th.muted.Render("Δ ") + strings.Join(parts, " ")
}

// mdRenderer caches glamour renderers per style/width and rendered output
// per file version, so redraws and repeat visits are cheap.
type mdRenderer struct {
	renderers map[string]*glamour.TermRenderer
	cache     map[string]string
}

func newMDRenderer() *mdRenderer {
	return &mdRenderer{renderers: map[string]*glamour.TermRenderer{}, cache: map[string]string{}}
}

func (r *mdRenderer) render(path string, mod time.Time, src string, width int, style string) string {
	key := fmt.Sprintf("%s|%d|%d|%s", path, mod.UnixNano(), width, style)
	if out, ok := r.cache[key]; ok {
		return out
	}
	rkey := fmt.Sprintf("%s|%d", style, width)
	tr, ok := r.renderers[rkey]
	if !ok {
		var err error
		tr, err = glamour.NewTermRenderer(
			glamour.WithStandardStyle(style),
			glamour.WithWordWrap(max(width-4, 20)),
		)
		if err != nil {
			return src
		}
		r.renderers[rkey] = tr
	}
	out, err := tr.Render(src)
	if err != nil {
		out = src
	}
	if len(r.cache) > 256 {
		clear(r.cache)
	}
	r.cache[key] = out
	return out
}
