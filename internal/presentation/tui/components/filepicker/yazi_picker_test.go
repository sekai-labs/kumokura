package filepicker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sekai-labs/kumokura/internal/presentation/tui/styles"
	"github.com/stretchr/testify/assert"
)

func TestYaziPicker_NavigationAndSelection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "yazi-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	subDir := filepath.Join(tempDir, "subfolder")
	assert.NoError(t, os.Mkdir(subDir, 0755))

	file1 := filepath.Join(tempDir, "file1.txt")
	assert.NoError(t, os.WriteFile(file1, []byte("hello world"), 0644))

	file2 := filepath.Join(tempDir, "file2.log")
	assert.NoError(t, os.WriteFile(file2, []byte("log data"), 0644))

	st := styles.NewStyles(styles.DarkTheme)
	picker := NewYaziPicker(tempDir, "test-bucket", "prefix/", st)
	assert.Equal(t, tempDir, picker.CurrentDir)
	assert.Len(t, picker.CurrentEntries, 3)

	assert.Equal(t, 0, picker.SelectedIdx)

	picker.MoveDown()
	assert.Equal(t, 1, picker.SelectedIdx)

	picker.MoveUp()
	assert.Equal(t, 0, picker.SelectedIdx)

	picker.ToggleSelect()
	selected := picker.GetSelectedPaths()
	assert.Len(t, selected, 1)

	picker.SelectAll()
	assert.Len(t, picker.SelectedPaths, 3)

	rendered := picker.Render(120, 35)
	assert.NotEmpty(t, rendered)
	assert.Contains(t, rendered, "YAZI FILE/FOLDER SELECTOR")
	assert.Contains(t, rendered, "s3://test-bucket/prefix/")
}
