package keyring_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/sekai-labs/kumokura/internal/security/keyring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore(t *testing.T) {
	store := keyring.NewMemoryStore()

	_, err := store.Get("service", "user1")
	assert.ErrorIs(t, err, keyring.ErrNotFound)

	err = store.Set("service", "user1", "secret-value")
	require.NoError(t, err)

	val, err := store.Get("service", "user1")
	require.NoError(t, err)
	assert.Equal(t, "secret-value", val)

	err = store.Delete("service", "user1")
	require.NoError(t, err)

	_, err = store.Get("service", "user1")
	assert.ErrorIs(t, err, keyring.ErrNotFound)

	err = store.Delete("service", "user1")
	assert.ErrorIs(t, err, keyring.ErrNotFound)
}

func TestFileEncryptedStore(t *testing.T) {
	tmp := t.TempDir()
	store := keyring.NewFileEncryptedStore(filepath.Join(tmp, "secrets"))

	_, err := store.Get("svc", "usr")
	assert.ErrorIs(t, err, keyring.ErrNotFound)

	err = store.Set("svc", "usr", "my-secret-token")
	require.NoError(t, err)

	val, err := store.Get("svc", "usr")
	require.NoError(t, err)
	assert.Equal(t, "my-secret-token", val)

	store2 := keyring.NewFileEncryptedStore(filepath.Join(tmp, "secrets"))
	val2, err := store2.Get("svc", "usr")
	require.NoError(t, err)
	assert.Equal(t, "my-secret-token", val2)

	err = store.Delete("svc", "usr")
	require.NoError(t, err)

	_, err = store.Get("svc", "usr")
	assert.ErrorIs(t, err, keyring.ErrNotFound)
}

func TestSecureKeyringFallback(t *testing.T) {
	tmp := t.TempDir()
	k := keyring.NewSecureKeyring(filepath.Join(tmp, "secrets"))

	err := k.Set("kumokura-test", "acc-1", "test-secret-data")
	require.NoError(t, err)

	val, err := k.Get("kumokura-test", "acc-1")
	require.NoError(t, err)
	assert.Equal(t, "test-secret-data", val)

	err = k.Delete("kumokura-test", "acc-1")
	require.NoError(t, err)

	_, err = k.Get("kumokura-test", "acc-1")
	assert.True(t, errors.Is(err, keyring.ErrNotFound))
}
