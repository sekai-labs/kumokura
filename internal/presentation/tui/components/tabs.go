package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/styles"
)

type TabItem struct {
	ID    string
	Title string
}

type TabBar struct {
	Tabs   []TabItem
	Active int
	styles styles.Styles
}

func NewTabBar(tabs []TabItem, s styles.Styles) TabBar {
	return TabBar{
		Tabs:   tabs,
		Active: 0,
		styles: s,
	}
}

func (t *TabBar) Render(width int, activeAccount string, activeBucket string) string {
	var tabsRendered []string

	for i, tab := range t.Tabs {
		title := tab.Title
		if width < 90 {
			// Compact titles for smaller terminals (e.g. "1:Exp" or "1:Explorer")
			title = strings.Replace(title, " Explorer", " Exp", 1)
			title = strings.Replace(title, " Transfers", " Xfer", 1)
			title = strings.Replace(title, " Accounts", " Acc", 1)
		}
		if i == t.Active {
			tabsRendered = append(tabsRendered, t.styles.ActiveTab.Render(title))
		} else {
			tabsRendered = append(tabsRendered, t.styles.InactiveTab.Render(title))
		}
	}

	left := lipgloss.JoinHorizontal(lipgloss.Top, tabsRendered...)

	var rightParts []string
	// Only show badges if there is enough space
	if width >= 100 && activeAccount != "" {
		rightParts = append(rightParts, t.styles.BadgeAccount.Render(fmt.Sprintf("Acc: %s", activeAccount)))
	}
	if width >= 80 && activeBucket != "" {
		bucketLabel := activeBucket
		if len(bucketLabel) > 20 && width < 120 {
			bucketLabel = bucketLabel[:19] + "…"
		}
		rightParts = append(rightParts, t.styles.BadgeAccount.Render(fmt.Sprintf("Bucket: %s", bucketLabel)))
	}

	right := lipgloss.JoinHorizontal(lipgloss.Top, rightParts...)

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	spacerWidth := width - leftWidth - rightWidth - 2 // account for padding
	if spacerWidth < 1 {
		spacerWidth = 1
	}

	spacer := strings.Repeat(" ", spacerWidth)
	content := lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right)
	return t.styles.TopBar.Width(width).Render(content)
}
