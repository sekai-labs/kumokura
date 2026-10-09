package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	bucketDomain "github.com/sekai-labs/kumokura/internal/buckets/domain"
	objDomain "github.com/sekai-labs/kumokura/internal/objects/domain"
	objPorts "github.com/sekai-labs/kumokura/internal/objects/ports"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/messages"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
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
	assert.True(t, updated.yaziPicker.Active)
	assert.Contains(t, updated.yaziPicker.TargetBucket, "test-bucket")

	view := updated.View()
	assert.Contains(t, view, "YAZI FILE/FOLDER SELECTOR")
	assert.Contains(t, view, "s3://test-bucket/")
	mEsc, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	closed := mEsc.(Model)
	assert.False(t, closed.yaziPicker.Active)
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
func TestTUI_CleanLocalInputPath(t *testing.T) {
	assert.Equal(t, "/tmp/folder", cleanLocalInputPath("\"/tmp/folder\""))
	assert.Equal(t, "/tmp/folder", cleanLocalInputPath("'/tmp/folder'"))
	assert.Equal(t, "/tmp/folder", cleanLocalInputPath("  \"/tmp/folder\"  "))

	cleanTilde := cleanLocalInputPath("~/test")
	assert.NotContains(t, cleanTilde, "~")
	assert.True(t, strings.HasSuffix(cleanTilde, "test"))
}

func TestTUI_FolderMarkerNavigationAndPreviewClear(t *testing.T) {
	model := NewModel(Services{})
	model.activeBucket = "test-bucket"
	model.explorerView.ActivePaneIndex = 1
	model.explorerView.Prefixes = []objDomain.Prefix{}
	model.explorerView.Objects = []objDomain.Object{
		{Key: "photos/", Size: 0},
		{Key: "notes.txt", Size: 120},
	}
	model.explorerView.PreviewContent = []byte("stale note content")

	model.explorerView.SelectedObject = 0
	mEnter, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	uEnter := mEnter.(Model)
	assert.Equal(t, "photos/", uEnter.explorerView.CurrentPrefix)
	assert.Equal(t, 0, uEnter.explorerView.SelectedObject)
	assert.Nil(t, uEnter.explorerView.PreviewContent)

	mBack, _ := uEnter.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	uBack := mBack.(Model)
	assert.Equal(t, "", uBack.explorerView.CurrentPrefix)
	assert.Nil(t, uBack.explorerView.PreviewContent)

	mL, _ := uBack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	uL := mL.(Model)
	assert.Equal(t, "photos/", uL.explorerView.CurrentPrefix)

	mH, _ := uL.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	uH := mH.(Model)
	assert.Equal(t, "", uH.explorerView.CurrentPrefix)
}

func TestTUI_MultiSelectAndBulkActions(t *testing.T) {
	model := NewModel(Services{})
	model.activeTab = 1
	model.explorerView.ActivePaneIndex = 1
	model.activeBucket = "test-b"
	model.explorerView.ActiveBucket = "test-b"
	model.explorerView.Objects = []objDomain.Object{
		{Key: "item1.jpg", Size: 100},
		{Key: "item2.png", Size: 200},
		{Key: "item3.txt", Size: 300},
	}

	// Test Space selection toggle
	model.explorerView.SelectedObject = 0
	m1, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	u1 := m1.(Model)
	assert.True(t, u1.explorerView.SelectedKeys["item1.jpg"])

	// Toggle it off
	m2, _ := u1.Update(tea.KeyMsg{Type: tea.KeySpace})
	u2 := m2.(Model)
	assert.False(t, u2.explorerView.SelectedKeys["item1.jpg"])

	// Select all ('a')
	mAll, _ := u2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	uAll := mAll.(Model)
	assert.True(t, uAll.explorerView.SelectedKeys["item1.jpg"])
	assert.True(t, uAll.explorerView.SelectedKeys["item2.png"])
	assert.True(t, uAll.explorerView.SelectedKeys["item3.txt"])

	// Clear selection ('c')
	mClear, _ := uAll.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	uClear := mClear.(Model)
	assert.Empty(t, uClear.explorerView.SelectedKeys)

	// Bulk Delete prompt ('x')
	uAll.explorerView.SelectedKeys = map[string]bool{"item1.jpg": true, "item2.png": true}
	mDel, _ := uAll.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	uDel := mDel.(Model)
	assert.True(t, uDel.showDeleteModal)
	assert.Len(t, uDel.deleteTargetKeys, 2)
	viewDel := uDel.View()
	assert.Contains(t, viewDel, "Permanently delete 2 selected objects?")

	// Bulk Download modal ('d')
	mDl, _ := uAll.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	uDl := mDl.(Model)
	assert.True(t, uDl.downloadModal.Active)
	assert.Len(t, uDl.downloadTargets, 2)

	// Bulk Presign modal ('p')
	mP, _ := uAll.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	uP := mP.(Model)
	assert.True(t, uP.presignModal.Active)
	assert.Len(t, uP.presignTargetKeys, 2)
}

func TestTUI_OpenMediaKey(t *testing.T) {
	model := NewModel(Services{})
	model.activeTab = 1
	model.explorerView.ActivePaneIndex = 1
	model.activeBucket = "test-b"
	model.explorerView.ActiveBucket = "test-b"
	model.explorerView.Objects = []objDomain.Object{
		{Key: "video.mp4", Size: 1048576},
	}
	model.explorerView.SelectedObject = 0

	mOpen, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	uOpen := mOpen.(Model)
	assert.NotNil(t, cmd)
	assert.Contains(t, uOpen.notification, "Opening video.mp4 with system player")
}

func TestTUI_BucketActionsModal(t *testing.T) {
	model := NewModel(Services{})
	model.explorerView.Buckets = []bucketDomain.Bucket{
		{Name: "my-test-bucket", Region: "us-east-1"},
	}
	model.explorerView.SelectedBucket = 0
	model.explorerView.ActivePaneIndex = 0

	// Press 'i' on bucket list
	mI, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	uI := mI.(Model)
	assert.True(t, uI.bucketActionsModal.Active)
	assert.Equal(t, "my-test-bucket", uI.bucketActionsModal.BucketName)

	view := uI.View()
	assert.Contains(t, view, "BUCKET ACTIONS: my-test-bucket")
	assert.Contains(t, view, "1. Bucket Properties / Info")
	assert.Contains(t, view, "2. Delete Bucket")
	assert.Contains(t, view, "3. Empty Bucket")
	assert.Contains(t, view, "4. Bucket Configuration / Edit")

	// Navigate down with 'j'
	mDown, _ := uI.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	uDown := mDown.(Model)
	assert.Equal(t, 1, uDown.bucketActionsModal.SelectedIdx)
	assert.Equal(t, "delete", uDown.bucketActionsModal.SelectedOption().ID)

	// Navigate down to 'empty'
	mDown2, _ := uDown.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	uDown2 := mDown2.(Model)
	assert.Equal(t, 2, uDown2.bucketActionsModal.SelectedIdx)
	assert.Equal(t, "empty", uDown2.bucketActionsModal.SelectedOption().ID)

	// Select 'Empty Bucket' with Enter
	mEnter, _ := uDown2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	uEnter := mEnter.(Model)
	assert.False(t, uEnter.bucketActionsModal.Active)
	assert.True(t, uEnter.showBucketEmptyModal)
	assert.Contains(t, uEnter.View(), "CONFIRM EMPTY BUCKET")

	// Cancel with Esc
	mEsc, _ := uEnter.Update(tea.KeyMsg{Type: tea.KeyEsc})
	uEsc := mEsc.(Model)
	assert.False(t, uEsc.showBucketEmptyModal)
}

func TestTUI_BucketActionsModal_EnterKey(t *testing.T) {
	model := NewModel(Services{})
	model.explorerView.Buckets = []bucketDomain.Bucket{
		{Name: "prod-bucket", Region: "us-west-2"},
	}
	model.explorerView.SelectedBucket = 0
	model.explorerView.ActivePaneIndex = 0

	// Pressing Enter on bucket list opens the Bucket Actions Modal
	mEnter, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	uEnter := mEnter.(Model)
	assert.True(t, uEnter.bucketActionsModal.Active)
	assert.Equal(t, "prod-bucket", uEnter.bucketActionsModal.BucketName)

	// Pressing 'Esc' cancels modal
	mEsc, _ := uEnter.Update(tea.KeyMsg{Type: tea.KeyEsc})
	uEsc := mEsc.(Model)
	assert.False(t, uEsc.bucketActionsModal.Active)
}

func TestTUI_UploadTargetRouting_CursorFolder(t *testing.T) {
	model := NewModel(Services{})
	model.activeBucket = "my-bucket"
	model.explorerView.ActiveBucket = "my-bucket"
	model.explorerView.ActivePaneIndex = 1
	model.explorerView.CurrentPrefix = "a/"
	model.explorerView.Prefixes = []objDomain.Prefix{
		{Prefix: "a/b/"},
	}
	model.explorerView.SelectedObject = 0 // Cursor is on folder "a/b/"

	// Target prefix calculation
	targetPrefix := model.explorerView.GetCurrentTargetPrefix()
	assert.Equal(t, "a/b/", targetPrefix)

	// Press 'u' to open upload
	mU, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	uU := mU.(Model)
	assert.True(t, uU.yaziPicker.Active)
	assert.Equal(t, "a/b/", uU.yaziPicker.TargetPrefix)
	assert.Contains(t, uU.uploadModal.Destination, "s3://my-bucket/a/b/")
}
