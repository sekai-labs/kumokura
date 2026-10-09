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
		"  Tab / Shift+Tab     Cycle active focus between left and right panes",
		"  1, 2, 3, 4          Directly jump to Tab (1: Buckets, 2: Objects, 3: Transfers, 4: Sync)",
		"  ?                   Toggle this in-app Help overlay modal",
		"  Escape              Dismiss modal, clear search filter, or unfocus input",
		"  r                   Force refresh active pane from remote S3 API",
		"  q, Ctrl+C           Gracefully quit Kumokura TUI",
		"",
		"List & Directory Navigation:",
		"  j / Down, k / Up    Move cursor down / up one row",
		"  g / G               Jump to very top / bottom of list",
		"  Ctrl+d / Ctrl+u     Half-page scroll down / up",
		"  Enter               Open selected bucket or descend into directory prefix",
		"  Backspace / h       Ascend to parent directory (../) or bucket root",
		"  /                   Open live fuzzy search filter input",
		"",
		"Object & Transfer Actions:",
		"  u                   Upload a file or directory from local disk to current prefix",
		"  d                   Download selected object(s) or folder to local filesystem",
		"  x, Delete           Prompt confirmation to delete selected object or bucket",
		"  p                   Generate presigned URL dialog (configurable expiration, OSC 52)",
		"  i                   Toggle Object Metadata & Tags Inspector Drawer",
	}

	innerW := width - 4
	if innerW < 20 {
		innerW = 20
	}
	innerH := height - 4
	if innerH < 10 {
		innerH = 10
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", strings.Join(helpLines, "\n"))
	return v.styles.ModalBox.Width(innerW).Height(innerH).Render(content)
}
