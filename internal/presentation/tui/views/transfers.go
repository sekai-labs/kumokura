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
	title := v.styles.PanelTitle.Render(fmt.Sprintf("ACTIVE & COMPLETED TRANSFERS (%d)", len(v.Jobs)))
	var rows []string

	header := fmt.Sprintf("%-28s %-8s %-12s %-10s %s", "JOB ID", "TYPE", "STATUS", "PROGRESS", "DESTINATION")
	rows = append(rows, v.styles.StatusKey.Render(header))
	dividerLen := width - 6
	if dividerLen < 10 {
		dividerLen = 10
	}
	rows = append(rows, strings.Repeat("─", dividerLen))

	maxItems := height - 6
	if maxItems < 1 {
		maxItems = 1
	}

	if len(v.Jobs) == 0 {
		rows = append(rows, v.styles.StatusDesc.Render("No active or historical transfers"))
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
			if len(jobID) > 26 {
				jobID = jobID[:25] + "…"
			}

			line := fmt.Sprintf("%-28s %-8s %-12s %-10s %s",
				jobID,
				string(j.Type),
				string(j.Status),
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
