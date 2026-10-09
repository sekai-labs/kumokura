package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	accDomain "github.com/sekai-labs/kumokura/internal/accounts/domain"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/styles"
)

type AccountsView struct {
	Accounts        []accDomain.Account
	SelectedAccount int
	styles          styles.Styles
}

func NewAccountsView(s styles.Styles) AccountsView {
	return AccountsView{styles: s}
}

func (v *AccountsView) Render(width, height int) string {
	title := v.styles.PanelTitle.Render(fmt.Sprintf("● CONNECTED STORAGE PROFILES (%d)", len(v.Accounts)))
	var rows []string

	header := fmt.Sprintf("   %-24s %-16s %-16s %s", "ACCOUNT NAME", "PROVIDER TYPE", "REGION", "ENDPOINT")
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

	if len(v.Accounts) == 0 {
		emptyCard := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(v.styles.Theme.BorderInactive).
			Padding(1, 2).
			Foreground(v.styles.Theme.TextMuted).
			Render("No configured cloud profiles found.\nRun 'kumokura account add' to configure AWS/S3 credentials.")
		rows = append(rows, "", emptyCard)
	} else {
		for i, a := range v.Accounts {
			if len(rows)-2 >= maxItems {
				break
			}
			endpoint := a.Endpoint
			if endpoint == "" {
				endpoint = "-"
			}
			line := fmt.Sprintf("%-24s %-16s %-16s %s",
				a.Name,
				string(a.Type),
				a.Region,
				endpoint,
			)
			var rowStr string
			if i == v.SelectedAccount {
				rowStr = v.styles.SelectedRow.Width(width - 4).Render(fmt.Sprintf(" ▶ 👤 %s", line))
			} else {
				rowStr = v.styles.NormalRow.Width(width - 4).Render(fmt.Sprintf("   👤 %s", line))
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
