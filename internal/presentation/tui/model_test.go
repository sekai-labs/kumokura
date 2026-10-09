package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	bucketDomain "github.com/sekai-labs/kumokura/internal/buckets/domain"
	objDomain "github.com/sekai-labs/kumokura/internal/objects/domain"
	objPorts "github.com/sekai-labs/kumokura/internal/objects/ports"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/messages"
	"github.com/stretchr/testify/assert"
)

func TestTUI_ModelInitAndRender(t *testing.T) {
	model := NewModel(Services{})
	assert.Equal(t, 0, model.activeTab)

	viewStr := model.View()
	assert.NotEmpty(t, viewStr)
	assert.Contains(t, viewStr, "[1 Buckets]")
	assert.Contains(t, viewStr, "BUCKETS")
}

func TestTUI_ResponsiveResize(t *testing.T) {
	model := NewModel(Services{})

	m, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	updated := m.(Model)
	assert.Equal(t, 80, updated.width)
	assert.Equal(t, 24, updated.height)

	render80x24 := updated.View()
	assert.NotEmpty(t, render80x24)

	m160, _ := model.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	updated160 := m160.(Model)
	render160 := updated160.View()
	assert.NotEmpty(t, render160)
}

func TestTUI_TabSwitching(t *testing.T) {
	model := NewModel(Services{})

	m3, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	updated := m3.(Model)
	assert.Equal(t, 2, updated.activeTab)
	viewTransfers := updated.View()
	assert.Contains(t, viewTransfers, "TRANSFERS")

	m4, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	updated4 := m4.(Model)
	assert.Equal(t, 3, updated4.activeTab)
	viewSync := updated4.View()
	assert.Contains(t, viewSync, "SYNC ENGINE")

	m1, _ := updated4.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	updated1 := m1.(Model)
	assert.Equal(t, 0, updated1.activeTab)
	mTab1, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	uTab1 := mTab1.(Model)
	assert.Equal(t, 0, uTab1.activeTab)
	assert.Equal(t, 0, uTab1.explorerView.ActivePaneIndex)

	mTab2, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	uTab2 := mTab2.(Model)
	assert.Equal(t, 1, uTab2.activeTab)
	assert.Equal(t, 1, uTab2.explorerView.ActivePaneIndex)

	uTab2.explorerView.Buckets = []bucketDomain.Bucket{{Name: "b1"}}
	uTab2.explorerView.Objects = []objDomain.Object{
		{Key: "file1.txt", Size: 100},
		{Key: "file2.txt", Size: 200},
	}
	assert.Equal(t, 0, uTab2.explorerView.SelectedObject)

	mDown, _ := uTab2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	uDown := mDown.(Model)
	assert.Equal(t, 1, uDown.explorerView.SelectedObject)

	mUp, _ := uDown.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	uUp := mUp.(Model)
	assert.Equal(t, 0, uUp.explorerView.SelectedObject)

	mG, _ := uUp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	uG := mG.(Model)
	assert.Equal(t, 1, uG.explorerView.SelectedObject)

	mg, _ := uG.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	ug := mg.(Model)
	assert.Equal(t, 0, ug.explorerView.SelectedObject)
}

func TestTUI_DataLoadedHandling(t *testing.T) {
	model := NewModel(Services{})

	m, _ := model.Update(messages.BucketsLoadedMsg{
		Buckets: []bucketDomain.Bucket{
			{Name: "prod-storage", Region: "us-west-2"},
		},
	})
	updated := m.(Model)
	assert.Len(t, updated.explorerView.Buckets, 1)
	assert.Equal(t, "prod-storage", updated.explorerView.Buckets[0].Name)

	mObj, _ := updated.Update(messages.ObjectsLoadedMsg{
		Result: objPorts.ListObjectsResult{
			Objects: []objDomain.Object{
				{Key: "data/db.sqlite", Size: 4096},
			},
			CommonPrefixes: []objDomain.Prefix{
				{Prefix: "data/logs/"},
			},
		},
	})
	updatedObj := mObj.(Model)
	assert.Len(t, updatedObj.explorerView.Objects, 1)
	assert.Len(t, updatedObj.explorerView.Prefixes, 1)
}

func TestTUI_HelpModal(t *testing.T) {
	model := NewModel(Services{})

	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	updated := m.(Model)
	assert.True(t, updated.showHelpModal)
	view := updated.View()
	assert.Contains(t, view, "HELP OVERLAY")

	mEsc, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	closed := mEsc.(Model)
	assert.False(t, closed.showHelpModal)
}

func TestTUI_UploadModal(t *testing.T) {
	model := NewModel(Services{})
	model.activeBucket = "test-bucket"

	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	updated := m.(Model)
	assert.True(t, updated.uploadModal.Active)
	assert.Contains(t, updated.uploadModal.Destination, "test-bucket")

	view := updated.View()
	assert.Contains(t, view, "UPLOAD (FILE OR FOLDER)")
	assert.Contains(t, view, "Local path (file or directory):")
	mEsc, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	closed := mEsc.(Model)
	assert.False(t, closed.uploadModal.Active)
}

func TestTUI_PreviewSanitization(t *testing.T) {
	model := NewModel(Services{})
	model.activeBucket = "test-bucket"

	binaryData := []byte{0x00, 0x01, 0x1b, 0x5b, 0x32, 0x4a, 0x48, 0x65, 0x6c, 0x6c, 0x6f}
	m, _ := model.Update(messages.ContentPreviewLoadedMsg{
		Key:     "test.bin",
		Content: string(binaryData),
	})
	updated := m.(Model)
	assert.Equal(t, binaryData, updated.explorerView.PreviewContent)

	textData := "Line 1\nLine 2\nLine 3"
	mText, _ := model.Update(messages.ContentPreviewLoadedMsg{
		Key:     "test.txt",
		Content: textData,
	})
	updatedText := mText.(Model)
	assert.Equal(t, []byte(textData), updatedText.explorerView.PreviewContent)
}
func TestTUI_DownloadModalAndFolderDownload(t *testing.T) {
	model := NewModel(Services{})
	model.activeBucket = "test-bucket"
	model.explorerView.Objects = []objDomain.Object{
		{Key: "data/file.txt", Size: 1024},
	}
	model.explorerView.Prefixes = []objDomain.Prefix{
		{Prefix: "data/subfolder/"},
	}

	model.explorerView.SelectedObject = 0
	mFolder, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	uFolder := mFolder.(Model)
	assert.True(t, uFolder.downloadModal.Active)
	assert.True(t, uFolder.downloadModal.IsFolder)
	assert.Equal(t, "data/subfolder/", uFolder.downloadModal.TargetName)
	viewFolder := uFolder.View()
	assert.Contains(t, viewFolder, "DOWNLOAD FOLDER (RECURSIVE)")

	model.explorerView.SelectedObject = 1
	mObj, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	uObj := mObj.(Model)
	assert.True(t, uObj.downloadModal.Active)
	assert.False(t, uObj.downloadModal.IsFolder)
	assert.Equal(t, "data/file.txt", uObj.downloadModal.TargetName)
	viewObj := uObj.View()
	assert.Contains(t, viewObj, "DOWNLOAD OBJECT")
}

func TestTUI_PresignModalAndInspectorToggle(t *testing.T) {
	model := NewModel(Services{})
	model.activeBucket = "test-bucket"
	model.explorerView.Objects = []objDomain.Object{
		{Key: "documents/spec.pdf", Size: 2048},
	}
	model.explorerView.SelectedObject = 0

	mP, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	uP := mP.(Model)
	assert.True(t, uP.presignModal.Active)
	assert.Equal(t, "documents/spec.pdf", uP.presignModal.ObjectKey)
	viewP := uP.View()
	assert.Contains(t, viewP, "GENERATE PRESIGNED URL")

	initialPreview := model.explorerView.ShowPreview
	mI, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	uI := mI.(Model)
	assert.Equal(t, !initialPreview, uI.explorerView.ShowPreview)
}
