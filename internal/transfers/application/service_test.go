package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sekai-labs/kumokura/internal/platform/database"
	"github.com/sekai-labs/kumokura/internal/transfers/adapters"
	"github.com/sekai-labs/kumokura/internal/transfers/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupServiceTestDB(t *testing.T) *database.DB {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "transfer_svc.db")
	db, err := database.Open(dbPath)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Migrate(ctx)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
INSERT INTO accounts (id, name, account_type) VALUES ('acc-svc', 'Transfer Account', 'aws')
`)
	require.NoError(t, err)
	return db
}

func TestTransferService_SubmitAndCancel(t *testing.T) {
	db := setupServiceTestDB(t)
	defer db.Close()

	repo := adapters.NewSQLiteTransferRepository(db)
	pub := adapters.NewInMemoryEventPublisher()
	tiered := adapters.NewTieredBufferPool()
	svc := NewTransferService(repo, pub, nil, tiered, 4)

	ctx := context.Background()

	tmpFile := filepath.Join(t.TempDir(), "upload.dat")
	data := make([]byte, 1024*1024)
	err := os.WriteFile(tmpFile, data, 0644)
	require.NoError(t, err)

	job := domain.TransferJob{
		ID:              "job-svc-1",
		AccountID:       "acc-svc",
		Type:            domain.TransferTypeUpload,
		SourcePath:      tmpFile,
		DestinationPath: "s3://b/upload.dat",
		Bucket:          "b",
		Key:             "upload.dat",
		TotalBytes:      int64(len(data)),
	}

	id, err := svc.SubmitJob(ctx, job)
	assert.NoError(t, err)
	assert.Equal(t, "job-svc-1", id)

	time.Sleep(50 * time.Millisecond)

	err = svc.CancelJob(ctx, id)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	cancelledJob, err := repo.GetJob(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, domain.JobStatusCanceled, cancelledJob.Status)
}

func TestTransferService_PauseAndResume(t *testing.T) {
	db := setupServiceTestDB(t)
	defer db.Close()

	repo := adapters.NewSQLiteTransferRepository(db)
	pub := adapters.NewInMemoryEventPublisher()
	tiered := adapters.NewTieredBufferPool()
	svc := NewTransferService(repo, pub, nil, tiered, 4)

	ctx := context.Background()

	job := domain.TransferJob{
		ID:              "job-svc-pause",
		AccountID:       "acc-svc",
		Type:            domain.TransferTypeUpload,
		SourcePath:      "/tmp/nonexistent.dat",
		DestinationPath: "s3://b/nonexistent.dat",
		Bucket:          "b",
		Key:             "nonexistent.dat",
		TotalBytes:      1000,
	}

	id, err := svc.SubmitJob(ctx, job)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	err = svc.PauseJob(ctx, id)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	pausedJob, err := repo.GetJob(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, domain.JobStatusPaused, pausedJob.Status)

	err = svc.ResumeJob(ctx, id)
	assert.NoError(t, err)
}

func TestTransferService_EventSubscription(t *testing.T) {
	db := setupServiceTestDB(t)
	defer db.Close()

	repo := adapters.NewSQLiteTransferRepository(db)
	pub := adapters.NewInMemoryEventPublisher()
	tiered := adapters.NewTieredBufferPool()
	svc := NewTransferService(repo, pub, nil, tiered, 4)

	ctx := context.Background()
	events, unsubscribe := pub.Subscribe(ctx, "job-event-test")
	defer unsubscribe()

	job := domain.TransferJob{
		ID:              "job-event-test",
		AccountID:       "acc-svc",
		Type:            domain.TransferTypeUpload,
		SourcePath:      "/tmp/file.dat",
		DestinationPath: "s3://b/file.dat",
		TotalBytes:      5000,
	}

	_, err := svc.SubmitJob(ctx, job)
	assert.NoError(t, err)

	select {
	case evt := <-events:
		assert.Equal(t, "job-event-test", evt.JobID)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for job event")
	}
}
