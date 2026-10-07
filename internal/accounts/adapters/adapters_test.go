package adapters_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sekai-labs/kumokura/internal/accounts/adapters"
	"github.com/sekai-labs/kumokura/internal/accounts/domain"
	"github.com/sekai-labs/kumokura/internal/accounts/ports"
	"github.com/sekai-labs/kumokura/internal/platform/database"
	"github.com/sekai-labs/kumokura/internal/security/keyring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteAccountRepositoryCRUD(t *testing.T) {
	tmp := t.TempDir()
	db, err := database.Open(filepath.Join(tmp, "accounts.db"))
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	err = db.Migrate(ctx)
	require.NoError(t, err)

	repo := adapters.NewSQLiteAccountRepository(db.DB)

	acc, err := domain.NewAccount("acc-1", "Test Account", domain.TypeAWS, "", "us-west-2", false)
	require.NoError(t, err)

	err = repo.Save(ctx, acc)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, acc.ID, found.ID)
	assert.Equal(t, "Test Account", found.Name)
	assert.Equal(t, domain.TypeAWS, found.Type)
	assert.Equal(t, "us-west-2", found.Region)
	assert.False(t, found.UsePathStyle)

	foundByName, err := repo.FindByName(ctx, "Test Account")
	require.NoError(t, err)
	assert.Equal(t, acc.ID, foundByName.ID)

	acc.Name = "Updated Name"
	acc.Region = "eu-central-1"
	acc.UsePathStyle = true
	err = repo.Update(ctx, acc)
	require.NoError(t, err)

	updated, err := repo.FindByID(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, "eu-central-1", updated.Region)
	assert.True(t, updated.UsePathStyle)

	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	err = repo.Delete(ctx, acc.ID)
	require.NoError(t, err)

	_, err = repo.FindByID(ctx, acc.ID)
	assert.ErrorIs(t, err, ports.ErrAccountNotFound)

	_, err = repo.FindByName(ctx, "Updated Name")
	assert.ErrorIs(t, err, ports.ErrAccountNotFound)

	err = repo.Delete(ctx, acc.ID)
	assert.ErrorIs(t, err, ports.ErrAccountNotFound)
}

func TestKeyringCredentialAdapter(t *testing.T) {
	memStore := keyring.NewMemoryStore()
	adapter := adapters.NewKeyringCredentialAdapter(memStore)

	ctx := context.Background()
	accID := domain.AccountID("acc-123")
	creds, err := domain.NewCredentials("access-1", "secret-1", "session-1")
	require.NoError(t, err)

	err = adapter.Store(ctx, accID, creds)
	require.NoError(t, err)

	retrieved, err := adapter.Retrieve(ctx, accID)
	require.NoError(t, err)
	assert.Equal(t, creds.AccessKeyID, retrieved.AccessKeyID)
	assert.Equal(t, creds.SecretAccessKey, retrieved.SecretAccessKey)
	assert.Equal(t, creds.SessionToken, retrieved.SessionToken)

	err = adapter.Remove(ctx, accID)
	require.NoError(t, err)

	_, err = adapter.Retrieve(ctx, accID)
	assert.Error(t, err)
}
