package filepicker

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/styles"
)

func truncateStr(s string, maxLen int) string {
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

type FileItem struct {
	Name    string
	Path    string
	IsDir   bool
	Size    int64
	ModTime string
}

type YaziPicker struct {
	CurrentDir      string
	ParentDir       string
	ParentEntries   []FileItem
	CurrentEntries  []FileItem
	SelectedIdx     int
	ScrollOffset    int
	ParentScroll    int
	SelectedPaths   map[string]bool
	PreviewContent  string
	PreviewLines    []string
	PreviewIsDir    bool
	PreviewEntries  []FileItem
	Active          bool
	TargetBucket    string
	TargetPrefix    string
	SelectDirOnly   bool
	styles          styles.Styles
}

func NewYaziPicker(startDir string, bucket string, prefix string, s styles.Styles) YaziPicker {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			startDir = "."
		}
	}
	abs, err := filepath.Abs(startDir)
	if err == nil {
		startDir = abs
	}

	yp := YaziPicker{
		CurrentDir:    startDir,
		SelectedPaths: make(map[string]bool),
		TargetBucket:  bucket,
		TargetPrefix:  prefix,
		styles:        s,
	}
	yp.Refresh()
	return yp
}

func (yp *YaziPicker) Refresh() {
	clean := filepath.Clean(yp.CurrentDir)
	yp.CurrentDir = clean
	parent := filepath.Dir(clean)
	if parent != clean {
		yp.ParentDir = parent
		yp.ParentEntries = readDirEntries(parent)
	} else {
		yp.ParentDir = ""
		yp.ParentEntries = nil
	}

	yp.CurrentEntries = readDirEntries(clean)
	if yp.SelectedIdx >= len(yp.CurrentEntries) {
		yp.SelectedIdx = max(0, len(yp.CurrentEntries)-1)
	}
	if yp.SelectedIdx < 0 {
		yp.SelectedIdx = 0
	}
	yp.updatePreview()
}

func readDirEntries(dirPath string) []FileItem {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil
	}

	var dirs []FileItem
	var files []FileItem
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") && name != ".env" && name != ".gitignore" {
			continue
		}
		fullPath := filepath.Join(dirPath, name)
		info, err := entry.Info()
		var size int64
		var modTime string
		if err == nil {
			size = info.Size()
			modTime = info.ModTime().Format("2006-01-02 15:04")
		}

		item := FileItem{
			Name:    name,
			Path:    fullPath,
			IsDir:   entry.IsDir(),
			Size:    size,
			ModTime: modTime,
		}
		if entry.IsDir() {
			dirs = append(dirs, item)
		} else {
			files = append(files, item)
		}
	}

	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	return append(dirs, files...)
}

func (yp *YaziPicker) updatePreview() {
	if len(yp.CurrentEntries) == 0 || yp.SelectedIdx >= len(yp.CurrentEntries) {
		yp.PreviewContent = "Empty directory"
		yp.PreviewLines = []string{"[Empty]"}
		yp.PreviewIsDir = false
		yp.PreviewEntries = nil
		return
	}

	item := yp.CurrentEntries[yp.SelectedIdx]
	if item.IsDir {
		yp.PreviewIsDir = true
		yp.PreviewEntries = readDirEntries(item.Path)
		yp.PreviewContent = fmt.Sprintf("Directory: %s (%d items)", item.Name, len(yp.PreviewEntries))
		yp.PreviewLines = nil
		return
	}

	yp.PreviewIsDir = false
	yp.PreviewEntries = nil
	yp.PreviewContent = ""

	f, err := os.Open(item.Path)
	if err != nil {
		yp.PreviewLines = []string{fmt.Sprintf("Error opening: %v", err)}
		return
	}
	defer f.Close()

	buf := make([]byte, 4096)
	n, _ := f.Read(buf)
	if n == 0 {
		yp.PreviewLines = []string{"[Empty file]"}
		return
	}

	var lines []string
	rawStr := string(buf[:n])
	for _, l := range strings.Split(rawStr, "\n") {
		clean := strings.Map(func(r rune) rune {
			if r < 32 && r != '\t' {
				return ' '
			}
			return r
		}, l)
		lines = append(lines, clean)
		if len(lines) >= 30 {
			lines = append(lines, "…")
			break
		}
	}
	yp.PreviewLines = lines
}

func (yp *YaziPicker) MoveUp() {
	if yp.SelectedIdx > 0 {
		yp.SelectedIdx--
		yp.updatePreview()
	}
}

func (yp *YaziPicker) MoveDown() {
	if yp.SelectedIdx < len(yp.CurrentEntries)-1 {
		yp.SelectedIdx++
		yp.updatePreview()
	}
}

func (yp *YaziPicker) EnterDir() {
	if len(yp.CurrentEntries) == 0 {
		return
	}
	item := yp.CurrentEntries[yp.SelectedIdx]
	if item.IsDir {
		yp.CurrentDir = item.Path
		yp.SelectedIdx = 0
		yp.ScrollOffset = 0
		yp.Refresh()
	}
}

func (yp *YaziPicker) ParentDirectory() {
	if yp.ParentDir != "" && yp.ParentDir != yp.CurrentDir {
		oldDir := yp.CurrentDir
		yp.CurrentDir = yp.ParentDir
		yp.SelectedIdx = 0
		yp.ScrollOffset = 0
		yp.Refresh()
		for i, entry := range yp.CurrentEntries {
			if entry.Path == oldDir {
				yp.SelectedIdx = i
				break
			}
		}
		yp.updatePreview()
	}
}

func (yp *YaziPicker) ToggleSelect() {
	if len(yp.CurrentEntries) == 0 {
		return
	}
	item := yp.CurrentEntries[yp.SelectedIdx]
	if yp.SelectedPaths[item.Path] {
		delete(yp.SelectedPaths, item.Path)
	} else {
		yp.SelectedPaths[item.Path] = true
	}
}

func (yp *YaziPicker) SelectAll() {
	allSelected := true
	for _, item := range yp.CurrentEntries {
		if !yp.SelectedPaths[item.Path] {
			allSelected = false
			break
		}
	}
	if allSelected {
		for _, item := range yp.CurrentEntries {
			delete(yp.SelectedPaths, item.Path)
		}
	} else {
		for _, item := range yp.CurrentEntries {
			yp.SelectedPaths[item.Path] = true
		}
	}
}

func (yp *YaziPicker) ToggleDirMode() {
	yp.SelectDirOnly = !yp.SelectDirOnly
	if yp.SelectDirOnly {
		yp.SelectedPaths = make(map[string]bool)
		yp.SelectedPaths[yp.CurrentDir] = true
	} else {
		delete(yp.SelectedPaths, yp.CurrentDir)
	}
}

func (yp *YaziPicker) GetSelectedPaths() []string {
	if len(yp.SelectedPaths) == 0 {
		if len(yp.CurrentEntries) > 0 && yp.SelectedIdx < len(yp.CurrentEntries) {
			return []string{yp.CurrentEntries[yp.SelectedIdx].Path}
		}
		return []string{yp.CurrentDir}
	}

	var paths []string
	for p := range yp.SelectedPaths {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
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
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (yp *YaziPicker) Render(totalWidth, totalHeight int) string {
	dialogW := totalWidth - 6
	if dialogW > 130 {
		dialogW = 130
	}
	if dialogW < 76 {
		dialogW = totalWidth - 2
	}
	if dialogW < 40 {
		dialogW = 40
	}

	dialogH := totalHeight - 4
	if dialogH > 36 {
		dialogH = 36
	}
	if dialogH < 18 {
		dialogH = totalHeight - 2
	}
	if dialogH < 12 {
		dialogH = 12
	}

	innerW := dialogW - 4
	innerH := dialogH - 6

	headerTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(yp.styles.Theme.BorderActive).
		Render("📁 YAZI FILE/FOLDER SELECTOR")

	destS3 := fmt.Sprintf("s3://%s/%s", yp.TargetBucket, yp.TargetPrefix)
	destText := lipgloss.NewStyle().
		Foreground(yp.styles.Theme.AccentSky).
		Bold(true).
		Render(destS3)
	destLabel := lipgloss.NewStyle().
		Foreground(yp.styles.Theme.TextMuted).
		Render("Upload Target: ")
	headerLine := lipgloss.JoinHorizontal(lipgloss.Left, headerTitle, "  │  ", destLabel, destText)

	leftW := max(16, innerW*22/100)
	rightW := max(22, innerW*38/100)
	midW := innerW - leftW - rightW - 4
	if midW < 24 {
		midW = 24
	}

	colH := max(6, innerH)

	var leftRows []string
	leftTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(yp.styles.Theme.TextMuted).
		Render(truncateStr(".. "+filepath.Base(yp.ParentDir), leftW))
	leftRows = append(leftRows, leftTitle)

	parentMax := colH - 2
	for _, entry := range yp.ParentEntries {
		if len(leftRows)-1 >= parentMax {
			break
		}
		icon := "📄"
		if entry.IsDir {
			icon = "📁"
		}
		name := truncateStr(entry.Name, leftW-5)
		line := fmt.Sprintf("%s %s", icon, name)
		if entry.Path == yp.CurrentDir {
			line = lipgloss.NewStyle().Bold(true).Foreground(yp.styles.Theme.AccentSky).Render("▶ " + line)
		} else {
			line = "  " + line
		}
		leftRows = append(leftRows, line)
	}
	leftContent := strings.Join(leftRows, "\n")
	leftCol := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(yp.styles.Theme.BorderInactive).
		Width(leftW).
		Height(colH).
		Render(leftContent)

	var midRows []string
	curBase := filepath.Base(yp.CurrentDir)
	if curBase == "/" || curBase == "." {
		curBase = yp.CurrentDir
	}
	midTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(yp.styles.Theme.BorderActive).
		Render(truncateStr(fmt.Sprintf("● %s (%d items)", curBase, len(yp.CurrentEntries)), midW))
	midRows = append(midRows, midTitle)

	midMax := colH - 2
	if yp.SelectedIdx < yp.ScrollOffset {
		yp.ScrollOffset = yp.SelectedIdx
	} else if yp.SelectedIdx >= yp.ScrollOffset+midMax {
		yp.ScrollOffset = yp.SelectedIdx - midMax + 1
	}
	if yp.ScrollOffset < 0 {
		yp.ScrollOffset = 0
	}

	if len(yp.CurrentEntries) == 0 {
		midRows = append(midRows, lipgloss.NewStyle().Foreground(yp.styles.Theme.TextMuted).Render("  (empty folder)"))
	} else {
		endIdx := min(len(yp.CurrentEntries), yp.ScrollOffset+midMax)
		for i := yp.ScrollOffset; i < endIdx; i++ {
			entry := yp.CurrentEntries[i]
			icon := "📄"
			if entry.IsDir {
				icon = "📁"
			}
			mark := "[ ]"
			if yp.SelectedPaths[entry.Path] {
				mark = "[x]"
			}

			sizeOrDir := formatBytes(entry.Size)
			if entry.IsDir {
				sizeOrDir = "[DIR]"
			}

			nameW := midW - 18
			if nameW < 8 {
				nameW = 8
			}
			name := truncateStr(entry.Name, nameW)
			display := fmt.Sprintf("%s %s %-*s %8s", mark, icon, nameW, name, sizeOrDir)

			var styledRow string
			if i == yp.SelectedIdx {
				styledRow = yp.styles.SelectedRow.Width(midW).MaxHeight(1).Render(" ▶ " + display)
			} else {
				styledRow = yp.styles.NormalRow.Width(midW).MaxHeight(1).Render("   " + display)
			}
			midRows = append(midRows, styledRow)
		}
	}
	midContent := strings.Join(midRows, "\n")
	midCol := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(yp.styles.Theme.BorderInactive).
		Width(midW).
		Height(colH).
		Render(midContent)

	var rightRows []string
	rightTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(yp.styles.Theme.TextMuted).
		Render("PREVIEW")
	rightRows = append(rightRows, rightTitle)

	rightMax := colH - 2
	if yp.PreviewIsDir {
		rightRows = append(rightRows, lipgloss.NewStyle().Foreground(yp.styles.Theme.AccentSky).Render("📁 Folder Contents:"))
		for _, e := range yp.PreviewEntries {
			if len(rightRows)-1 >= rightMax {
				break
			}
			icon := "📄"
			if e.IsDir {
				icon = "📁"
			}
			name := truncateStr(e.Name, rightW-6)
			rightRows = append(rightRows, fmt.Sprintf("  %s %s", icon, name))
		}
	} else if len(yp.PreviewLines) > 0 {
		for _, line := range yp.PreviewLines {
			if len(rightRows)-1 >= rightMax {
				break
			}
			rightRows = append(rightRows, truncateStr(line, rightW-2))
		}
	} else {
		rightRows = append(rightRows, lipgloss.NewStyle().Foreground(yp.styles.Theme.TextMuted).Render(yp.PreviewContent))
	}

	rightContent := strings.Join(rightRows, "\n")
	rightCol := lipgloss.NewStyle().
		Width(rightW).
		Height(colH).
		Render(rightContent)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, " ", midCol, " ", rightCol)

	selectedCount := len(yp.SelectedPaths)
	if selectedCount == 0 && len(yp.CurrentEntries) > 0 {
		selectedCount = 1
	}

	summary := lipgloss.NewStyle().
		Bold(true).
		Foreground(yp.styles.Theme.BadgeSuccess).
		Render(fmt.Sprintf("Selected: %d item(s)", selectedCount))

	pathBreadcrumb := lipgloss.NewStyle().
		Foreground(yp.styles.Theme.TextMuted).
		Render(truncateStr(yp.CurrentDir, innerW-35))

	statusLine := lipgloss.JoinHorizontal(lipgloss.Left, summary, "  │  ", pathBreadcrumb)

	legendKey := func(k, desc string) string {
		return yp.styles.StatusKey.Render(k) + yp.styles.StatusDesc.Render(" "+desc+"  ")
	}

	legend := lipgloss.JoinHorizontal(lipgloss.Left,
		legendKey("[Enter/u/y]", "Upload"),
		legendKey("[Space]", "Toggle"),
		legendKey("[a]", "All"),
		legendKey("[h/l]", "Parent/Enter"),
		legendKey("[j/k]", "Up/Down"),
		legendKey("[Tab]", "Folder Mode"),
		legendKey("[Esc/q]", "Cancel"),
	)

	body := lipgloss.JoinVertical(lipgloss.Left,
		headerLine,
		"",
		columns,
		"",
		statusLine,
		legend,
	)

	modalBox := yp.styles.ModalBox.Width(dialogW).Height(dialogH).Render(body)
	return lipgloss.Place(totalWidth, totalHeight, lipgloss.Center, lipgloss.Center, modalBox)
}
