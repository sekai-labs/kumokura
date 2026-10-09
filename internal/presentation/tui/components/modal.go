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
	dialogWidth := 54
	if totalWidth-6 < dialogWidth {
		dialogWidth = totalWidth - 6
	}
	if dialogWidth < 30 {
		dialogWidth = 30
	}

	headerBox := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.styles.Theme.BorderActive).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(m.styles.Theme.BorderInactive).
		Padding(0, 0, 1, 0).
		Width(dialogWidth - 4).
		Render("◆ " + m.Title)

	msgRendered := m.styles.ModalBody.Width(dialogWidth - 4).Render(m.Message)

	var optsRendered []string
	for i, opt := range m.Options {
		if i == m.SelectedOpt {
			btn := lipgloss.NewStyle().
				Bold(true).
				Background(m.styles.Theme.BorderActive).
				Foreground(m.styles.Theme.CardBackground).
				Padding(0, 2).
				Render(opt)
			optsRendered = append(optsRendered, btn)
		} else {
			btn := lipgloss.NewStyle().
				Background(m.styles.Theme.HighlightRow).
				Foreground(m.styles.Theme.TextMuted).
				Padding(0, 2).
				Render(opt)
			optsRendered = append(optsRendered, btn)
		}
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(optsRendered, "  "))

	content := lipgloss.JoinVertical(lipgloss.Left, headerBox, "", msgRendered, "", buttons)
	box := m.styles.ModalBox.Width(dialogWidth).Render(content)

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
	dialogWidth := 60
	if totalWidth-6 < dialogWidth {
		dialogWidth = totalWidth - 6
	}
	if dialogWidth < 30 {
		dialogWidth = 30
	}

	headerBox := lipgloss.NewStyle().
		Bold(true).
		Foreground(u.styles.Theme.BorderActive).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(u.styles.Theme.BorderInactive).
		Padding(0, 0, 1, 0).
		Width(dialogWidth - 4).
		Render("📤 UPLOAD (FILE OR FOLDER)")

	destLabel := lipgloss.NewStyle().Foreground(u.styles.Theme.TextSubtle).Render("Target S3 Destination:")
	destVal := lipgloss.NewStyle().Foreground(u.styles.Theme.AccentSky).Bold(true).Render(u.Destination)
	destBox := lipgloss.JoinVertical(lipgloss.Left, destLabel, destVal)

	promptLabel := lipgloss.NewStyle().Foreground(u.styles.Theme.TextPrimary).Render("Local path (file or directory):")
	inputFrame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(u.styles.Theme.BorderActive).
		Padding(0, 1).
		Width(dialogWidth - 6).
		Render(u.Input.View())

	var errLine string
	if u.ErrorText != "" {
		errLine = u.styles.DangerPill.Render("✖ " + u.ErrorText)
	}

	keyEnter := u.styles.StatusKey.Render("[Enter]")
	descEnter := u.styles.StatusDesc.Render(" Start Upload   ")
	keyEsc := u.styles.StatusKey.Render("[Esc]")
	descEsc := u.styles.StatusDesc.Render(" Cancel")
	hint := lipgloss.JoinHorizontal(lipgloss.Left, keyEnter, descEnter, keyEsc, descEsc)

	items := []string{headerBox, "", destBox, "", promptLabel, inputFrame}
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
	dialogWidth := 60
	if totalWidth-6 < dialogWidth {
		dialogWidth = totalWidth - 6
	}
	if dialogWidth < 30 {
		dialogWidth = 30
	}

	dialogTitle := "📥 DOWNLOAD OBJECT"
	targetLabelText := "Selected Object: "
	if d.IsFolder {
		dialogTitle = "📥 DOWNLOAD FOLDER (RECURSIVE)"
		targetLabelText = "Selected Folder: "
	}

	headerBox := lipgloss.NewStyle().
		Bold(true).
		Foreground(d.styles.Theme.BorderActive).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(d.styles.Theme.BorderInactive).
		Padding(0, 0, 1, 0).
		Width(dialogWidth - 4).
		Render(dialogTitle)

	sourceLabel := lipgloss.NewStyle().Foreground(d.styles.Theme.TextSubtle).Render(targetLabelText)
	sourceVal := lipgloss.NewStyle().Foreground(d.styles.Theme.AccentSky).Bold(true).Render(d.TargetName)
	sourceBox := lipgloss.JoinVertical(lipgloss.Left, sourceLabel, sourceVal)

	promptLabel := lipgloss.NewStyle().Foreground(d.styles.Theme.TextPrimary).Render("Local destination directory:")
	inputFrame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(d.styles.Theme.BorderActive).
		Padding(0, 1).
		Width(dialogWidth - 6).
		Render(d.Input.View())

	var errLine string
	if d.ErrorText != "" {
		errLine = d.styles.DangerPill.Render("✖ " + d.ErrorText)
	}

	keyEnter := d.styles.StatusKey.Render("[Enter]")
	descEnter := d.styles.StatusDesc.Render(" Start Download   ")
	keyEsc := d.styles.StatusKey.Render("[Esc]")
	descEsc := d.styles.StatusDesc.Render(" Cancel")
	hint := lipgloss.JoinHorizontal(lipgloss.Left, keyEnter, descEnter, keyEsc, descEsc)

	items := []string{headerBox, "", sourceBox, "", promptLabel, inputFrame}
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
	dialogWidth := 66
	if totalWidth-6 < dialogWidth {
		dialogWidth = totalWidth - 6
	}
	if dialogWidth < 30 {
		dialogWidth = 30
	}

	headerBox := lipgloss.NewStyle().
		Bold(true).
		Foreground(p.styles.Theme.BorderActive).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(p.styles.Theme.BorderInactive).
		Padding(0, 0, 1, 0).
		Width(dialogWidth - 4).
		Render("🔗 GENERATE PRESIGNED URL")

	targetLabel := lipgloss.NewStyle().Foreground(p.styles.Theme.TextSubtle).Render("Target Object(s):")
	targetVal := lipgloss.NewStyle().Foreground(p.styles.Theme.AccentSky).Bold(true).Render(p.ObjectKey)
	targetBox := lipgloss.JoinVertical(lipgloss.Left, targetLabel, targetVal)

	var items []string
	items = append(items, headerBox, "", targetBox, "")

	if p.GeneratedURL != "" {
		statusMsg := "✔ URL copied to system clipboard via OSC 52!"
		if !p.Copied {
			statusMsg = "✔ Presigned URL ready:"
		}
		urlBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.styles.Theme.BorderActive).
			Padding(0, 1).
			Width(dialogWidth - 6).
			Render(p.GeneratedURL)

		closeHint := lipgloss.JoinHorizontal(lipgloss.Left,
			p.styles.StatusKey.Render("[Enter/Esc]"),
			p.styles.StatusDesc.Render(" Close"),
		)
		items = append(items, p.styles.SuccessPill.Render(statusMsg), "", urlBox, "", closeHint)
	} else {
		promptLabel := lipgloss.NewStyle().Foreground(p.styles.Theme.TextPrimary).Render("Expiration duration (e.g. 15m, 1h, 24h):")
		inputFrame := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.styles.Theme.BorderActive).
			Padding(0, 1).
			Width(dialogWidth - 6).
			Render(p.Input.View())

		items = append(items, promptLabel, inputFrame)
		if p.ErrorText != "" {
			items = append(items, "", p.styles.DangerPill.Render("✖ "+p.ErrorText))
		}
		keyEnter := p.styles.StatusKey.Render("[Enter]")
		descEnter := p.styles.StatusDesc.Render(" Generate & Copy   ")
		keyEsc := p.styles.StatusKey.Render("[Esc]")
		descEsc := p.styles.StatusDesc.Render(" Cancel")
		hint := lipgloss.JoinHorizontal(lipgloss.Left, keyEnter, descEnter, keyEsc, descEsc)
		items = append(items, "", hint)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, items...)
	box := p.styles.ModalBox.Width(dialogWidth).Render(content)

	return lipgloss.Place(totalWidth, totalHeight, lipgloss.Center, lipgloss.Center, box)
}
