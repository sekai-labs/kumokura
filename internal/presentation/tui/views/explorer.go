package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	bucketDomain "github.com/sekai-labs/kumokura/internal/buckets/domain"
	objDomain "github.com/sekai-labs/kumokura/internal/objects/domain"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/styles"
)

type ExplorerView struct {
	Buckets         []bucketDomain.Bucket
	SelectedBucket  int
	Objects         []objDomain.Object
	Prefixes        []objDomain.Prefix
	SelectedObject  int
	CurrentPrefix   string
	PreviewMetadata *objDomain.ObjectMetadata
	PreviewTags     []objDomain.ObjectTag
	PreviewContent  string
	ShowPreview     bool
	ActivePaneIndex int
	styles          styles.Styles
}

func NewExplorerView(s styles.Styles) ExplorerView {
	return ExplorerView{
		ShowPreview:     true,
		ActivePaneIndex: 1,
		styles:          s,
	}
}

func (v *ExplorerView) Render(width, height int) string {
	if width < 80 {
		return lipgloss.NewStyle().Width(width).Height(height).Render("Terminal window too small. Minimum 80x24 required.")
	}

	showPreview := v.ShowPreview && width >= 120

	var leftWidth, centerWidth, rightWidth int
	if showPreview {
		leftWidth = width * 22 / 100
		rightWidth = width * 28 / 100
		centerWidth = width - leftWidth - rightWidth - 6
	} else if width >= 100 {
		leftWidth = width * 28 / 100
		centerWidth = width - leftWidth - 4
	} else {
		leftWidth = width * 32 / 100
		centerWidth = width - leftWidth - 4
	}

	leftPanel := v.renderLeftPanel(leftWidth, height-2)
	centerPanel := v.renderCenterPanel(centerWidth, height-2)

	var panels []string
	panels = append(panels, leftPanel, centerPanel)

	if showPreview {
		rightPanel := v.renderRightPanel(rightWidth, height-2)
		panels = append(panels, rightPanel)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, panels...)
}

func (v *ExplorerView) renderLeftPanel(width, height int) string {
	title := v.styles.PanelTitle.Render(fmt.Sprintf("BUCKETS (%d)", len(v.Buckets)))
	var rows []string

	for i, b := range v.Buckets {
		var rowStr string
		name := b.Name
		if len(name) > width-4 {
			name = fmt.Sprintf("%s…", name[:width-5])
		}
		if i == v.SelectedBucket && v.ActivePaneIndex == 0 {
			rowStr = v.styles.SelectedRow.Width(width - 2).Render(fmt.Sprintf("> %s", name))
		} else {
			rowStr = v.styles.NormalRow.Width(width - 2).Render(fmt.Sprintf("  %s", name))
		}
		rows = append(rows, rowStr)
		if len(rows) >= height-4 {
			break
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(rows, "\n"))
	style := v.styles.InactivePanel
	if v.ActivePaneIndex == 0 {
		style = v.styles.ActivePanel
	}
	return style.Width(width).Height(height).Render(content)
}

func (v *ExplorerView) renderCenterPanel(width, height int) string {
	totalItems := len(v.Prefixes) + len(v.Objects)
	titleText := fmt.Sprintf("OBJECTS: /%s (%d)", v.CurrentPrefix, totalItems)
	title := v.styles.PanelTitle.Render(titleText)

	var rows []string

	for i, p := range v.Prefixes {
		var rowStr string
		name := p.Prefix
		if len(name) > width-15 {
			name = fmt.Sprintf("%s…", name[:width-16])
		}
		display := fmt.Sprintf("📁 %-30s [DIR]", name)
		if i == v.SelectedObject && v.ActivePaneIndex == 1 {
			rowStr = v.styles.SelectedRow.Width(width - 2).Render(fmt.Sprintf("> %s", display))
		} else {
			rowStr = v.styles.NormalRow.Width(width - 2).Render(fmt.Sprintf("  %s", display))
		}
		rows = append(rows, rowStr)
		if len(rows) >= height-4 {
			break
		}
	}

	prefixOffset := len(v.Prefixes)
	for i, obj := range v.Objects {
		if len(rows) >= height-4 {
			break
		}
		idx := prefixOffset + i
		name := obj.Key
		if v.CurrentPrefix != "" {
			name = strings.TrimPrefix(name, v.CurrentPrefix)
		}
		if len(name) > width-25 {
			name = fmt.Sprintf("%s…", name[:width-26])
		}

		sizeStr := formatBytes(obj.Size)
		display := fmt.Sprintf("%-28s %10s %s", name, sizeStr, string(obj.StorageClass))
		var rowStr string
		if idx == v.SelectedObject && v.ActivePaneIndex == 1 {
			rowStr = v.styles.SelectedRow.Width(width - 2).Render(fmt.Sprintf("> %s", display))
		} else {
			rowStr = v.styles.NormalRow.Width(width - 2).Render(fmt.Sprintf("  %s", display))
		}
		rows = append(rows, rowStr)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(rows, "\n"))
	style := v.styles.InactivePanel
	if v.ActivePaneIndex == 1 {
		style = v.styles.ActivePanel
	}
	return style.Width(width).Height(height).Render(content)
}

func (v *ExplorerView) renderRightPanel(width, height int) string {
	title := v.styles.PanelTitle.Render("DETAILS & PREVIEW")
	var lines []string

	if v.PreviewMetadata != nil {
		lines = append(lines, fmt.Sprintf("%s%s", v.styles.StatusKey.Render("Type: "), v.PreviewMetadata.ContentType))
		lines = append(lines, fmt.Sprintf("%s%s", v.styles.StatusKey.Render("Size: "), formatBytes(v.PreviewMetadata.ContentLength)))
		lines = append(lines, fmt.Sprintf("%s%s", v.styles.StatusKey.Render("ETag: "), v.PreviewMetadata.ETag))
		lines = append(lines, fmt.Sprintf("%s%s", v.styles.StatusKey.Render("Class: "), string(v.PreviewMetadata.StorageClass)))
		lines = append(lines, fmt.Sprintf("%s%s", v.styles.StatusKey.Render("Modified: "), v.PreviewMetadata.LastModified.Format("2006-01-02 15:04")))

		if len(v.PreviewTags) > 0 {
			lines = append(lines, "")
			lines = append(lines, v.styles.PanelTitle.Render("Tags:"))
			for _, tag := range v.PreviewTags {
				lines = append(lines, fmt.Sprintf("  %s = %s", tag.Key, tag.Value))
			}
		}

		if v.PreviewContent != "" {
			lines = append(lines, "")
			lines = append(lines, v.styles.PanelTitle.Render("Content Preview:"))
			previewSnippet := v.PreviewContent
			if len(previewSnippet) > 300 {
				previewSnippet = fmt.Sprintf("%s...", previewSnippet[:300])
			}
			lines = append(lines, v.styles.StatusDesc.Render(previewSnippet))
		}
	} else {
		lines = append(lines, v.styles.StatusDesc.Render("Select an object to inspect details"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(lines, "\n"))
	style := v.styles.InactivePanel
	if v.ActivePaneIndex == 2 {
		style = v.styles.ActivePanel
	}
	return style.Width(width).Height(height).Render(content)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
