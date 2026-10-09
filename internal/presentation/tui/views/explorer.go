package views

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

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
	PreviewKey      string
	PreviewMetadata *objDomain.ObjectMetadata
	PreviewTags     []objDomain.ObjectTag
	PreviewContent  []byte
	PreviewScroll   int
	ShowPreview     bool
	ActivePaneIndex int
	SelectedKeys    map[string]bool
	styles          styles.Styles
}

func NewExplorerView(s styles.Styles) ExplorerView {
	return ExplorerView{
		ShowPreview:     true,
		ActivePaneIndex: 1,
		SelectedKeys:    make(map[string]bool),
		styles:          s,
	}
}
func (v ExplorerView) GetCurrentTargetPrefix() string {
	if v.ActivePaneIndex == 1 {
		prefixLen := len(v.Prefixes)
		if v.SelectedObject >= 0 && v.SelectedObject < prefixLen {
			return v.Prefixes[v.SelectedObject].Prefix
		}
		objIdx := v.SelectedObject - prefixLen
		var visibleObjects []objDomain.Object
		for _, obj := range v.Objects {
			if v.CurrentPrefix != "" && (obj.Key == v.CurrentPrefix || obj.Key == strings.TrimSuffix(v.CurrentPrefix, "/")) {
				continue
			}
			visibleObjects = append(visibleObjects, obj)
		}
		if objIdx >= 0 && objIdx < len(visibleObjects) {
			k := visibleObjects[objIdx].Key
			if strings.HasSuffix(k, "/") {
				return k
			}
		}
	}
	return v.CurrentPrefix
}

func (v ExplorerView) SelectedBucketItem() *bucketDomain.Bucket {
	if len(v.Buckets) == 0 {
		return nil
	}
	idx := v.SelectedBucket
	if idx < 0 {
		idx = 0
	}
	if idx >= len(v.Buckets) {
		idx = len(v.Buckets) - 1
	}
	return &v.Buckets[idx]
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
	headerBadge := "BUCKETS"
	if isActive {
		headerBadge = "● BUCKETS"
	}
	headerTitle := fmt.Sprintf("%s (%d)", headerBadge, len(v.Buckets))
	title := v.styles.PanelTitle.Render(headerTitle)
	maxItems := height - 3
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
		emptyCard := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(v.styles.Theme.BorderInactive).
			Padding(1, 2).
			Foreground(v.styles.Theme.TextMuted).
			Render("No buckets found.\nCheck cloud credentials.")
		rows = append(rows, emptyCard)
	} else {
		endIdx := min(len(v.Buckets), bucketOffset+maxItems)
		for i := bucketOffset; i < endIdx; i++ {
			b := v.Buckets[i]
			name := b.Name
			availTextW := width - 7
			if availTextW < 4 {
				availTextW = 4
			}
			name = truncateString(name, availTextW)

			var rowStr string
			if i == selectedBucket && isActive {
				rowStr = v.styles.SelectedRow.Width(width - 2).Render(fmt.Sprintf(" ▶ 🪣 %s", name))
			} else if i == selectedBucket {
				rowStr = v.styles.SelectedRow.Width(width - 2).Render(fmt.Sprintf(" ▷ 🪣 %s", name))
			} else {
				rowStr = v.styles.NormalRow.Width(width - 2).Render(fmt.Sprintf("   🪣 %s", name))
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

	var breadcrumbParts []string
	breadcrumbParts = append(breadcrumbParts, "s3://"+bucketName)
	if v.CurrentPrefix != "" {
		segments := strings.Split(strings.Trim(v.CurrentPrefix, "/"), "/")
		for _, seg := range segments {
			if seg != "" {
				breadcrumbParts = append(breadcrumbParts, seg)
			}
		}
	}
	formattedBreadcrumbs := " " + strings.Join(breadcrumbParts, " ❯ ") + " "
	availBcWidth := width - 14
	if availBcWidth > 10 && lipgloss.Width(formattedBreadcrumbs) > availBcWidth {
		formattedBreadcrumbs = truncateLeadingString(formattedBreadcrumbs, availBcWidth)
	}

	bcHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(v.styles.Theme.AccentSky).
		Render(formattedBreadcrumbs)

	headerTitle := "OBJECT EXPLORER"
	if isActive {
		headerTitle = "● OBJECT EXPLORER"
	}
	panelHeader := lipgloss.JoinHorizontal(lipgloss.Left, v.styles.PanelTitle.Render(headerTitle), " ", bcHeader)

	var visibleObjects []objDomain.Object
	for _, obj := range v.Objects {
		if v.CurrentPrefix != "" && (obj.Key == v.CurrentPrefix || obj.Key == strings.TrimSuffix(v.CurrentPrefix, "/")) {
			continue
		}
		visibleObjects = append(visibleObjects, obj)
	}

	totalItems := len(v.Prefixes) + len(visibleObjects)
	maxItems := height - 4
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

	availColW := width - 4
	if availColW < 20 {
		availColW = 20
	}

	showDetails := availColW >= 65
	showStorageClass := availColW >= 80

	sizeColW := 10
	scColW := 15
	dateColW := 12
	fixedW := sizeColW + 2
	if showStorageClass {
		fixedW += scColW + 1
	}
	if showDetails {
		fixedW += dateColW + 1
	}

	keyColW := availColW - fixedW - 4
	if keyColW < 12 {
		keyColW = 12
	}

	var tblHeader string
	if showStorageClass {
		tblHeader = fmt.Sprintf("   %s %s %s %s",
			padVisual("Name", keyColW, false),
			padVisual("Size", sizeColW, true),
			padVisual("Storage Class", scColW, true),
			padVisual("Last Modified", dateColW, true),
		)
	} else if showDetails {
		tblHeader = fmt.Sprintf("   %s %s %s",
			padVisual("Name", keyColW, false),
			padVisual("Size", sizeColW, true),
			padVisual("Last Modified", dateColW, true),
		)
	} else {
		tblHeader = fmt.Sprintf("   %s %s",
			padVisual("Name", keyColW, false),
			padVisual("Size", sizeColW, true),
		)
	}

	hdrStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(v.styles.Theme.TextSubtle).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(v.styles.Theme.BorderInactive).
		MaxWidth(width - 2).
		MaxHeight(1)
	var rows []string
	rows = append(rows, hdrStyle.Width(width-2).Render(tblHeader))

	if totalItems == 0 {
		emptyCard := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(v.styles.Theme.BorderInactive).
			Padding(1, 2).
			Foreground(v.styles.Theme.TextMuted).
			Render("Empty directory.\nPress 'u' to upload a file/folder or 'Backspace'/'h' to go up.")
		rows = append(rows, "", emptyCard)
	} else {
		prefixLen := len(v.Prefixes)
		endIdx := min(totalItems, objectOffset+maxItems)
		hasSelected := len(v.SelectedKeys) > 0
		showMultiMarker := v.ShowPreview || hasSelected

		for i := objectOffset; i < endIdx; i++ {
			var display string
			var itemKey string
			var isDir bool

			if i < prefixLen {
				p := v.Prefixes[i]
				itemKey = p.Prefix
				isDir = true
				name := p.Prefix
				if v.CurrentPrefix != "" {
					name = strings.TrimPrefix(name, v.CurrentPrefix)
				}
				maxNameW := keyColW - 3
				if maxNameW < 8 {
					maxNameW = 8
				}
				name = truncateString(name, maxNameW)

				var marker string
				if showMultiMarker {
					if v.SelectedKeys[itemKey] {
						marker = "[x] "
					} else {
						marker = "[ ] "
					}
				}
				iconAndName := fmt.Sprintf("%s📁 %s", marker, padVisual(name, maxNameW, false))

				if showStorageClass {
					display = fmt.Sprintf("%s %s %s %s",
						iconAndName,
						padVisual("[DIR]", sizeColW, true),
						padVisual("-", scColW, true),
						padVisual("-", dateColW, true),
					)
				} else if showDetails {
					display = fmt.Sprintf("%s %s %s",
						iconAndName,
						padVisual("[DIR]", sizeColW, true),
						padVisual("-", dateColW, true),
					)
				} else {
					display = fmt.Sprintf("%s %s",
						iconAndName,
						padVisual("[DIR]", sizeColW, true),
					)
				}
			} else {
				objIdx := i - prefixLen
				obj := visibleObjects[objIdx]
				itemKey = obj.Key
				isDir = false
				name := obj.Key
				if v.CurrentPrefix != "" {
					name = strings.TrimPrefix(name, v.CurrentPrefix)
				}
				sizeStr := formatBytes(obj.Size)
				storageClass := string(obj.StorageClass)
				if storageClass == "" {
					storageClass = "STANDARD"
				}
				lastModStr := "-"
				if !obj.LastModified.IsZero() {
					lastModStr = obj.LastModified.Format("2006-01-02")
				}

				nameWidth := keyColW - 3
				if nameWidth < 8 {
					nameWidth = 8
				}
				name = truncateString(name, nameWidth)

				var marker string
				if showMultiMarker {
					if v.SelectedKeys[itemKey] {
						marker = "[x] "
					} else {
						marker = "[ ] "
					}
				}
				iconAndName := fmt.Sprintf("%s📄 %s", marker, padVisual(name, nameWidth, false))

				if showStorageClass {
					display = fmt.Sprintf("%s %s %s %s",
						iconAndName,
						padVisual(sizeStr, sizeColW, true),
						padVisual(storageClass, scColW, true),
						padVisual(lastModStr, dateColW, true),
					)
				} else if showDetails {
					display = fmt.Sprintf("%s %s %s",
						iconAndName,
						padVisual(sizeStr, sizeColW, true),
						padVisual(lastModStr, dateColW, true),
					)
				} else {
					display = fmt.Sprintf("%s %s",
						iconAndName,
						padVisual(sizeStr, sizeColW, true),
					)
				}
			}

			_ = isDir
			var rowStr string
			rowStyle := v.styles.NormalRow
			prefixBadge := "   "
			if i == selectedObject && isActive {
				rowStyle = v.styles.SelectedRow
				prefixBadge = " ▶ "
			} else if i == selectedObject {
				rowStyle = v.styles.SelectedRow
				prefixBadge = " ▷ "
			}
			rowStr = rowStyle.Width(width - 2).MaxHeight(1).Render(prefixBadge + display)
			rows = append(rows, rowStr)
		}

		if totalItems > maxItems {
			scrollIndicator := fmt.Sprintf("[%d-%d/%d]", objectOffset+1, endIdx, totalItems)
			indStyle := lipgloss.NewStyle().
				Foreground(v.styles.Theme.TextMuted).
				Align(lipgloss.Right).
				Width(width - 4)
			rows = append(rows, indStyle.Render(scrollIndicator))
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, panelHeader, strings.Join(rows, "\n"))
	style := v.styles.InactivePanel
	if isActive {
		style = v.styles.ActivePanel
	}
	return style.Width(width).Height(height).Render(content)
}

func padVisual(s string, targetWidth int, alignRight bool) string {
	w := lipgloss.Width(s)
	if w >= targetWidth {
		return truncateString(s, targetWidth)
	}
	padding := strings.Repeat(" ", targetWidth-w)
	if alignRight {
		return padding + s
	}
	return s + padding
}

func (v ExplorerView) renderRightPanel(width, height int) string {
	isActive := v.ActivePaneIndex == 2
	headerBadge := "INSPECTOR"
	if isActive {
		headerBadge = "● INSPECTOR"
	}
	title := v.styles.PanelTitle.Render(headerBadge)
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
				tagLine := fmt.Sprintf("  🏷 %s = %s", tag.Key, tag.Value)
				lines = append(lines, truncateString(tagLine, width-4))
			}
		}

		if len(v.PreviewContent) > 0 {
			lines = append(lines, "")
			targetName := v.PreviewKey
			if IsVideoFile(targetName) {
				lines = append(lines, v.styles.PanelTitle.Render("MEDIA PREVIEW:"))
				lines = append(lines, renderVideoPreview(targetName, v.PreviewMetadata, width-4)...)
			} else if IsImageFile(targetName) {
				lines = append(lines, v.styles.PanelTitle.Render("IMAGE PREVIEW:"))
				lines = append(lines, renderImagePreview(v.PreviewContent, targetName, v.PreviewMetadata, width-4, 12)...)
			} else {
				lines = append(lines, v.styles.PanelTitle.Render("PREVIEW:"))
				sanitizedLines := SanitizePreviewContent(v.PreviewContent, width-4, 15)
				for _, sl := range sanitizedLines {
					lines = append(lines, v.styles.StatusDesc.Render(sl))
				}
			}
		} else if v.PreviewMetadata != nil && IsVideoFile(v.PreviewKey) {
			lines = append(lines, "")
			lines = append(lines, v.styles.PanelTitle.Render("MEDIA PREVIEW:"))
			lines = append(lines, renderVideoPreview(v.PreviewKey, v.PreviewMetadata, width-4)...)
		}

	} else {
		emptyCard := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(v.styles.Theme.BorderInactive).
			Padding(1, 1).
			Foreground(v.styles.Theme.TextMuted).
			Render("No object selected.\nNavigate to an object to inspect.")
		lines = append(lines, emptyCard, "")
		lines = append(lines, v.styles.StatusDesc.Render("[i] Toggle Inspector"))
		lines = append(lines, v.styles.StatusDesc.Render("[p] Generate presigned URL"))
		lines = append(lines, v.styles.StatusDesc.Render("[u] Upload file or folder"))
		lines = append(lines, v.styles.StatusDesc.Render("[d] Download object or folder"))
		lines = append(lines, v.styles.StatusDesc.Render("[x] Delete selected object"))
		lines = append(lines, v.styles.StatusDesc.Render("[Space] Toggle selection"))
		lines = append(lines, v.styles.StatusDesc.Render("[o] Open in external viewer/player"))
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

func IsImageFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp", ".svg":
		return true
	default:
		return false
	}
}

func IsVideoFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".mp4", ".mkv", ".webm", ".avi", ".mov":
		return true
	default:
		return false
	}
}

func renderVideoPreview(name string, meta *objDomain.ObjectMetadata, maxWidth int) []string {
	var lines []string
	ext := strings.ToUpper(strings.TrimPrefix(filepath.Ext(name), "."))
	if ext == "" {
		ext = "VIDEO"
	}
	badge := fmt.Sprintf("🎬 VIDEO [%s]", ext)
	badgeRendered := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#06B6D4")).
		Background(lipgloss.Color("#1E293B")).
		Padding(0, 1).
		Render(badge)
	lines = append(lines, badgeRendered)

	if meta != nil {
		if meta.ContentType != "" {
			lines = append(lines, fmt.Sprintf("Format: %s", truncateString(meta.ContentType, maxWidth-8)))
		}
		if meta.ContentLength > 0 {
			lines = append(lines, fmt.Sprintf("Size:   %s", formatBytes(meta.ContentLength)))
		}
	}

	prompt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#38BDF8")).
		Bold(true).
		Render("Press 'o' to open video in default player")
	lines = append(lines, "", prompt)
	return lines
}

func renderImagePreview(data []byte, name string, meta *objDomain.ObjectMetadata, maxWidth, maxHeight int) []string {
	var lines []string
	ext := strings.ToLower(filepath.Ext(name))

	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err == nil && cfg.Width > 0 && cfg.Height > 0 {
		dimStr := fmt.Sprintf("Dimensions: %d × %d px", cfg.Width, cfg.Height)
		fmtStr := fmt.Sprintf("Format:     %s", strings.ToUpper(format))
		lines = append(lines, dimStr, fmtStr)
	} else if ext == ".svg" {
		lines = append(lines, "Format:     SVG (Vector graphics)")
	} else if meta != nil && meta.ContentType != "" {
		lines = append(lines, fmt.Sprintf("Format:     %s", meta.ContentType))
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err == nil {
		asciiArt := renderASCIIImage(img, maxWidth, maxHeight)
		if len(asciiArt) > 0 {
			lines = append(lines, "")
			lines = append(lines, asciiArt...)
		}
	} else if ext == ".svg" {
		lines = append(lines, "")
		svgPreview := SanitizePreviewContent(data, maxWidth, maxHeight)
		lines = append(lines, svgPreview...)
	}

	openPrompt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#38BDF8")).
		Render("Press 'o' to open image in system viewer")
	lines = append(lines, "", openPrompt)
	return lines
}

func renderASCIIImage(img image.Image, maxWidth, maxHeight int) []string {
	bounds := img.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()
	if origW == 0 || origH == 0 {
		return nil
	}

	targetW := maxWidth
	if targetW > 36 {
		targetW = 36
	}
	if targetW < 10 {
		targetW = 10
	}

	// Terminal characters are roughly twice as tall as they are wide.
	targetH := (origH * targetW) / (origW * 2)
	if targetH > maxHeight {
		targetH = maxHeight
	}
	if targetH < 3 {
		targetH = 3
	}

	// Ramp from dark to bright:
	asciiChars := []rune(" .:-=+*#%@")
	numChars := len(asciiChars)

	var lines []string
	for y := range targetH {
		var row strings.Builder
		for x := range targetW {
			srcX := bounds.Min.X + (x * origW / targetW)
			srcY := bounds.Min.Y + (y * origH / targetH)
			r, g, b, _ := img.At(srcX, srcY).RGBA()
			// Luminance formula (ITU-R BT.601) scaled to 0..255
			gray := (299*(r>>8) + 587*(g>>8) + 114*(b>>8)) / 1000
			charIdx := int(gray) * (numChars - 1) / 255
			if charIdx >= numChars {
				charIdx = numChars - 1
			}
			row.WriteRune(asciiChars[charIdx])
		}
		lines = append(lines, row.String())
	}
	return lines
}
