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
	ActivePaneIndex int // 0: Buckets, 1: Objects, 2: Preview
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
	if width < 80 || height < 12 {
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Terminal window too small. Minimum 80x24 recommended.")
	}

	// Only show preview dock side-by-side if terminal is wide enough and user hasn't toggled it off
	showPreview := v.ShowPreview && width >= 110

	var leftInnerW, centerInnerW, rightInnerW int
	if showPreview {
		avail := width - 6 // 3 panels * 2 border cols
		leftInnerW = max(20, width*20/100)
		rightInnerW = max(28, width*32/100)
		centerInnerW = avail - leftInnerW - rightInnerW
		if centerInnerW < 24 {
			// Not enough room for 3 panels comfortably, fall back to 2 panels
			showPreview = false
		}
	}

	if !showPreview {
		avail := width - 4 // 2 panels * 2 border cols
		leftInnerW = max(22, min(35, width*26/100))
		centerInnerW = avail - leftInnerW
	}

	panelInnerH := height - 2 // account for borders
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

func (v *ExplorerView) renderLeftPanel(width, height int) string {
	isActive := v.ActivePaneIndex == 0
	headerTitle := fmt.Sprintf("BUCKETS (%d)", len(v.Buckets))
	title := v.styles.PanelTitle.Render(headerTitle)

	maxItems := height - 2 // 1 header line, 1 bottom pad
	if maxItems < 1 {
		maxItems = 1
	}

	// Ensure selection is within bounds
	if v.SelectedBucket >= len(v.Buckets) {
		v.SelectedBucket = max(0, len(v.Buckets)-1)
	}

	// Adjust scroll window
	if v.SelectedBucket < v.BucketOffset {
		v.BucketOffset = v.SelectedBucket
	} else if v.SelectedBucket >= v.BucketOffset+maxItems {
		v.BucketOffset = v.SelectedBucket - maxItems + 1
	}
	if v.BucketOffset < 0 {
		v.BucketOffset = 0
	}

	var rows []string
	if len(v.Buckets) == 0 {
		emptyMsg := v.styles.StatusDesc.Render("No buckets found")
		rows = append(rows, emptyMsg)
	} else {
		endIdx := min(len(v.Buckets), v.BucketOffset+maxItems)
		for i := v.BucketOffset; i < endIdx; i++ {
			b := v.Buckets[i]
			name := b.Name
			availTextW := width - 4
			if availTextW < 4 {
				availTextW = 4
			}
			if lipgloss.Width(name) > availTextW {
				name = name[:availTextW-1] + "…"
			}

			var rowStr string
			if i == v.SelectedBucket && isActive {
				rowStr = v.styles.SelectedRow.Width(width).Render(fmt.Sprintf(" ▶ %s", name))
			} else if i == v.SelectedBucket {
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

func (v *ExplorerView) renderCenterPanel(width, height int) string {
	isActive := v.ActivePaneIndex == 1
	totalItems := len(v.Prefixes) + len(v.Objects)
	prefixDisplay := "/" + v.CurrentPrefix
	availTitle := width - 18
	if availTitle > 6 && lipgloss.Width(prefixDisplay) > availTitle {
		prefixDisplay = "…" + prefixDisplay[len(prefixDisplay)-availTitle+1:]
	}
	titleText := fmt.Sprintf("OBJECTS: %s (%d)", prefixDisplay, totalItems)
	title := v.styles.PanelTitle.Render(titleText)

	maxItems := height - 2
	if maxItems < 1 {
		maxItems = 1
	}

	if v.SelectedObject >= totalItems {
		v.SelectedObject = max(0, totalItems-1)
	}

	if v.SelectedObject < v.ObjectOffset {
		v.ObjectOffset = v.SelectedObject
	} else if v.SelectedObject >= v.ObjectOffset+maxItems {
		v.ObjectOffset = v.SelectedObject - maxItems + 1
	}
	if v.ObjectOffset < 0 {
		v.ObjectOffset = 0
	}

	var rows []string
	if totalItems == 0 {
		emptyMsg := v.styles.StatusDesc.Render("Prefix is empty (or no objects match filter)")
		rows = append(rows, emptyMsg)
	} else {
		prefixLen := len(v.Prefixes)
		endIdx := min(totalItems, v.ObjectOffset+maxItems)

		for i := v.ObjectOffset; i < endIdx; i++ {
			var display string
			if i < prefixLen {
				// Directory
				p := v.Prefixes[i]
				name := p.Prefix
				if v.CurrentPrefix != "" {
					name = strings.TrimPrefix(name, v.CurrentPrefix)
				}
				maxNameW := width - 14
				if maxNameW < 4 {
					maxNameW = 4
				}
				if lipgloss.Width(name) > maxNameW {
					name = name[:maxNameW-1] + "…"
				}
				display = fmt.Sprintf("📁 %-*s [DIR]", maxNameW, name)
			} else {
				// Object
				objIdx := i - prefixLen
				obj := v.Objects[objIdx]
				name := obj.Key
				if v.CurrentPrefix != "" {
					name = strings.TrimPrefix(name, v.CurrentPrefix)
				}
				sizeStr := formatBytes(obj.Size)
				storageClass := string(obj.StorageClass)
				if storageClass == "" {
					storageClass = "STANDARD"
				}

				// Reserve room for size and storage class
				metaWidth := 11 + len(storageClass) // e.g. " 1024.0 KiB STANDARD"
				nameWidth := width - metaWidth - 7
				if nameWidth < 8 {
					nameWidth = 8
				}
				if lipgloss.Width(name) > nameWidth {
					name = name[:nameWidth-1] + "…"
				}

				display = fmt.Sprintf("📄 %-*s %9s %s", nameWidth, name, sizeStr, storageClass)
			}

			var rowStr string
			if i == v.SelectedObject && isActive {
				rowStr = v.styles.SelectedRow.Width(width).Render(fmt.Sprintf(" ▶ %s", display))
			} else if i == v.SelectedObject {
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

func (v *ExplorerView) renderRightPanel(width, height int) string {
	isActive := v.ActivePaneIndex == 2
	title := v.styles.PanelTitle.Render("DETAILS & PREVIEW")
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
		lines = append(lines, v.styles.StatusDesc.Render("Press [p] to inspect/toggle preview"))
		lines = append(lines, v.styles.StatusDesc.Render("Press [u] to upload a file"))
		lines = append(lines, v.styles.StatusDesc.Render("Press [D] to delete selected object"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, strings.Join(lines, "\n"))
	style := v.styles.InactivePanel
	if isActive {
		style = v.styles.ActivePanel
	}
	return style.Width(width).Height(height).Render(content)
}

func truncateString(s string, maxLen int) string {
	if maxLen < 3 {
		maxLen = 3
	}
	if lipgloss.Width(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

// SanitizePreviewContent safely prepares arbitrary preview bytes for terminal display without
// emitting terminal escape sequences, control characters, or invalid UTF-8 sequences.
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

	// Binary detection: check if invalid UTF-8 or contains null bytes
	isBinary := !utf8.Valid(data) || bytes.IndexByte(data, 0) != -1

	if isBinary {
		// Format clean hex dump
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
			// Pad hexPart if chunk < 16
			for i := len(chunk); i < 16; i++ {
				hexPart.WriteString("   ")
			}

			line := fmt.Sprintf("%04x  %s |%s|", start, hexPart.String(), asciiPart.String())
			lines = append(lines, truncateString(line, maxWidth))
		}
		return lines
	}

	// Plain text: strip escape codes (\x1b) and non-printable control characters
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
				// Skip escape sequences and control characters
				continue
			} else {
				b.WriteRune(r)
			}
		}
		lineStr := b.String()
		if lipgloss.Width(lineStr) > maxWidth {
			lineStr = lineStr[:maxWidth-1] + "…"
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
