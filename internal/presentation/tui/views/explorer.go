package views

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	bucketDomain "github.com/sekai-labs/kumokura/internal/buckets/domain"
	objDomain "github.com/sekai-labs/kumokura/internal/objects/domain"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/styles"
)

type ExplorerView struct {
	Buckets         []bucketDomain.Bucket
	SelectedBucket  int
	BucketOffset    int
	ActiveBucket    string
	Objects         []objDomain.Object
	Prefixes        []objDomain.Prefix
	SelectedObject  int
	ObjectOffset    int
	CurrentPrefix   string
	PreviewMetadata *objDomain.ObjectMetadata
	PreviewTags     []objDomain.ObjectTag
	PreviewContent  []byte
	PreviewScroll   int
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

func (v ExplorerView) Render(width, height int) string {
	if width < 80 || height < 24 {
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Terminal window too small: Kumokura requires minimum 80x24")
	}

	showPreview := v.ShowPreview && (width >= 160 || (width >= 120 && v.ShowPreview))

	var leftInnerW, centerInnerW, rightInnerW int
	if showPreview && width >= 160 {
		avail := width - 6
		leftInnerW = max(24, avail*20/100)
		rightInnerW = max(30, avail*30/100)
		centerInnerW = avail - leftInnerW - rightInnerW
	} else if showPreview && width >= 120 {
		avail := width - 6
		leftInnerW = max(22, avail*25/100)
		rightInnerW = max(30, avail*30/100)
		centerInnerW = avail - leftInnerW - rightInnerW
		if centerInnerW < 30 {
			showPreview = false
		}
	}

	if !showPreview {
		avail := width - 4
		leftInnerW = max(24, avail*35/100)
		centerInnerW = avail - leftInnerW
	}

	panelInnerH := height - 2
	if panelInnerH < 4 {
		panelInnerH = 4
	}

	leftPanel := v.renderLeftPanel(leftInnerW, panelInnerH)
	centerPanel := v.renderCenterPanel(centerInnerW, panelInnerH)

	panels := []string{leftPanel, centerPanel}
	if showPreview {
		rightPanel := v.renderRightPanel(rightInnerW, panelInnerH)
		panels = append(panels, rightPanel)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, panels...)
}

func (v ExplorerView) renderLeftPanel(width, height int) string {
	isActive := v.ActivePaneIndex == 0
	prefixText := ""
	if isActive {
		prefixText = "ACTIVE PANE: "
	}
	headerTitle := fmt.Sprintf("%sBUCKETS (%d)", prefixText, len(v.Buckets))
	title := v.styles.PanelTitle.Render(headerTitle)
	maxItems := height - 2
	if maxItems < 1 {
		maxItems = 1
	}

	selectedBucket := v.SelectedBucket
	if selectedBucket >= len(v.Buckets) {
		selectedBucket = max(0, len(v.Buckets)-1)
	}

	bucketOffset := v.BucketOffset
	if selectedBucket < bucketOffset {
		bucketOffset = selectedBucket
	} else if selectedBucket >= bucketOffset+maxItems {
		bucketOffset = selectedBucket - maxItems + 1
	}
	if bucketOffset < 0 {
		bucketOffset = 0
	}

	var rows []string
	if len(v.Buckets) == 0 {
		emptyMsg := v.styles.StatusDesc.Render("No buckets found")
		rows = append(rows, emptyMsg)
	} else {
		endIdx := min(len(v.Buckets), bucketOffset+maxItems)
		for i := bucketOffset; i < endIdx; i++ {
			b := v.Buckets[i]
			name := b.Name
			availTextW := width - 4
			if availTextW < 4 {
				availTextW = 4
			}
			name = truncateString(name, availTextW)

			var rowStr string
			if i == selectedBucket && isActive {
				rowStr = v.styles.SelectedRow.Width(width).Render(fmt.Sprintf(" ▶ %s", name))
			} else if i == selectedBucket {
				rowStr = v.styles.SelectedRow.Width(width).Render(fmt.Sprintf(" ▷ %s", name))
			} else {
				rowStr = v.styles.NormalRow.Width(width).Render(fmt.Sprintf("   %s", name))
			}
			rows = append(rows, rowStr)
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(rows, "\n"))
	style := v.styles.InactivePanel
	if isActive {
		style = v.styles.ActivePanel
	}
	return style.Width(width).Height(height).Render(content)
}

func (v ExplorerView) renderCenterPanel(width, height int) string {
	isActive := v.ActivePaneIndex == 1
	bucketName := v.ActiveBucket
	if bucketName == "" && len(v.Buckets) > v.SelectedBucket {
		bucketName = v.Buckets[v.SelectedBucket].Name
	}
	s3Uri := fmt.Sprintf("s3://%s/%s", bucketName, v.CurrentPrefix)
	availTitle := width - 20
	if availTitle > 10 && lipgloss.Width(s3Uri) > availTitle {
		s3Uri = truncateLeadingString(s3Uri, availTitle)
	}
	titleText := fmt.Sprintf("OBJECT EXPLORER: %s", s3Uri)
	title := v.styles.PanelTitle.Render(titleText)
	totalItems := len(v.Prefixes) + len(v.Objects)
	maxItems := height - 2
	if maxItems < 1 {
		maxItems = 1
	}

	selectedObject := v.SelectedObject
	if selectedObject >= totalItems {
		selectedObject = max(0, totalItems-1)
	}

	objectOffset := v.ObjectOffset
	if selectedObject < objectOffset {
		objectOffset = selectedObject
	} else if selectedObject >= objectOffset+maxItems {
		objectOffset = selectedObject - maxItems + 1
	}
	if objectOffset < 0 {
		objectOffset = 0
	}

	var rows []string
	keyColW := width - 26
	if keyColW < 12 {
		keyColW = 12
	}
	tblHeader := fmt.Sprintf("   %-*s %9s %12s", keyColW, "Key", "Size", "Last Modified")
	rows = append(rows, v.styles.StatusKey.Render(tblHeader))
	maxItems = max(1, maxItems-1)

	if totalItems == 0 {
		emptyMsg := v.styles.StatusDesc.Render("   Prefix is empty (or no objects match filter)")
		rows = append(rows, emptyMsg)
	} else {
		prefixLen := len(v.Prefixes)
		endIdx := min(totalItems, objectOffset+maxItems)

		for i := objectOffset; i < endIdx; i++ {
			var display string
			if i < prefixLen {
				p := v.Prefixes[i]
				name := p.Prefix
				if v.CurrentPrefix != "" {
					name = strings.TrimPrefix(name, v.CurrentPrefix)
				}
				maxNameW := width - 26
				if maxNameW < 8 {
					maxNameW = 8
				}
				name = truncateString(name, maxNameW)
				display = fmt.Sprintf("📁 %-*s %9s %12s", maxNameW, name, "[DIR]", "-")
			} else {
				objIdx := i - prefixLen
				obj := v.Objects[objIdx]
				name := obj.Key
				if v.CurrentPrefix != "" {
					name = strings.TrimPrefix(name, v.CurrentPrefix)
				}
				sizeStr := formatBytes(obj.Size)
				lastModStr := "-"
				if !obj.LastModified.IsZero() {
					lastModStr = obj.LastModified.Format("2006-01-02")
				}

				nameWidth := width - 26
				if nameWidth < 8 {
					nameWidth = 8
				}
				name = truncateString(name, nameWidth)

				display = fmt.Sprintf("📄 %-*s %9s %12s", nameWidth, name, sizeStr, lastModStr)
			}

			var rowStr string
			if i == selectedObject && isActive {
				rowStr = v.styles.SelectedRow.Width(width).Render(fmt.Sprintf(" ▶ %s", display))
			} else if i == selectedObject {
				rowStr = v.styles.SelectedRow.Width(width).Render(fmt.Sprintf(" ▷ %s", display))
			} else {
				rowStr = v.styles.NormalRow.Width(width).Render(fmt.Sprintf("   %s", display))
			}
			rows = append(rows, rowStr)
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(rows, "\n"))
	style := v.styles.InactivePanel
	if isActive {
		style = v.styles.ActivePanel
	}
	return style.Width(width).Height(height).Render(content)
}

func (v ExplorerView) renderRightPanel(width, height int) string {
	isActive := v.ActivePaneIndex == 2
	title := v.styles.PanelTitle.Render("OBJECT INSPECTOR")
	var lines []string

	if v.PreviewMetadata != nil {
		lines = append(lines, fmt.Sprintf("%s%s", v.styles.StatusKey.Render("Type:     "), truncateString(v.PreviewMetadata.ContentType, width-12)))
		lines = append(lines, fmt.Sprintf("%s%s", v.styles.StatusKey.Render("Size:     "), formatBytes(v.PreviewMetadata.ContentLength)))
		lines = append(lines, fmt.Sprintf("%s%s", v.styles.StatusKey.Render("ETag:     "), truncateString(v.PreviewMetadata.ETag, width-12)))
		storageClass := string(v.PreviewMetadata.StorageClass)
		if storageClass == "" {
			storageClass = "STANDARD"
		}
		lines = append(lines, fmt.Sprintf("%s%s", v.styles.StatusKey.Render("Class:    "), storageClass))
		lines = append(lines, fmt.Sprintf("%s%s", v.styles.StatusKey.Render("Modified: "), v.PreviewMetadata.LastModified.Format("2006-01-02 15:04:05")))

		if len(v.PreviewTags) > 0 {
			lines = append(lines, "")
			lines = append(lines, v.styles.PanelTitle.Render("TAGS:"))
			for _, tag := range v.PreviewTags {
				tagLine := fmt.Sprintf("  • %s = %s", tag.Key, tag.Value)
				lines = append(lines, truncateString(tagLine, width-4))
			}
		}

		if len(v.PreviewContent) > 0 {
			lines = append(lines, "")
			lines = append(lines, v.styles.PanelTitle.Render("PREVIEW:"))
			sanitizedLines := SanitizePreviewContent(v.PreviewContent, width-4, 15)
			for _, sl := range sanitizedLines {
				lines = append(lines, v.styles.StatusDesc.Render(sl))
			}
		}
	} else {
		lines = append(lines, v.styles.StatusDesc.Render("Select an object to inspect details"))
		lines = append(lines, "")
		lines = append(lines, v.styles.StatusDesc.Render("Press [i] to toggle Inspector drawer"))
		lines = append(lines, v.styles.StatusDesc.Render("Press [p] to generate presigned URL"))
		lines = append(lines, v.styles.StatusDesc.Render("Press [u] to upload file/folder"))
		lines = append(lines, v.styles.StatusDesc.Render("Press [d] to download object/folder"))
		lines = append(lines, v.styles.StatusDesc.Render("Press [x] to delete selected object"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(lines, "\n"))
	style := v.styles.InactivePanel
	if isActive {
		style = v.styles.ActivePanel
	}
	return style.Width(width).Height(height).Render(content)
}

func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	targetWidth := maxLen - 1
	var curWidth int
	var runes []rune
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if curWidth+rw > targetWidth {
			break
		}
		curWidth += rw
		runes = append(runes, r)
	}
	return string(runes) + "…"
}

func truncateLeadingString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	targetWidth := maxLen - 1
	runes := []rune(s)
	var curWidth int
	var chosen []rune
	for i := len(runes) - 1; i >= 0; i-- {
		rw := lipgloss.Width(string(runes[i]))
		if curWidth+rw > targetWidth {
			break
		}
		curWidth += rw
		chosen = append([]rune{runes[i]}, chosen...)
	}
	return "…" + string(chosen)
}

func TruncateSafe(s string, maxLen int) string {
	return truncateString(s, maxLen)
}

func SanitizePreviewContent(data []byte, maxWidth, maxLines int) []string {
	if len(data) == 0 {
		return []string{"[Empty object]"}
	}

	if maxWidth < 10 {
		maxWidth = 10
	}
	if maxLines < 3 {
		maxLines = 3
	}

	isBinary := !utf8.Valid(data) || bytes.IndexByte(data, 0) != -1

	if isBinary {
		var lines []string
		lines = append(lines, fmt.Sprintf("[Binary file - %d bytes shown in hex]", min(len(data), 128)))
		hexRows := min(len(data)/16+1, maxLines-1)
		for row := range hexRows {
			start := row * 16
			if start >= len(data) {
				break
			}
			end := min(len(data), start+16)
			chunk := data[start:end]

			var hexPart strings.Builder
			var asciiPart strings.Builder
			for _, b := range chunk {
				fmt.Fprintf(&hexPart, "%02x ", b)
				if b >= 32 && b < 127 {
					asciiPart.WriteByte(b)
				} else {
					asciiPart.WriteByte('.')
				}
			}
			for i := len(chunk); i < 16; i++ {
				hexPart.WriteString("   ")
			}

			line := fmt.Sprintf("%04x  %s |%s|", start, hexPart.String(), asciiPart.String())
			lines = append(lines, truncateString(line, maxWidth))
		}
		return lines
	}

	text := string(data)
	rawLines := strings.Split(text, "\n")
	var cleanLines []string

	for _, rl := range rawLines {
		if len(cleanLines) >= maxLines {
			cleanLines = append(cleanLines, "...")
			break
		}
		var b strings.Builder
		for _, r := range rl {
			if r == '\r' {
				continue
			}
			if r == '\t' {
				b.WriteString("  ")
			} else if unicode.IsControl(r) {
				continue
			} else {
				b.WriteRune(r)
			}
		}
		lineStr := b.String()
		if lipgloss.Width(lineStr) > maxWidth {
			lineStr = truncateString(lineStr, maxWidth)
		}
		cleanLines = append(cleanLines, lineStr)
	}

	return cleanLines
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
