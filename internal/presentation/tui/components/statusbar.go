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
	var left string
	if notification != "" {
		notifText := notification
		maxNotifLen := width * 50 / 100
		if maxNotifLen > 10 && lipgloss.Width(notifText) > maxNotifLen {
			targetW := maxNotifLen - 1
			var curW int
			var runes []rune
			for _, r := range notifText {
				rw := lipgloss.Width(string(r))
				if curW+rw > targetW {
					break
				}
				curW += rw
				runes = append(runes, r)
			}
			notifText = string(runes) + "…"
		}
		isErr := strings.Contains(strings.ToLower(notification), "err") || strings.Contains(strings.ToLower(notification), "fail")
		pillStyle := b.styles.SuccessPill
		prefix := "✔ "
		if isErr {
			pillStyle = b.styles.DangerPill
			prefix = "✖ "
		}
		left = pillStyle.Render(prefix + notifText)
	} else if throughput > 0 {
		speedStr := fmt.Sprintf("⚡ %.2f MiB/s (%.0f%%)", throughput/(1024*1024), progressPercent)
		left = b.styles.SuccessPill.Render(speedStr)
	} else {
		left = lipgloss.NewStyle().Foreground(b.styles.Theme.TextSubtle).Render("⚡ Ready")
	}

	leftWidth := lipgloss.Width(left)

	availForHelp := width - leftWidth - 2
	if availForHelp < 0 {
		availForHelp = 0
	}

	var rightParts []string
	curHelpLen := 0
	for _, h := range activeHelp {
		hLen := lipgloss.Width(h) + 1
		if curHelpLen+hLen > availForHelp && len(rightParts) > 0 {
			break
		}
		rightParts = append(rightParts, b.styles.StatusKey.Render(h))
		curHelpLen += hLen
	}
	right := strings.Join(rightParts, " ")
	rightWidth := lipgloss.Width(right)

	spacerWidth := width - leftWidth - rightWidth - 2
	if spacerWidth < 1 {
		spacerWidth = 1
	}

	spacer := strings.Repeat(" ", spacerWidth)
	line := lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right)
	return b.styles.StatusBar.Width(width).Render(line)
}
