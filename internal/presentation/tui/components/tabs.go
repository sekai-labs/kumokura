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
	titleBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.styles.Theme.BorderActive).
		Padding(0, 1).
		Render("☁  KUMOKURA")

	var tabsRendered []string
	for i, tab := range t.Tabs {
		title := tab.Title
		if width < 90 {
			title = strings.Replace(title, " Buckets", " Bkt", 1)
			title = strings.Replace(title, " Objects", " Obj", 1)
			title = strings.Replace(title, " Transfers", " Xfer", 1)
			title = strings.Replace(title, " Sync", " Sync", 1)
		}
		pillText := title
		if !strings.HasPrefix(pillText, "[") {
			pillText = "[" + pillText + "]"
		}
		pillText = fmt.Sprintf(" %s ", pillText)
		if i == t.Active {
			tabsRendered = append(tabsRendered, t.styles.ActiveTab.Render(pillText))
		} else {
			tabsRendered = append(tabsRendered, t.styles.InactiveTab.Render(pillText))
		}
	}

	tabsBar := lipgloss.JoinHorizontal(lipgloss.Top, strings.Join(tabsRendered, " "))
	left := lipgloss.JoinHorizontal(lipgloss.Center, titleBadge, " ", tabsBar)

	var rightParts []string
	if width >= 80 && activeAccount != "" {
		profileBadge := lipgloss.NewStyle().
			Background(t.styles.Theme.HighlightRow).
			Foreground(t.styles.Theme.AccentSky).
			Bold(true).
			Padding(0, 1).
			Render("👤 " + activeAccount)
		rightParts = append(rightParts, profileBadge)
	}
	if width >= 95 && activeRegion != "" {
		regionBadge := lipgloss.NewStyle().
			Background(t.styles.Theme.HighlightRow).
			Foreground(t.styles.Theme.BadgeWarning).
			Padding(0, 1).
			Render("🌐 " + activeRegion)
		rightParts = append(rightParts, regionBadge)
	}
	if width >= 115 && activeBucket != "" {
		bucketLabel := activeBucket
		if lipgloss.Width(bucketLabel) > 20 && width < 135 {
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
		bucketBadge := lipgloss.NewStyle().
			Background(t.styles.Theme.HighlightRow).
			Foreground(t.styles.Theme.BadgeSuccess).
			Padding(0, 1).
			Render("🪣 " + bucketLabel)
		rightParts = append(rightParts, bucketBadge)
	}

	right := strings.Join(rightParts, " ")

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
