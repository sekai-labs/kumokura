package database_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sekai-labs/kumokura/internal/platform/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAndMigrate(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")

	db, err := database.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	err = db.Migrate(ctx)
	require.NoError(t, err)

	err = db.Migrate(ctx)
	require.NoError(t, err)

	tables := []string{
		"schema_migrations",
		"accounts",
		"transfer_jobs",
		"sync_jobs",
		"favorites",
		"history",
	}

	for _, tbl := range tables {
		var name string
		err := db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name=?", tbl).Scan(&name)
		require.NoError(t, err, "table %s should exist", tbl)
		assert.Equal(t, tbl, name)
	}

	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(1) FROM schema_migrations WHERE version=1").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
