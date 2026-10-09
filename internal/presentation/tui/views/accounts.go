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
	title := v.styles.PanelTitle.Render(fmt.Sprintf("CONFIGURED STORAGE ACCOUNTS (%d)", len(v.Accounts)))
	var rows []string

	header := fmt.Sprintf("%-20s %-12s %-16s %s", "NAME", "PROVIDER", "REGION", "ENDPOINT")
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

	if len(v.Accounts) == 0 {
		rows = append(rows, v.styles.StatusDesc.Render("No accounts configured"))
	} else {
		for i, acc := range v.Accounts {
			if len(rows)-2 >= maxItems {
				break
			}
			line := fmt.Sprintf("%-20s %-12s %-16s %s",
				acc.Name,
				string(acc.Type),
				acc.Region,
				acc.Endpoint,
			)

			var rowStr string
			if i == v.SelectedAccount {
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
