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

func (t *TabBar) Render(width int, activeAccount string, activeRegion string, activeBucket string) string {
	var tabsRendered []string

	for i, tab := range t.Tabs {
		title := tab.Title
		if width < 90 {
			title = strings.Replace(title, " Buckets", " Bkt", 1)
			title = strings.Replace(title, " Objects", " Obj", 1)
			title = strings.Replace(title, " Transfers", " Xfer", 1)
			title = strings.Replace(title, " Sync", " Sync", 1)
		}
		if i == t.Active {
			tabsRendered = append(tabsRendered, t.styles.ActiveTab.Render(title))
		} else {
			tabsRendered = append(tabsRendered, t.styles.InactiveTab.Render(title))
		}
	}

	left := lipgloss.JoinHorizontal(lipgloss.Top, tabsRendered...)

	var rightParts []string
	if width >= 90 && activeAccount != "" {
		profileLabel := fmt.Sprintf("Profile: %s", activeAccount)
		if activeRegion != "" {
			profileLabel = fmt.Sprintf("Profile: %s (%s)", activeAccount, activeRegion)
		}
		rightParts = append(rightParts, t.styles.BadgeAccount.Render(profileLabel))
	}
	if width >= 110 && activeBucket != "" {
		bucketLabel := activeBucket
		if lipgloss.Width(bucketLabel) > 20 && width < 130 {
			targetW := 19
			var curW int
			var runes []rune
			for _, r := range bucketLabel {
				rw := lipgloss.Width(string(r))
				if curW+rw > targetW {
					break
				}
				curW += rw
				runes = append(runes, r)
			}
			bucketLabel = string(runes) + "…"
		}
		rightParts = append(rightParts, t.styles.BadgeAccount.Render(fmt.Sprintf("Bucket: %s", bucketLabel)))
	}

	right := lipgloss.JoinHorizontal(lipgloss.Top, rightParts...)

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	spacerWidth := width - leftWidth - rightWidth - 2
	if spacerWidth < 1 {
		spacerWidth = 1
	}

	spacer := strings.Repeat(" ", spacerWidth)
	content := lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right)
	return t.styles.TopBar.Width(width).Render(content)
}
