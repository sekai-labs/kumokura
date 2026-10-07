package adapters

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sekai-labs/kumokura/internal/platform/database"
	"github.com/sekai-labs/kumokura/internal/transfers/domain"
	"github.com/sekai-labs/kumokura/internal/transfers/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *database.DB {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.Open(dbPath)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Migrate(ctx)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
INSERT INTO accounts (id, name, account_type) VALUES ('acc-1', 'Test Acc', 'aws')
`)
	require.NoError(t, err)

	return db
}

func TestSQLiteTransferRepository_JobOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteTransferRepository(db)
	ctx := context.Background()

	job := domain.TransferJob{
		ID:               "job-101",
		AccountID:        "acc-1",
		Type:             domain.TransferTypeUpload,
		SourcePath:       "/tmp/file.txt",
		DestinationPath:  "s3://my-bucket/file.txt",
		Status:           domain.JobStatusPending,
		BytesTransferred: 0,
		TotalBytes:       1024,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := repo.SaveJob(ctx, job)
	assert.NoError(t, err)

	retrieved, err := repo.GetJob(ctx, "job-101")
	assert.NoError(t, err)
	assert.Equal(t, job.ID, retrieved.ID)
	assert.Equal(t, domain.JobStatusPending, retrieved.Status)

	err = repo.UpdateJobStatus(ctx, "job-101", domain.JobStatusRunning, "")
	assert.NoError(t, err)

	err = repo.UpdateJobProgress(ctx, "job-101", 512)
	assert.NoError(t, err)

	updated, err := repo.GetJob(ctx, "job-101")
	assert.NoError(t, err)
	assert.Equal(t, domain.JobStatusRunning, updated.Status)
	assert.Equal(t, int64(512), updated.BytesTransferred)
}

func TestSQLiteTransferRepository_Checkpoints(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteTransferRepository(db)
	ctx := context.Background()

	now := time.Now()
	checkpoints := []domain.TransferPart{
		{
			JobID:      "job-chk-1",
			PartNumber: 1,
			Offset:     0,
			Size:       5242880,
			Status:     domain.PartStatusPending,
			UpdatedAt:  now,
		},
		{
			JobID:      "job-chk-1",
			PartNumber: 2,
			Offset:     5242880,
			Size:       5242880,
			Status:     domain.PartStatusPending,
			UpdatedAt:  now,
		},
	}

	err := repo.SaveCheckpoints(ctx, checkpoints)
	assert.NoError(t, err)

	retrieved, err := repo.GetCheckpoints(ctx, "job-chk-1")
	assert.NoError(t, err)
	assert.Len(t, retrieved, 2)

	checkpoints[0].Status = domain.PartStatusCompleted
	checkpoints[0].ETag = "\"etag-part-1\""
	err = repo.UpdateCheckpoint(ctx, checkpoints[0])
	assert.NoError(t, err)

	retrievedUpdated, err := repo.GetCheckpoints(ctx, "job-chk-1")
	assert.NoError(t, err)
	assert.Equal(t, domain.PartStatusCompleted, retrievedUpdated[0].Status)
	assert.Equal(t, "\"etag-part-1\"", retrievedUpdated[0].ETag)

	err = repo.DeleteCheckpoints(ctx, "job-chk-1")
	assert.NoError(t, err)

	afterDelete, err := repo.GetCheckpoints(ctx, "job-chk-1")
	assert.NoError(t, err)
	assert.Empty(t, afterDelete)
}

func TestBufferPoolManager(t *testing.T) {
	mgr := NewBufferPoolManager(1024)
	buf := mgr.Get()
	assert.Equal(t, 1024, len(*buf))
	mgr.Put(buf)

	tiered := NewTieredBufferPool()
	b1 := tiered.Get(2048)
	assert.Equal(t, 2048, len(*b1))
	tiered.Put(2048, b1)
}

func TestEventPublisher(t *testing.T) {
	pub := NewInMemoryEventPublisher()
	ctx := context.Background()

	ch, cancel := pub.Subscribe(ctx, "job-sub")
	defer cancel()

	evt := ports.TransferEvent{
		JobID:  "job-sub",
		Status: domain.JobStatusRunning,
		Metrics: domain.TransferMetrics{
			BytesTransferred: 50,
			TotalBytes:       100,
		},
	}

	pub.Publish(evt)

	select {
	case received := <-ch:
		assert.Equal(t, "job-sub", received.JobID)
		assert.Equal(t, domain.JobStatusRunning, received.Status)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}
