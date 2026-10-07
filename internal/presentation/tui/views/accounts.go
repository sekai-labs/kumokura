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

	header := fmt.Sprintf("%-24s %-16s %-20s %s", "NAME", "PROVIDER", "REGION", "ENDPOINT")
	rows = append(rows, v.styles.StatusKey.Render(header))
	rows = append(rows, strings.Repeat("─", width-6))

	for i, acc := range v.Accounts {
		line := fmt.Sprintf("%-24s %-16s %-20s %s",
			acc.Name,
			string(acc.Type),
			acc.Region,
			acc.Endpoint,
		)

		var rowStr string
		if i == v.SelectedAccount {
			rowStr = v.styles.SelectedRow.Width(width - 4).Render("> " + line)
		} else {
			rowStr = v.styles.NormalRow.Width(width - 4).Render("  " + line)
		}
		rows = append(rows, rowStr)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(rows, "\n"))
	return v.styles.ActivePanel.Width(width - 2).Height(height - 2).Render(content)
}
