package desktop

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	accountDomain "github.com/sekai-labs/kumokura/internal/accounts/domain"
	accountPorts "github.com/sekai-labs/kumokura/internal/accounts/ports"
	"github.com/sekai-labs/kumokura/internal/bootstrap"
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
