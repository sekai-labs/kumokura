package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/styles"
)

type StatusBar struct {
	styles styles.Styles
}

func NewStatusBar(s styles.Styles) StatusBar {
	return StatusBar{styles: s}
}

func (b *StatusBar) Render(width int, activeHelp []string, throughput float64, progressPercent float64, notification string) string {
	var leftParts []string

	if notification != "" {
		leftParts = append(leftParts, b.styles.WarningPill.Render(notification))
	} else if throughput > 0 {
		speedStr := fmt.Sprintf("%.2f MiB/s (%.0f%%)", throughput/(1024*1024), progressPercent)
		leftParts = append(leftParts, b.styles.SuccessPill.Render(speedStr))
	} else {
		leftParts = append(leftParts, b.styles.StatusDesc.Render("Ready"))
	}

	left := strings.Join(leftParts, " ")

	var rightParts []string
	for _, h := range activeHelp {
		rightParts = append(rightParts, b.styles.StatusKey.Render(h))
	}
	right := strings.Join(rightParts, "  ")

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	spacerWidth := width - leftWidth - rightWidth - 2
	if spacerWidth < 1 {
		spacerWidth = 1
	}

	spacer := strings.Repeat(" ", spacerWidth)
	line := lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right)
	return b.styles.StatusBar.Width(width).Render(line)
}
