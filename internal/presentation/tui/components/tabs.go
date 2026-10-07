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
		if i == t.Active {
			tabsRendered = append(tabsRendered, t.styles.ActiveTab.Render(tab.Title))
		} else {
			tabsRendered = append(tabsRendered, t.styles.InactiveTab.Render(tab.Title))
		}
	}

	left := lipgloss.JoinHorizontal(lipgloss.Top, tabsRendered...)

	var rightParts []string
	if activeAccount != "" {
		rightParts = append(rightParts, t.styles.BadgeAccount.Render(fmt.Sprintf("Acc: %s", activeAccount)))
	}
	if activeBucket != "" {
		rightParts = append(rightParts, t.styles.BadgeAccount.Render(fmt.Sprintf("Bucket: %s", activeBucket)))
	}

	right := lipgloss.JoinHorizontal(lipgloss.Top, rightParts...)

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	spacerWidth := width - leftWidth - rightWidth
	if spacerWidth < 1 {
		spacerWidth = 1
	}

	spacer := lipgloss.NewStyle().Width(spacerWidth).Render(strings.Repeat(" ", spacerWidth))

	return t.styles.TopBar.Width(width).Render(lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right))
}
