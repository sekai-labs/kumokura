package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/styles"
)

type HelpView struct {
	styles styles.Styles
}

func NewHelpView(s styles.Styles) HelpView {
	return HelpView{styles: s}
}

func (v *HelpView) Render(width, height int) string {
	title := v.styles.ModalHeader.Render("KUMOKURA KEYBOARD SHORTCUTS & HELP")

	helpLines := []string{
		"Global Navigation:",
		"  Tab / Shift+Tab     Switch focus between panels (Buckets / Objects / Preview)",
		"  1 - 5               Switch main tabs (1: Explorer, 2: Transfers, 3: Sync, 4: Accounts, 5: Help)",
		"  /                   Filter objects in active prefix",
		"  p                   Toggle details & content preview pane",
		"  r                   Refresh active view",
		"  ?                   Open this help modal",
		"  q, Ctrl+C           Quit Kumokura TUI",
		"",
		"List & Tree Navigation:",
		"  j / k, ↓ / ↑        Move cursor down / up",
		"  g / G               Jump to top / bottom of list",
		"  Ctrl+d / Ctrl+u     Half-page scroll down / up",
		"  Enter               Open bucket, drill into folder, select item",
		"  Backspace, Esc      Navigate up to parent folder / clear filter",
		"",
		"Object Operations:",
		"  u                   Upload local file to current prefix",
		"  d                   Download selected object to local directory",
		"  D                   Delete selected object (triggers confirmation modal)",
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", strings.Join(helpLines, "\n"))
	return v.styles.ModalBox.Width(width - 4).Height(height - 4).Render(content)
}
