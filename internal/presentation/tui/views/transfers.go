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

	header := fmt.Sprintf("%-36s %-10s %-12s %-12s %s", "JOB ID", "TYPE", "STATUS", "PROGRESS", "DESTINATION")
	rows = append(rows, v.styles.StatusKey.Render(header))
	rows = append(rows, strings.Repeat("─", width-6))

	for i, j := range v.Jobs {
		prog := "0%"
		if j.TotalBytes > 0 {
			percent := float64(j.BytesTransferred) / float64(j.TotalBytes) * 100
			prog = fmt.Sprintf("%.0f%%", percent)
		}

		line := fmt.Sprintf("%-36s %-10s %-12s %-12s %s",
			j.ID,
			string(j.Type),
			string(j.Status),
			prog,
			j.DestinationPath,
		)

		var rowStr string
		if i == v.SelectedJob {
			rowStr = v.styles.SelectedRow.Width(width - 4).Render("> " + line)
		} else {
			rowStr = v.styles.NormalRow.Width(width - 4).Render("  " + line)
		}
		rows = append(rows, rowStr)
		if len(rows) >= height-4 {
			break
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(rows, "\n"))
	return v.styles.ActivePanel.Width(width - 2).Height(height - 2).Render(content)
}
