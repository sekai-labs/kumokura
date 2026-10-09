package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/styles"
	transferDomain "github.com/sekai-labs/kumokura/internal/transfers/domain"
)

type TransfersView struct {
	Jobs        []transferDomain.TransferJob
	SelectedJob int
	styles      styles.Styles
}

func NewTransfersView(s styles.Styles) TransfersView {
	return TransfersView{styles: s}
}

func (v *TransfersView) Render(width, height int) string {
	title := v.styles.PanelTitle.Render(fmt.Sprintf("● TRANSFERS (%d)", len(v.Jobs)))
	var rows []string

	header := fmt.Sprintf("   %-28s %-8s %-12s %-10s %s", "JOB ID", "TYPE", "STATUS", "PROGRESS", "DESTINATION")
	hdrStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(v.styles.Theme.TextSubtle).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(v.styles.Theme.BorderInactive)
	rows = append(rows, hdrStyle.Width(width-4).Render(header))

	maxItems := height - 5
	if maxItems < 1 {
		maxItems = 1
	}

	if len(v.Jobs) == 0 {
		emptyCard := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(v.styles.Theme.BorderInactive).
			Padding(1, 2).
			Foreground(v.styles.Theme.TextMuted).
			Render("No active or historical transfers.\nUpload or download files to track transfer operations.")
		rows = append(rows, "", emptyCard)
	} else {
		for i, j := range v.Jobs {
			if len(rows)-2 >= maxItems {
				break
			}
			prog := "0%"
			if j.TotalBytes > 0 {
				percent := float64(j.BytesTransferred) / float64(j.TotalBytes) * 100
				prog = fmt.Sprintf("%.0f%%", percent)
			}

			jobID := j.ID
			if lipgloss.Width(jobID) > 26 {
				targetW := 25
				var curW int
				var runes []rune
				for _, r := range jobID {
					rw := lipgloss.Width(string(r))
					if curW+rw > targetW {
						break
					}
					curW += rw
					runes = append(runes, r)
				}
				jobID = string(runes) + "…"
			}

			statusStr := string(j.Status)
			if strings.EqualFold(statusStr, "completed") {
				statusStr = "✔ COMPLETED"
			} else if strings.EqualFold(statusStr, "failed") {
				statusStr = "✖ FAILED"
			} else if strings.EqualFold(statusStr, "running") || strings.EqualFold(statusStr, "in_progress") {
				statusStr = "⚡ ACTIVE"
			}

			line := fmt.Sprintf("%-28s %-8s %-12s %-10s %s",
				jobID,
				string(j.Type),
				statusStr,
				prog,
				j.DestinationPath,
			)

			var rowStr string
			if i == v.SelectedJob {
				rowStr = v.styles.SelectedRow.Width(width - 4).Render(fmt.Sprintf(" ▶ %s", line))
			} else {
				rowStr = v.styles.NormalRow.Width(width - 4).Render(fmt.Sprintf("   %s", line))
			}
			rows = append(rows, rowStr)
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(rows, "\n"))
	innerW := width - 2
	if innerW < 10 {
		innerW = 10
	}
	innerH := height - 2
	if innerH < 4 {
		innerH = 4
	}
	return v.styles.ActivePanel.Width(innerW).Height(innerH).Render(content)
}
