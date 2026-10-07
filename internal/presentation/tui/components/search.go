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
	ti.Placeholder = "Filter objects by prefix or substring (/)..."
	ti.CharLimit = 128
	ti.Width = 40

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
	label := sb.styles.StatusKey.Render("Filter: ")
	inputBox := sb.Input.View()
	return lipgloss.JoinHorizontal(lipgloss.Top, label, inputBox)
}
