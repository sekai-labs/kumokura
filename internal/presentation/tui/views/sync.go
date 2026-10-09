package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/styles"
	syncDomain "github.com/sekai-labs/kumokura/internal/synchronization/domain"
	syncPorts "github.com/sekai-labs/kumokura/internal/synchronization/ports"
)

type SyncView struct {
	Jobs        []*syncPorts.SyncJobRecord
	SelectedJob int
	CurrentPlan *syncDomain.SyncPlan
	styles      styles.Styles
}

func NewSyncView(s styles.Styles) SyncView {
	return SyncView{styles: s}
}

func (v *SyncView) Render(width, height int) string {
	title := v.styles.PanelTitle.Render(fmt.Sprintf("SYNC ENGINE: JOBS & CONFIGURATIONS (%d)", len(v.Jobs)))
	var rows []string

	header := fmt.Sprintf("%-24s %-12s %-20s %-20s %-10s", "JOB ID", "DIRECTION", "SOURCE", "DESTINATION", "STATUS")
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
		rows = append(rows, v.styles.StatusDesc.Render("No active or saved synchronization jobs"))
		rows = append(rows, "")
		rows = append(rows, v.styles.StatusDesc.Render("Use `kumokura sync <source> <target>` to run high-throughput sync operations."))
	} else {
		for i, j := range v.Jobs {
			if len(rows)-2 >= maxItems {
				break
			}

			jobID := j.ID
			if len(jobID) > 22 {
				jobID = jobID[:21] + "…"
			}

			src := j.SourceBucket
			if j.SourcePrefix != "" {
				src += "/" + j.SourcePrefix
			}
			if len(src) > 18 {
				src = src[:17] + "…"
			}

			dst := j.DestinationPath
			if len(dst) > 18 {
				dst = dst[:17] + "…"
			}

			line := fmt.Sprintf("%-24s %-12s %-20s %-20s %-10s",
				jobID,
				string(j.SyncDirection),
				src,
				dst,
				j.Status,
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
