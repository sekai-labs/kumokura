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
	ti.Placeholder = "/path/to/local/file_or_directory"
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

	titleRendered := u.styles.ModalHeader.Render("UPLOAD (FILE OR FOLDER)")
	destLabel := u.styles.StatusKey.Render("Target Destination: ") + u.styles.StatusDesc.Render(u.Destination)
	promptLabel := u.styles.NormalRow.Render("Local path (file or directory):")
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

type DownloadModal struct {
	Input      textinput.Model
	TargetName string
	IsFolder   bool
	Active     bool
	ErrorText  string
	styles     styles.Styles
}

func NewDownloadModal(s styles.Styles) DownloadModal {
	ti := textinput.New()
	ti.Placeholder = "./ (local directory)"
	ti.CharLimit = 512
	ti.Width = 44

	return DownloadModal{
		Input:  ti,
		styles: s,
	}
}

func (d *DownloadModal) Render(totalWidth, totalHeight int) string {
	dialogWidth := 58
	if totalWidth-6 < dialogWidth {
		dialogWidth = totalWidth - 6
	}
	if dialogWidth < 30 {
		dialogWidth = 30
	}

	dialogTitle := "DOWNLOAD OBJECT"
	targetLabelText := "Selected Object: "
	if d.IsFolder {
		dialogTitle = "DOWNLOAD FOLDER (RECURSIVE)"
		targetLabelText = "Selected Folder: "
	}

	titleRendered := d.styles.ModalHeader.Render(dialogTitle)
	sourceLabel := d.styles.StatusKey.Render(targetLabelText) + d.styles.StatusDesc.Render(d.TargetName)
	promptLabel := d.styles.NormalRow.Render("Local destination directory:")
	inputBox := d.Input.View()

	var errLine string
	if d.ErrorText != "" {
		errLine = d.styles.DangerPill.Render(d.ErrorText)
	}

	hint := d.styles.StatusDesc.Render("[Enter] Download   [Esc] Cancel")

	items := []string{titleRendered, sourceLabel, "", promptLabel, inputBox}
	if errLine != "" {
		items = append(items, "", errLine)
	}
	items = append(items, "", hint)

	content := lipgloss.JoinVertical(lipgloss.Left, items...)
	box := d.styles.ModalBox.Width(dialogWidth).Render(content)

	return lipgloss.Place(totalWidth, totalHeight, lipgloss.Center, lipgloss.Center, box)
}

type PresignModal struct {
	Input        textinput.Model
	ObjectKey    string
	Active       bool
	GeneratedURL string
	ErrorText    string
	Copied       bool
	styles       styles.Styles
}

func NewPresignModal(s styles.Styles) PresignModal {
	ti := textinput.New()
	ti.Placeholder = "60m (e.g. 15m, 1h, 24h)"
	ti.SetValue("60m")
	ti.CharLimit = 32
	ti.Width = 24

	return PresignModal{
		Input:  ti,
		styles: s,
	}
}

func (p *PresignModal) Render(totalWidth, totalHeight int) string {
	dialogWidth := 64
	if totalWidth-6 < dialogWidth {
		dialogWidth = totalWidth - 6
	}
	if dialogWidth < 30 {
		dialogWidth = 30
	}

	titleRendered := p.styles.ModalHeader.Render("GENERATE PRESIGNED URL")
	targetLabel := p.styles.StatusKey.Render("Object: ") + p.styles.StatusDesc.Render(p.ObjectKey)

	var items []string
	items = append(items, titleRendered, targetLabel, "")

	if p.GeneratedURL != "" {
		statusMsg := "URL generated and copied via OSC 52!"
		if !p.Copied {
			statusMsg = "URL generated:"
		}
		items = append(items, p.styles.SuccessPill.Render(statusMsg), "")
		items = append(items, p.styles.NormalRow.Render(p.GeneratedURL), "")
		items = append(items, p.styles.StatusDesc.Render("[Enter/Esc] Close"))
	} else {
		promptLabel := p.styles.NormalRow.Render("Expiration duration (e.g. 15m, 1h, 24h):")
		inputBox := p.Input.View()
		items = append(items, promptLabel, inputBox)
		if p.ErrorText != "" {
			items = append(items, "", p.styles.DangerPill.Render(p.ErrorText))
		}
		hint := p.styles.StatusDesc.Render("[Enter] Generate & Copy   [Esc] Cancel")
		items = append(items, "", hint)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, items...)
	box := p.styles.ModalBox.Width(dialogWidth).Render(content)

	return lipgloss.Place(totalWidth, totalHeight, lipgloss.Center, lipgloss.Center, box)
}
