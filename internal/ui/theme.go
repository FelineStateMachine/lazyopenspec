package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// theme holds every style the UI uses, resolved for a light or dark terminal.
type theme struct {
	isDark bool

	accent, dim, faint, text          color.Color
	added, modified, removed, renamed color.Color
	fresh, warm, selBg                color.Color

	title, muted, subtle, bold lipgloss.Style
	sel, selDim                lipgloss.Style
	tabActive, tabInactive     lipgloss.Style
	key, keyDesc               lipgloss.Style
	addSt, modSt, remSt, renSt lipgloss.Style
	freshSt, warmSt            lipgloss.Style
	diffAdd, diffDel, diffHunk lipgloss.Style
}

func newTheme(isDark bool) theme {
	ld := lipgloss.LightDark(isDark)
	t := theme{
		isDark:   isDark,
		accent:   ld(lipgloss.Color("#007F5F"), lipgloss.Color("#5FD7AF")),
		dim:      ld(lipgloss.Color("#8A8A8A"), lipgloss.Color("#6C6C6C")),
		faint:    ld(lipgloss.Color("#BCBCBC"), lipgloss.Color("#3A3A3A")),
		text:     ld(lipgloss.Color("#1C1C1C"), lipgloss.Color("#E4E4E4")),
		added:    ld(lipgloss.Color("#2E7D32"), lipgloss.Color("#87D75F")),
		modified: ld(lipgloss.Color("#B26A00"), lipgloss.Color("#FFD75F")),
		removed:  ld(lipgloss.Color("#C62828"), lipgloss.Color("#FF5F87")),
		renamed:  ld(lipgloss.Color("#1565C0"), lipgloss.Color("#5FAFFF")),
		fresh:    ld(lipgloss.Color("#00A86B"), lipgloss.Color("#5FFF87")),
		warm:     ld(lipgloss.Color("#7FBF9F"), lipgloss.Color("#3F8F5F")),
		selBg:    ld(lipgloss.Color("#E0E8E4"), lipgloss.Color("#26323A")),
	}
	t.title = lipgloss.NewStyle().Foreground(t.accent).Bold(true)
	t.muted = lipgloss.NewStyle().Foreground(t.dim)
	t.subtle = lipgloss.NewStyle().Foreground(t.faint)
	t.bold = lipgloss.NewStyle().Foreground(t.text).Bold(true)
	t.sel = lipgloss.NewStyle().Background(t.selBg).Foreground(t.text).Bold(true)
	t.selDim = lipgloss.NewStyle().Background(t.selBg)
	t.tabActive = lipgloss.NewStyle().Foreground(t.accent).Bold(true).Underline(true)
	t.tabInactive = lipgloss.NewStyle().Foreground(t.dim)
	t.key = lipgloss.NewStyle().Foreground(t.accent).Bold(true)
	t.keyDesc = lipgloss.NewStyle().Foreground(t.dim)
	t.addSt = lipgloss.NewStyle().Foreground(t.added)
	t.modSt = lipgloss.NewStyle().Foreground(t.modified)
	t.remSt = lipgloss.NewStyle().Foreground(t.removed)
	t.renSt = lipgloss.NewStyle().Foreground(t.renamed)
	t.freshSt = lipgloss.NewStyle().Foreground(t.fresh).Bold(true)
	t.warmSt = lipgloss.NewStyle().Foreground(t.warm)
	t.diffAdd = lipgloss.NewStyle().Foreground(t.added)
	t.diffDel = lipgloss.NewStyle().Foreground(t.removed)
	t.diffHunk = lipgloss.NewStyle().Foreground(t.renamed)
	return t
}

func (t theme) glamourStyle() string {
	if t.isDark {
		return "dark"
	}
	return "light"
}
