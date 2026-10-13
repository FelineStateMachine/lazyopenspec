// Package ui is the Bubble Tea front end for lazyopenspec.
package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/dami/lazyopenspec/internal/gitx"
	"github.com/dami/lazyopenspec/internal/spec"
)

type panelID int

const (
	panelChanges panelID = iota
	panelSpecs
	panelArchive
	panelActivity
	panelMain

	numLists = 4
)

var panelTitles = [numLists]string{"Changes", "Specs", "Archive", "Activity"}

const (
	maxEvents     = 500
	gitEvery      = 5 * time.Second
	freshWindow   = 15 * time.Second
	warmWindow    = 2 * time.Minute
	scrollStep    = 3
	defaultPollMs = 400
)

// Options configure the UI.
type Options struct {
	Root     string
	Interval time.Duration
	Follow   bool
}

type (
	pollMsg    time.Time
	scannedMsg spec.Fingerprint
	loadedMsg  struct {
		snap *spec.Snapshot
		err  error
	}
	gitMsg  gitx.Status
	diffMsg struct {
		path string
		key  string // path + mtime + git status the diff was taken at
		text string
		err  error
	}
	editorDoneMsg struct{ err error }
)

// Model is the root Bubble Tea model.
type Model struct {
	opts  Options
	theme theme

	width, height int
	focus         panelID
	lastList      panelID // list panel driving the main pane
	cursors       [numLists]int
	offsets       [numLists]int
	selected      [numLists]string // selected item ID per list, kept across reloads
	sortRecent    bool
	follow        bool
	diffMode      bool
	zoom          bool
	showHelp      bool

	snap     *spec.Snapshot
	fp       spec.Fingerprint
	loading  bool
	err      error
	git      gitx.Status
	gitAt    time.Time
	gitBusy  bool         // a git read is in flight
	gitAgain bool         // another read was requested while busy
	events   []spec.Event // newest first
	now      time.Time
	lastSeen time.Time // time of the most recent event

	vp       viewport.Model
	tabs     map[string]string // target key -> selected tab key
	scroll   map[string]int    // content key -> remembered y offset
	mainKey  string            // identity of the content currently in vp
	viewKey  string            // target/tab/mode shown in vp, for scroll memory
	renderer *mdRenderer
	diffs    map[string]diffMsg // latest diff per path
	diffReq  map[string]bool    // diff keys currently being fetched
}

// New constructs the root model.
func New(opts Options) Model {
	if opts.Interval <= 0 {
		opts.Interval = defaultPollMs * time.Millisecond
	}
	vp := viewport.New()
	vp.MouseWheelEnabled = false // the root model routes wheel events
	return Model{
		opts:     opts,
		theme:    newTheme(true),
		follow:   opts.Follow,
		vp:       vp,
		tabs:     map[string]string{},
		scroll:   map[string]int{},
		renderer: newMDRenderer(),
		diffs:    map[string]diffMsg{},
		diffReq:  map[string]bool{},
		now:      time.Now(),
	}
}

// Init starts the first load, git read, and poll loop.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tea.RequestBackgroundColor, m.scanCmd())
}

func (m Model) scanCmd() tea.Cmd {
	dir := filepath.Join(m.opts.Root, spec.DirName)
	return func() tea.Msg { return scannedMsg(spec.Scan(dir)) }
}

func (m Model) loadCmd() tea.Cmd {
	root := m.opts.Root
	return func() tea.Msg {
		snap, err := spec.Load(root)
		return loadedMsg{snap, err}
	}
}

// refreshGit starts a git read unless one is in flight, in which case it
// queues exactly one more so results never arrive out of order.
func (m *Model) refreshGit() tea.Cmd {
	if m.gitBusy {
		m.gitAgain = true
		return nil
	}
	m.gitBusy, m.gitAt = true, m.now
	root := m.opts.Root
	dir := filepath.Join(root, spec.DirName)
	return func() tea.Msg { return gitMsg(gitx.Read(root, dir)) }
}

func (m Model) pollCmd() tea.Cmd {
	return tea.Tick(m.opts.Interval, func(t time.Time) tea.Msg { return pollMsg(t) })
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.clampAll()

	case tea.BackgroundColorMsg:
		m.theme = newTheme(msg.IsDark())
		m.mainKey = "" // re-render markdown with the new style

	case scannedMsg:
		// The fingerprint is only recorded when a load starts, so edits that
		// land while a load is in flight are caught by the next scan.
		fp := spec.Fingerprint(msg)
		if !m.loading && (m.fp == nil || len(m.fp.Changed(fp)) > 0) {
			m.fp = fp
			m.loading = true
			cmds = append(cmds, m.loadCmd())
		}
		cmds = append(cmds, m.pollCmd())

	case pollMsg:
		m.now = time.Time(msg)
		cmds = append(cmds, m.scanCmd())
		if m.now.Sub(m.gitAt) >= gitEvery {
			cmds = append(cmds, m.refreshGit())
		}

	case loadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err != nil {
			m.fp = nil // retry on the next poll even if nothing else changes
		} else {
			cmds = append(cmds, m.applySnapshot(msg.snap)...)
		}

	case gitMsg:
		m.git = gitx.Status(msg)
		m.gitBusy = false
		if m.gitAgain {
			m.gitAgain = false
			cmds = append(cmds, m.refreshGit())
		}

	case diffMsg:
		m.diffs[msg.path] = msg
		delete(m.diffReq, msg.key)
		m.mainKey = ""

	case editorDoneMsg:
		if msg.err != nil {
			m.err = msg.err
		}

	case tea.KeyPressMsg:
		cmds = append(cmds, m.handleKey(msg))

	case tea.MouseWheelMsg:
		m.handleWheel(msg)

	case tea.MouseClickMsg:
		m.handleClick(msg)
	}

	cmds = append(cmds, m.syncMain())
	return m, tea.Batch(cmds...)
}

// applySnapshot swaps in a freshly loaded snapshot, records activity, and
// keeps each list's selection pinned to the same item ID.
func (m *Model) applySnapshot(next *spec.Snapshot) []tea.Cmd {
	prev := m.snap
	m.snap = next
	m.now = time.Now()
	events := spec.Diff(prev, next, m.now)
	for i := len(events) - 1; i >= 0; i-- {
		m.events = append([]spec.Event{events[i]}, m.events...)
	}
	if len(m.events) > maxEvents {
		m.events = m.events[:maxEvents]
	}
	if len(events) > 0 {
		m.lastSeen = m.now
	}
	if m.cursors[panelActivity] > 0 || m.offsets[panelActivity] > 0 {
		// Keep a scrolled-back activity cursor on the same event; at the top
		// it keeps tracking the newest one.
		m.cursors[panelActivity] += len(events)
		m.offsets[panelActivity] += len(events)
	}
	for p := range panelID(numLists) {
		if p == panelActivity {
			continue
		}
		if i := m.indexOf(p, m.selected[p]); i >= 0 {
			m.cursors[p] = i
		}
	}
	if m.follow && len(events) > 0 {
		m.followEvent(events[len(events)-1])
	}
	m.clampAll()
	return []tea.Cmd{m.refreshGit()}
}

// followEvent moves the selection to the item an event refers to.
func (m *Model) followEvent(e spec.Event) {
	var p panelID
	switch e.Item {
	case spec.ItemChange:
		p = panelChanges
	case spec.ItemSpec:
		p = panelSpecs
	case spec.ItemArchive:
		p = panelArchive
	}
	i := m.indexOf(p, e.ID)
	if i < 0 {
		return
	}
	m.cursors[p] = i
	m.selected[p] = e.ID
	m.lastList = p
	if m.focus != panelMain {
		m.focus = p
	}
	if e.Rel != "" {
		m.tabs[targetKey(p, e.ID)] = e.Rel
	}
}

func (m Model) changes() []spec.Change {
	if m.snap == nil {
		return nil
	}
	cs := m.snap.Changes
	if m.sortRecent {
		cs = append([]spec.Change(nil), cs...)
		sort.SliceStable(cs, func(i, j int) bool { return cs[i].ModTime.After(cs[j].ModTime) })
	}
	return cs
}

func (m Model) count(p panelID) int {
	if m.snap == nil && p != panelActivity {
		return 0
	}
	switch p {
	case panelChanges:
		return len(m.snap.Changes)
	case panelSpecs:
		return len(m.snap.Specs)
	case panelArchive:
		return len(m.snap.Archive)
	case panelActivity:
		return len(m.events)
	}
	return 0
}

// idAt returns the item ID at index i of list p.
func (m Model) idAt(p panelID, i int) string {
	if i < 0 || i >= m.count(p) {
		return ""
	}
	switch p {
	case panelChanges:
		return m.changes()[i].ID
	case panelSpecs:
		return m.snap.Specs[i].ID
	case panelArchive:
		return m.snap.Archive[i].ID
	case panelActivity:
		e := m.events[i]
		return e.Time.Format(time.RFC3339Nano) + e.ID + e.Detail
	}
	return ""
}

func (m Model) indexOf(p panelID, id string) int {
	if id == "" {
		return -1
	}
	for i := range m.count(p) {
		if m.idAt(p, i) == id {
			return i
		}
	}
	return -1
}

func (m *Model) clampAll() {
	g := m.geometry()
	for p := range panelID(numLists) {
		n := m.count(p)
		h := g.panels[p].h - 2
		c := clamp(m.cursors[p], 0, max(n-1, 0))
		off := m.offsets[p]
		if c < off {
			off = c
		}
		if h > 0 && c >= off+h {
			off = c - h + 1
		}
		off = clamp(off, 0, max(n-h, 0))
		m.cursors[p], m.offsets[p] = c, off
		m.selected[p] = m.idAt(p, c)
	}
}

func (m *Model) moveCursor(p panelID, delta int) {
	if p >= numLists {
		return
	}
	m.cursors[p] += delta
	m.clampAll()
	m.lastList = p
}

func (m *Model) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	if m.showHelp {
		m.showHelp = false
		return nil
	}
	key := msg.String()
	switch key {
	case "q", "ctrl+c":
		return tea.Quit
	case "?":
		m.showHelp = true
	case "tab":
		m.setFocus((m.focus + 1) % (numLists + 1))
	case "shift+tab":
		m.setFocus((m.focus + numLists) % (numLists + 1))
	case "1", "2", "3", "4":
		m.setFocus(panelID(key[0] - '1'))
	case "0":
		m.setFocus(panelMain)
	case "enter", "l", "right":
		if m.focus == panelActivity {
			if i := m.cursors[panelActivity]; i < len(m.events) {
				m.jumpTo(m.events[i])
			}
		} else if m.focus < numLists {
			m.setFocus(panelMain)
		}
	case "esc", "h", "left":
		if m.focus == panelMain {
			m.setFocus(m.lastList)
		}
	case "j", "down":
		m.scrollOrMove(1)
	case "k", "up":
		m.scrollOrMove(-1)
	case "g", "home":
		m.scrollOrMove(-1 << 20)
	case "G", "end":
		m.scrollOrMove(1 << 20)
	case "J", "ctrl+d", "pgdown":
		m.vp.HalfPageDown()
	case "K", "ctrl+u", "pgup":
		m.vp.HalfPageUp()
	case "]":
		m.cycleTab(1)
	case "[":
		m.cycleTab(-1)
	case "d":
		m.diffMode = !m.diffMode
	case "f", "F":
		m.follow = !m.follow
	case "s":
		m.sortRecent = !m.sortRecent
		if i := m.indexOf(panelChanges, m.selected[panelChanges]); i >= 0 {
			m.cursors[panelChanges] = i
		}
		m.clampAll()
	case "z":
		m.zoom = !m.zoom
		m.clampAll()
	case "r":
		m.loading = true
		return tea.Batch(m.loadCmd(), m.refreshGit())
	case "c":
		m.events = nil
		m.cursors[panelActivity] = 0
		m.clampAll()
	case "e":
		return m.editCmd()
	}
	return nil
}

func (m *Model) setFocus(p panelID) {
	m.focus = p
	if p < numLists && p != panelActivity {
		m.lastList = p
	}
	if p == panelActivity {
		if i := m.cursors[panelActivity]; i < len(m.events) {
			m.peekEvent(m.events[i])
		}
	}
}

func (m *Model) scrollOrMove(delta int) {
	if m.focus == panelMain {
		switch {
		case delta <= -1<<20:
			m.vp.GotoTop()
		case delta >= 1<<20:
			m.vp.GotoBottom()
		case delta > 0:
			m.vp.ScrollDown(delta)
		default:
			m.vp.ScrollUp(-delta)
		}
		return
	}
	m.cursors[m.focus] += delta
	m.clampAll()
	if m.focus == panelActivity {
		if i := m.cursors[panelActivity]; i < len(m.events) {
			m.peekEvent(m.events[i])
		}
		return
	}
	m.lastList = m.focus
}

// peekEvent shows an activity item in the main pane without moving focus.
func (m *Model) peekEvent(e spec.Event) {
	p := map[spec.ItemKind]panelID{
		spec.ItemChange: panelChanges, spec.ItemSpec: panelSpecs, spec.ItemArchive: panelArchive,
	}[e.Item]
	if i := m.indexOf(p, e.ID); i >= 0 {
		m.cursors[p] = i
		m.selected[p] = e.ID
		m.lastList = p
		if e.Rel != "" {
			m.tabs[targetKey(p, e.ID)] = e.Rel
		}
		m.clampAll()
	}
}

// jumpTo focuses the list containing an event's item.
func (m *Model) jumpTo(e spec.Event) {
	m.peekEvent(e)
	m.focus = m.lastList
}

func (m *Model) handleWheel(msg tea.MouseWheelMsg) {
	mouse := msg.Mouse()
	delta := 0
	switch mouse.Button {
	case tea.MouseWheelDown:
		delta = 1
	case tea.MouseWheelUp:
		delta = -1
	default:
		return
	}
	g := m.geometry()
	if g.main.contains(mouse.X, mouse.Y) {
		if delta > 0 {
			m.vp.ScrollDown(scrollStep)
		} else {
			m.vp.ScrollUp(scrollStep)
		}
		return
	}
	for p := range panelID(numLists) {
		if g.panels[p].contains(mouse.X, mouse.Y) {
			m.setFocus(p)
			m.scrollOrMove(delta)
			return
		}
	}
}

func (m *Model) handleClick(msg tea.MouseClickMsg) {
	mouse := msg.Mouse()
	if mouse.Button != tea.MouseLeft {
		return
	}
	g := m.geometry()
	if g.main.contains(mouse.X, mouse.Y) {
		m.setFocus(panelMain)
		return
	}
	for p := range panelID(numLists) {
		r := g.panels[p]
		if !r.contains(mouse.X, mouse.Y) {
			continue
		}
		row := mouse.Y - r.y - 1
		if row >= 0 && row < r.h-2 {
			if i := m.offsets[p] + row; i < m.count(p) {
				m.cursors[p] = i
			}
		}
		m.clampAll()
		m.setFocus(p)
		return
	}
}

func (m *Model) editCmd() tea.Cmd {
	t := m.target()
	tab, ok := m.currentTab(t)
	if !ok {
		return nil
	}
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	args := append(strings.Fields(editor), tab.path)
	c := exec.Command(args[0], args[1:]...) //nolint:gosec // user-chosen editor
	return tea.ExecProcess(c, func(err error) tea.Msg { return editorDoneMsg{err} })
}

// View renders the UI.
func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.WindowTitle = "lazyopenspec - " + filepath.Base(m.opts.Root)
	return v
}
