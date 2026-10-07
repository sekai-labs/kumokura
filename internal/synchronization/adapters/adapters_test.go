package adapters_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sekai-labs/kumokura/internal/platform/database"
	"github.com/sekai-labs/kumokura/internal/synchronization/adapters"
	"github.com/sekai-labs/kumokura/internal/synchronization/domain"
	"github.com/sekai-labs/kumokura/internal/synchronization/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalScanner(t *testing.T) {
	tmp := t.TempDir()

	err := os.WriteFile(filepath.Join(tmp, "file1.txt"), []byte("hello world"), 0644)
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(tmp, "sub"), 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmp, "sub", "file2.log"), []byte("log entry"), 0644)
	require.NoError(t, err)

	scanner := adapters.NewLocalScanner(true)
	ctx := context.Background()

	entries, err := scanner.Scan(ctx, tmp, domain.Filter{
		Excludes: []string{"*.log"},
	})
	require.NoError(t, err)

	assert.Len(t, entries, 1)
	assert.Contains(t, entries, "file1.txt")
	assert.NotEmpty(t, entries["file1.txt"].Checksum)
	assert.Equal(t, int64(11), entries["file1.txt"].Size)
}

func TestSQLiteSyncRepository(t *testing.T) {
	tmp := t.TempDir()
	db, err := database.Open(filepath.Join(tmp, "sync.db"))
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	err = db.Migrate(ctx)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
INSERT INTO accounts (id, name, account_type)
VALUES ('acc-1', 'Test Acc', 'AWS')
`)
	require.NoError(t, err)

	repo := adapters.NewSQLiteSyncRepository(db.DB)

	job := &ports.SyncJobRecord{
		ID:               "job-1",
		AccountID:        "acc-1",
		SourceBucket:     "source-bkt",
		SourcePrefix:     "src/",
		DestinationPath:  "/tmp/dest",
		SyncDirection:    domain.DirectionS3ToLocal,
		DeleteExtraneous: true,
		Status:           "PENDING",
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	err = repo.SaveJob(ctx, job)
	require.NoError(t, err)

	retrieved, err := repo.GetJob(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, "job-1", retrieved.ID)
	assert.Equal(t, "source-bkt", retrieved.SourceBucket)
	assert.True(t, retrieved.DeleteExtraneous)

	err = repo.UpdateStatus(ctx, "job-1", "COMPLETED")
	require.NoError(t, err)

	updated, err := repo.GetJob(ctx, "job-1")
	require.NoError(t, err)
	assert.Equal(t, "COMPLETED", updated.Status)

	jobs, err := repo.ListJobs(ctx, "acc-1")
	require.NoError(t, err)
	assert.Len(t, jobs, 1)

	err = repo.DeleteJob(ctx, "job-1")
	require.NoError(t, err)

	_, err = repo.GetJob(ctx, "job-1")
	assert.Error(t, err)
}
