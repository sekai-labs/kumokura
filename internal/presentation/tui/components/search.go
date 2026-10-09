package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/styles"
)

type SearchBar struct {
	Input  textinput.Model
	Active bool
	styles styles.Styles
}

func NewSearchBar(s styles.Styles) SearchBar {
	ti := textinput.New()
	ti.Placeholder = "Type filter query... (Enter apply, Esc clear)"
	ti.CharLimit = 128
	ti.Width = 50

	return SearchBar{
		Input:  ti,
		Active: false,
		styles: s,
	}
}

func (sb *SearchBar) Render(width int) string {
	if !sb.Active && strings.TrimSpace(sb.Input.Value()) == "" {
		return ""
	}

	badge := lipgloss.NewStyle().
		Background(sb.styles.Theme.BorderActive).
		Foreground(sb.styles.Theme.CardBackground).
		Bold(true).
		Padding(0, 1).
		Render("🔍 FILTER")

	indicator := ""
	if sb.Active {
		indicator = lipgloss.NewStyle().
			Foreground(sb.styles.Theme.BadgeWarning).
			Bold(true).
			Render(" [LIVE] ")
	} else {
		indicator = lipgloss.NewStyle().
			Foreground(sb.styles.Theme.TextSubtle).
			Render(" [APPLIED] ")
	}

	inputBox := sb.Input.View()
	hint := lipgloss.NewStyle().
		Foreground(sb.styles.Theme.TextSubtle).
		Render("  (Press Esc to reset)")

	inner := lipgloss.JoinHorizontal(lipgloss.Center, badge, indicator, inputBox, hint)
	return lipgloss.NewStyle().
		Background(sb.styles.Theme.CardBackground).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(sb.styles.Theme.BorderActive).
		Padding(0, 1).
		Width(width).
		Render(inner)
}
