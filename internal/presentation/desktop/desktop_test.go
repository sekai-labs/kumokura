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
