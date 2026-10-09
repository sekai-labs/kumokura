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
	title := v.styles.PanelTitle.Render(fmt.Sprintf("● SYNC ENGINE (%d)", len(v.Jobs)))
	var rows []string

	header := fmt.Sprintf("   %-24s %-12s %-20s %-20s %-10s", "JOB ID", "DIRECTION", "SOURCE", "DESTINATION", "STATUS")
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
			Render("No active or saved synchronization jobs.\nRun 'kumokura sync <source> <target>' from the CLI for bidirectional sync.")
		rows = append(rows, "", emptyCard)
	} else {
		for i, j := range v.Jobs {
			if len(rows)-2 >= maxItems {
				break
			}

			jobID := j.ID
			if lipgloss.Width(jobID) > 22 {
				targetW := 21
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

			src := j.SourceBucket
			if j.SourcePrefix != "" {
				src += "/" + j.SourcePrefix
			}
			if lipgloss.Width(src) > 18 {
				targetW := 17
				var curW int
				var runes []rune
				for _, r := range src {
					rw := lipgloss.Width(string(r))
					if curW+rw > targetW {
						break
					}
					curW += rw
					runes = append(runes, r)
				}
				src = string(runes) + "…"
			}

			dst := j.DestinationPath
			if lipgloss.Width(dst) > 18 {
				targetW := 17
				var curW int
				var runes []rune
				for _, r := range dst {
					rw := lipgloss.Width(string(r))
					if curW+rw > targetW {
						break
					}
					curW += rw
					runes = append(runes, r)
				}
				dst = string(runes) + "…"
			}

			statusStr := string(j.Status)
			if strings.EqualFold(statusStr, "completed") {
				statusStr = "✔ COMPLETED"
			} else if strings.EqualFold(statusStr, "failed") {
				statusStr = "✖ FAILED"
			} else if strings.EqualFold(statusStr, "running") {
				statusStr = "⚡ SYNCING"
			}

			line := fmt.Sprintf("%-24s %-12s %-20s %-20s %-10s",
				jobID,
				string(j.SyncDirection),
				src,
				dst,
				statusStr,
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
