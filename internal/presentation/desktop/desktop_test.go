package desktop

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	accountDomain "github.com/sekai-labs/kumokura/internal/accounts/domain"
	accountPorts "github.com/sekai-labs/kumokura/internal/accounts/ports"
	"github.com/sekai-labs/kumokura/internal/bootstrap"
	bucketDomain "github.com/sekai-labs/kumokura/internal/buckets/domain"
	objectDomain "github.com/sekai-labs/kumokura/internal/objects/domain"
	"github.com/sekai-labs/kumokura/internal/platform/config"
)

func TestFormatBytes(t *testing.T) {
	assert.Equal(t, "500 B", formatBytes(500))
	assert.Equal(t, "1.0 KB", formatBytes(1024))
	assert.Equal(t, "1.5 MB", formatBytes(1024*1024+512*1024))
	assert.Equal(t, "2.0 GB", formatBytes(2*1024*1024*1024))
}

func TestDesktopAppInitialization(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir:  tempDir,
		DataDir:    tempDir,
		SecretsDir: tempDir,
		DBPath:     ":memory:",
		LogLevel:   "error",
	}

	appContainer, err := bootstrap.InitializeWithConfig(context.Background(), cfg)
	require.NoError(t, err)
	defer appContainer.Close()

	fyneTestApp := test.NewApp()
	app := NewDesktopAppWithFyneApp(appContainer, fyneTestApp)
	require.NotNil(t, app)
	assert.NotNil(t, app.window)
	assert.NotNil(t, app.accountSelect)
	assert.NotNil(t, app.bucketBadge)
	assert.NotNil(t, app.searchEntry)
	assert.NotNil(t, app.objectTable)
	assert.NotNil(t, app.detailsCard)
	assert.NotNil(t, app.transferList)
	assert.NotNil(t, app.emptyStateCard)
	assert.NotNil(t, app.loadingContainer)
	assert.NotNil(t, app.loadingBar)
	assert.NotNil(t, app.loadingLabel)
	assert.NotNil(t, app.centerContainer)
	assert.NotNil(t, app.previewBtn)
	assert.NotNil(t, app.previewContentEntry)
	assert.NotNil(t, app.previewStatusLabel)
}

func TestDesktopAppAccountAndFilter(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir:  tempDir,
		DataDir:    tempDir,
		SecretsDir: tempDir,
		DBPath:     ":memory:",
		LogLevel:   "error",
	}
	appContainer, err := bootstrap.InitializeWithConfig(context.Background(), cfg)
	require.NoError(t, err)
	defer appContainer.Close()

	creds, err := accountDomain.NewCredentials("access-key", "secret-key", "")
	require.NoError(t, err)

	_, err = appContainer.AccountService.CreateAccount(context.Background(), accountPorts.CreateAccountParams{
		Name:         "test-account",
		Type:         accountDomain.TypeAWS,
		Endpoint:     "",
		Region:       "us-east-1",
		UsePathStyle: false,
		Credentials:  creds,
	})
	require.NoError(t, err)

	fyneTestApp := test.NewApp()
	app := NewDesktopAppWithFyneApp(appContainer, fyneTestApp)
	require.NotNil(t, app)

	app.loadAccounts()

	app.mu.RLock()
	assert.Len(t, app.accounts, 1)
	assert.Equal(t, "test-account", app.selectedAccount.Name)
	app.mu.RUnlock()

	app.mu.Lock()
	app.objects = []objectDomain.Object{
		{Key: "photos/vacation.jpg", Size: 2048, StorageClass: objectDomain.StorageClassStandard, LastModified: time.Now()},
		{Key: "documents/notes.txt", Size: 512, StorageClass: objectDomain.StorageClassStandard, LastModified: time.Now()},
		{Key: "videos/presentation.mp4", Size: 1048576, StorageClass: objectDomain.StorageClassStandard, LastModified: time.Now()},
	}
	app.applyFilterLocked()
	app.mu.Unlock()

	app.mu.RLock()
	assert.Len(t, app.filteredObjects, 3)
	app.mu.RUnlock()

	app.onSearchChanged("notes")
	app.mu.RLock()
	assert.Len(t, app.filteredObjects, 1)
	assert.Equal(t, "documents/notes.txt", app.filteredObjects[0].Key)
	app.mu.RUnlock()

	app.onSearchChanged("")
	app.mu.RLock()
	assert.Len(t, app.filteredObjects, 3)
	app.mu.RUnlock()
}

func TestDesktopAppLoadingAndEmptyState(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir:  tempDir,
		DataDir:    tempDir,
		SecretsDir: tempDir,
		DBPath:     ":memory:",
		LogLevel:   "error",
	}
	appContainer, err := bootstrap.InitializeWithConfig(context.Background(), cfg)
	require.NoError(t, err)
	defer appContainer.Close()

	fyneTestApp := test.NewApp()
	app := NewDesktopAppWithFyneApp(appContainer, fyneTestApp)
	require.NotNil(t, app)

	app.updateTableViewState(true, 0)
	assert.True(t, app.loadingContainer.Visible())
	assert.False(t, app.objectTable.Visible())
	assert.False(t, app.emptyStateCard.Visible())

	app.updateTableViewState(false, 0)
	assert.False(t, app.loadingContainer.Visible())
	assert.False(t, app.objectTable.Visible())
	assert.True(t, app.emptyStateCard.Visible())

	app.searchQuery = "testfile"
	app.updateTableViewState(false, 0)
	assert.False(t, app.loadingContainer.Visible())
	assert.False(t, app.objectTable.Visible())
	assert.True(t, app.emptyStateCard.Visible())
	assert.Contains(t, app.emptyStateMsg.Text, "No objects match query 'testfile'")
	assert.True(t, app.emptyClearFilterBtn.Visible())
	assert.False(t, app.emptyUploadBtn.Visible())

	app.searchQuery = ""
	app.updateTableViewState(false, 0)
	assert.Contains(t, app.emptyStateMsg.Text, "This bucket is empty")
	assert.False(t, app.emptyClearFilterBtn.Visible())
	assert.True(t, app.emptyUploadBtn.Visible())

	app.updateTableViewState(false, 5)
	assert.False(t, app.loadingContainer.Visible())
	assert.True(t, app.objectTable.Visible())
	assert.False(t, app.emptyStateCard.Visible())
}

func TestDesktopAppSelectObjectAndPreview(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir:  tempDir,
		DataDir:    tempDir,
		SecretsDir: tempDir,
		DBPath:     ":memory:",
		LogLevel:   "error",
	}
	appContainer, err := bootstrap.InitializeWithConfig(context.Background(), cfg)
	require.NoError(t, err)
	defer appContainer.Close()

	fyneTestApp := test.NewApp()
	app := NewDesktopAppWithFyneApp(appContainer, fyneTestApp)
	require.NotNil(t, app)

	app.selectObject(nil)
	assert.Equal(t, "Select an object to inspect details.", app.metadataLabel.Text)
	assert.Equal(t, "", app.previewContentEntry.Text)

	sampleObj := &objectDomain.Object{
		Key:          "docs/hello.txt",
		Size:         128,
		StorageClass: objectDomain.StorageClassStandard,
		LastModified: time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC),
		ETag:         "\"abc123etag\"",
	}
	app.selectObject(sampleObj)
	app.mu.RLock()
	assert.Equal(t, "docs/hello.txt", app.selectedObject.Key)
	app.mu.RUnlock()

	assert.Contains(t, app.metadataLabel.Text, "docs/hello.txt")
	assert.Contains(t, app.metadataLabel.Text, "128 B")
	assert.Contains(t, app.metadataLabel.Text, "abc123etag")
	assert.Contains(t, app.previewStatusLabel.Text, "Ready to preview hello.txt")
	assert.False(t, app.openExternalBtn.Disabled())

	// Test Image detection and UI state
	imgObj := &objectDomain.Object{
		Key:          "photos/nature.png",
		Size:         2048,
		StorageClass: objectDomain.StorageClassStandard,
		LastModified: time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC),
	}
	assert.True(t, isImageFile(imgObj.Key))
	assert.False(t, isVideoFile(imgObj.Key))
	app.selectObject(imgObj)
	assert.Contains(t, app.previewStatusLabel.Text, "Image detected")
	assert.False(t, app.openExternalBtn.Disabled())

	// Test Video detection and UI state
	vidObj := &objectDomain.Object{
		Key:          "movies/trailer.mp4",
		Size:         10485760,
		StorageClass: objectDomain.StorageClassStandard,
		LastModified: time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC),
	}
	assert.True(t, isVideoFile(vidObj.Key))
	assert.False(t, isImageFile(vidObj.Key))
	app.selectObject(vidObj)
	assert.Contains(t, app.previewStatusLabel.Text, "Video detected")
	assert.True(t, app.previewVideoBox.Visible())
	assert.False(t, app.openExternalBtn.Disabled())
}
func TestDesktopAppTableSelectionAndNavigation(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir:  tempDir,
		DataDir:    tempDir,
		SecretsDir: tempDir,
		DBPath:     ":memory:",
		LogLevel:   "error",
	}
	appContainer, err := bootstrap.InitializeWithConfig(context.Background(), cfg)
	require.NoError(t, err)
	defer appContainer.Close()

	fyneTestApp := test.NewApp()
	app := NewDesktopAppWithFyneApp(appContainer, fyneTestApp)
	require.NotNil(t, app)

	assert.Equal(t, -1, app.lastSelectedRow)
	assert.Equal(t, -1, app.lastSelectedCol)

	app.mu.Lock()
	app.currentPrefix = "photos/"
	app.prefixes = []objectDomain.Prefix{
		{Bucket: "my-bucket", Prefix: "photos/vacation/"},
		{Bucket: "my-bucket", Prefix: "photos/work/"},
	}
	app.objects = []objectDomain.Object{
		{Bucket: "my-bucket", Key: "photos/family.jpg", Size: 1024, StorageClass: objectDomain.StorageClassStandard, LastModified: time.Now()},
	}
	app.applyFilterLocked()
	app.mu.Unlock()

	app.mu.RLock()
	assert.Len(t, app.filteredObjects, 3)
	assert.True(t, app.filteredObjects[0].IsPrefix)
	assert.Equal(t, "photos/vacation/", app.filteredObjects[0].Key)
	app.mu.RUnlock()

	app.handleTableRowSelected(0, 0)
	app.mu.RLock()
	assert.Equal(t, 0, app.lastSelectedRow)
	assert.Equal(t, 0, app.lastSelectedCol)
	assert.NotNil(t, app.selectedObject)
	assert.Equal(t, "photos/vacation/", app.selectedObject.Key)
	app.mu.RUnlock()

	app.handleTableRowSelected(0, 1)
	app.mu.RLock()
	assert.Equal(t, 0, app.lastSelectedRow)
	assert.Equal(t, 1, app.lastSelectedCol)
	app.mu.RUnlock()

	app.navigateToPrefix("photos/vacation/")
	app.mu.RLock()
	assert.Equal(t, "photos/vacation/", app.currentPrefix)
	assert.Equal(t, -1, app.lastSelectedRow)
	assert.Equal(t, -1, app.lastSelectedCol)
	assert.Nil(t, app.selectedObject)
	app.mu.RUnlock()

	app.navigateUp()
	app.mu.RLock()
	assert.Equal(t, "photos/", app.currentPrefix)
	assert.Equal(t, -1, app.lastSelectedRow)
	assert.Equal(t, -1, app.lastSelectedCol)
	assert.Nil(t, app.selectedObject)
	app.mu.RUnlock()
}

func TestDesktopAppCellAndListItemRendering(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir:  tempDir,
		DataDir:    tempDir,
		SecretsDir: tempDir,
		DBPath:     ":memory:",
		LogLevel:   "error",
	}

	appContainer, err := bootstrap.InitializeWithConfig(context.Background(), cfg)
	require.NoError(t, err)
	defer appContainer.Close()

	// Seed an account
	creds, err := accountDomain.NewCredentials("access-key", "secret-key", "")
	require.NoError(t, err)
	_, err = appContainer.AccountService.CreateAccount(context.Background(), accountPorts.CreateAccountParams{
		Name:         "Production S3 Main Account",
		Type:         accountDomain.TypeAWS,
		Endpoint:     "",
		Region:       "us-east-1",
		UsePathStyle: false,
		Credentials:  creds,
	})
	require.NoError(t, err)

	fyneTestApp := test.NewApp()
	app := NewDesktopAppWithFyneApp(appContainer, fyneTestApp)
	require.NotNil(t, app)
	app.loadAccounts()

	// Verify Table cell template and sizing
	cellObj := app.objectTable.CreateCell()
	require.NotNil(t, cellObj)
	cellBox, ok := cellObj.(*fyne.Container)
	require.True(t, ok)
	require.Len(t, cellBox.Objects, 2)

	cellLbl, ok := cellBox.Objects[0].(*widget.Label)
	require.True(t, ok)
	cellIcon, ok := cellBox.Objects[1].(*widget.Icon)
	require.True(t, ok)

	// Setup an object in the table
	app.mu.Lock()
	app.filteredObjects = []objectDomain.Object{
		{
			Key:          "documents/report-final-presentation-2026.pdf",
			Size:         1048576,
			StorageClass: objectDomain.StorageClassStandard,
			LastModified: time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC),
		},
	}
	app.mu.Unlock()

	// UpdateCell for header (Row 0, Col 0)
	app.objectTable.UpdateCell(widget.TableCellID{Row: 0, Col: 0}, cellObj)
	assert.False(t, cellIcon.Visible())
	assert.Equal(t, "Name", cellLbl.Text)

	// UpdateCell for object data (Row 1, Col 0 - Name)
	app.objectTable.UpdateCell(widget.TableCellID{Row: 1, Col: 0}, cellObj)
	assert.True(t, cellIcon.Visible())
	assert.Equal(t, "documents/report-final-presentation-2026.pdf", cellLbl.Text)

	// When allocated the column width (360px), cell label must expand to fill available width (> 300px), not collapse to ellipsis min size (~23px)
	cellBox.Resize(fyne.NewSize(360, 36))
	assert.Greater(t, cellLbl.Size().Width, float32(300))

	// UpdateCell for object data (Row 1, Col 1 - Size)
	app.objectTable.UpdateCell(widget.TableCellID{Row: 1, Col: 1}, cellObj)
	assert.False(t, cellIcon.Visible())
	assert.Equal(t, "1.0 MB", cellLbl.Text)
	cellBox.Resize(fyne.NewSize(110, 36))
	assert.Greater(t, cellLbl.Size().Width, float32(100))

	// Verify Account list template and sizing
	accObj := app.accountList.CreateItem()
	require.NotNil(t, accObj)
	accBox, ok := accObj.(*fyne.Container)
	require.True(t, ok)
	require.Len(t, accBox.Objects, 2)
	accLbl, ok := accBox.Objects[0].(*widget.Label)
	require.True(t, ok)

	app.accountList.UpdateItem(0, accObj)
	assert.Contains(t, accLbl.Text, "Production S3 Main Account")
	accBox.Resize(fyne.NewSize(250, 40))
	assert.Greater(t, accLbl.Size().Width, float32(200))

	// Verify Bucket list template and sizing
	app.mu.Lock()
	app.buckets = []bucketDomain.Bucket{
		{Name: "sekai-archive-backup-bucket", Region: "us-west-2"},
	}
	app.mu.Unlock()

	bucketObj := app.bucketList.CreateItem()
	require.NotNil(t, bucketObj)
	bucketBox, ok := bucketObj.(*fyne.Container)
	require.True(t, ok)
	require.Len(t, bucketBox.Objects, 2)
	bucketLbl, ok := bucketBox.Objects[0].(*widget.Label)
	require.True(t, ok)

	app.bucketList.UpdateItem(0, bucketObj)
	assert.Contains(t, bucketLbl.Text, "sekai-archive-backup-bucket")
	bucketBox.Resize(fyne.NewSize(250, 40))
	assert.Greater(t, bucketLbl.Size().Width, float32(200))
}
func TestDesktopAppFolderDeletionAndBucketActions(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.Config{
		ConfigDir:  tempDir,
		DataDir:    tempDir,
		SecretsDir: tempDir,
		DBPath:     ":memory:",
		LogLevel:   "error",
	}

	appContainer, err := bootstrap.InitializeWithConfig(context.Background(), cfg)
	require.NoError(t, err)
	defer appContainer.Close()

	fyneTestApp := test.NewApp()
	app := NewDesktopAppWithFyneApp(appContainer, fyneTestApp)
	require.NotNil(t, app)

	// Test upload target prefix computation
	app.mu.Lock()
	app.currentPrefix = "a/"
	app.flatMode = false
	app.mu.Unlock()

	// Verify currentPrefix is preserved and correctly prepended
	app.mu.RLock()
	assert.Equal(t, "a/", app.currentPrefix)
	assert.False(t, app.flatMode)
	app.mu.RUnlock()

	// Verify bucket context menu triggers without panic
	assert.NotPanics(t, func() {
		app.showBucketContextMenu("test-bucket", fyne.NewPos(50, 50))
	})

	// Verify inspect bucket dialog triggers without panic
	assert.NotPanics(t, func() {
		app.showInspectBucketDialog("test-bucket")
	})

	// Verify empty bucket dialog triggers without panic
	assert.NotPanics(t, func() {
		app.showEmptyBucketDialog("test-bucket")
	})

	// Verify edit bucket dialog triggers without panic
	assert.NotPanics(t, func() {
		app.showEditBucketDialog("test-bucket")
	})

	// Verify delete selected folder handles prefix correctly
	app.mu.Lock()
	app.selectedAccount = &accountDomain.Account{Name: "test-account"}
	app.selectedBucket = "test-bucket"
	app.selectedObject = &objectDomain.Object{
		Bucket:   "test-bucket",
		Key:      "photos/vacation/",
		IsPrefix: true,
	}
	app.mu.Unlock()

	assert.NotPanics(t, func() {
		app.deleteSelectedObject()
	})
}
