package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/styles"
)

type ModalDialog struct {
	Title       string
	Message     string
	Options     []string
	SelectedOpt int
	styles      styles.Styles
}

func NewModalDialog(title, message string, options []string, s styles.Styles) ModalDialog {
	return ModalDialog{
		Title:       title,
		Message:     message,
		Options:     options,
		SelectedOpt: 0,
		styles:      s,
	}
}

func (m *ModalDialog) Render(totalWidth, totalHeight int) string {
	dialogWidth := 50
	if totalWidth-6 < dialogWidth {
		dialogWidth = totalWidth - 6
	}

	titleRendered := m.styles.ModalHeader.Render(m.Title)
	msgRendered := m.styles.ModalBody.Width(dialogWidth - 4).Render(m.Message)

	var optsRendered []string
	for i, opt := range m.Options {
		if i == m.SelectedOpt {
			optsRendered = append(optsRendered, m.styles.ActiveTab.Render(opt))
		} else {
			optsRendered = append(optsRendered, m.styles.InactiveTab.Render(opt))
		}
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(optsRendered, "  "))

	content := lipgloss.JoinVertical(lipgloss.Left, titleRendered, msgRendered, "", buttons)
	box := m.styles.ModalBox.Width(dialogWidth).Render(content)

	boxWidth := lipgloss.Width(box)
	boxHeight := lipgloss.Height(box)

	padX := (totalWidth - boxWidth) / 2
	if padX < 0 {
		padX = 0
	}
	padY := (totalHeight - boxHeight) / 2
	if padY < 0 {
		padY = 0
	}

	return lipgloss.Place(totalWidth, totalHeight, lipgloss.Center, lipgloss.Center, box)
}

type UploadModal struct {
	Input       textinput.Model
	TargetKey   string
	Active      bool
	Destination string
	ErrorText   string
	styles      styles.Styles
}

func NewUploadModal(s styles.Styles) UploadModal {
	ti := textinput.New()
	ti.Placeholder = "/path/to/local/file.txt"
	ti.CharLimit = 512
	ti.Width = 44

	return UploadModal{
		Input:  ti,
		styles: s,
	}
}

func (u *UploadModal) Render(totalWidth, totalHeight int) string {
	dialogWidth := 58
	if totalWidth-6 < dialogWidth {
		dialogWidth = totalWidth - 6
	}
	if dialogWidth < 30 {
		dialogWidth = 30
	}

	titleRendered := u.styles.ModalHeader.Render("UPLOAD OBJECT")
	destLabel := u.styles.StatusKey.Render("Target Destination: ") + u.styles.StatusDesc.Render(u.Destination)
	promptLabel := u.styles.NormalRow.Render("Local file path:")
	inputBox := u.Input.View()

	var errLine string
	if u.ErrorText != "" {
		errLine = u.styles.DangerPill.Render(u.ErrorText)
	}

	hint := u.styles.StatusDesc.Render("[Enter] Upload   [Esc] Cancel")

	items := []string{titleRendered, destLabel, "", promptLabel, inputBox}
	if errLine != "" {
		items = append(items, "", errLine)
	}
	items = append(items, "", hint)

	content := lipgloss.JoinVertical(lipgloss.Left, items...)
	box := u.styles.ModalBox.Width(dialogWidth).Render(content)

	return lipgloss.Place(totalWidth, totalHeight, lipgloss.Center, lipgloss.Center, box)
}
