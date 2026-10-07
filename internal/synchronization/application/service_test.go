package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/sekai-labs/kumokura/internal/synchronization/application"
	"github.com/sekai-labs/kumokura/internal/synchronization/domain"
	"github.com/sekai-labs/kumokura/internal/synchronization/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockScanner struct {
	files map[string]*domain.FileEntry
}

func (m *mockScanner) Scan(ctx context.Context, target string, filter domain.Filter) (map[string]*domain.FileEntry, error) {
	filtered := make(map[string]*domain.FileEntry)
	for k, v := range m.files {
		if filter.Matches(k) {
			filtered[k] = v
		}
	}
	return filtered, nil
}

func TestSyncPlan_CreateUpdateDeleteSkip(t *testing.T) {
	now := time.Now().UTC()
	sourceFiles := map[string]*domain.FileEntry{
		"new.txt": {
			RelativePath: "new.txt",
			Size:         100,
			ModTime:      now,
		},
		"updated.txt": {
			RelativePath: "updated.txt",
			Size:         200,
			ModTime:      now.Add(1 * time.Hour),
		},
		"identical.txt": {
			RelativePath: "identical.txt",
			Size:         300,
			ModTime:      now,
		},
	}

	destFiles := map[string]*domain.FileEntry{
		"updated.txt": {
			RelativePath: "updated.txt",
			Size:         150,
			ModTime:      now,
		},
		"identical.txt": {
			RelativePath: "identical.txt",
			Size:         300,
			ModTime:      now,
		},
		"extraneous.txt": {
			RelativePath: "extraneous.txt",
			Size:         400,
			ModTime:      now,
		},
	}

	srcScanner := &mockScanner{files: sourceFiles}
	dstScanner := &mockScanner{files: destFiles}

	service := application.NewSyncService(nil)
	ctx := context.Background()

	optsWithoutDelete := ports.SyncOptions{
		Direction:        domain.DirectionLocalToS3,
		Mode:             domain.ModeUploadOnly,
		DeleteExtraneous: false,
		DryRun:           true,
	}

	plan, err := service.Plan(ctx, srcScanner, dstScanner, "/local", "s3://bucket", optsWithoutDelete)
	require.NoError(t, err)
	assert.Equal(t, 2, plan.TotalUploads)
	assert.Equal(t, 0, plan.TotalDeletes)
	assert.Equal(t, 2, plan.TotalSkips)
	assert.Equal(t, int64(300), plan.TotalBytes)

	optsWithDelete := ports.SyncOptions{
		Direction:        domain.DirectionLocalToS3,
		Mode:             domain.ModeUploadOnly,
		DeleteExtraneous: true,
		DryRun:           true,
	}

	planWithDelete, err := service.Plan(ctx, srcScanner, dstScanner, "/local", "s3://bucket", optsWithDelete)
	require.NoError(t, err)
	assert.Equal(t, 1, planWithDelete.TotalDeletes)
}

func TestSyncPlan_ConflictPolicyFail(t *testing.T) {
	now := time.Now().UTC()
	sourceFiles := map[string]*domain.FileEntry{
		"conflict.txt": {
			RelativePath: "conflict.txt",
			Size:         100,
			ModTime:      now,
		},
	}
	destFiles := map[string]*domain.FileEntry{
		"conflict.txt": {
			RelativePath: "conflict.txt",
			Size:         200,
			ModTime:      now,
		},
	}

	service := application.NewSyncService(nil)
	ctx := context.Background()

	opts := ports.SyncOptions{
		ConflictPolicy: domain.ConflictPolicyFail,
		Strategy:       domain.StrategySizeOnly,
	}

	plan, err := service.Plan(ctx, &mockScanner{files: sourceFiles}, &mockScanner{files: destFiles}, "src", "dst", opts)
	require.NoError(t, err)
	assert.Equal(t, 1, plan.TotalConflicts)
	assert.Equal(t, domain.SyncActionConflict, plan.Items[0].Action)
}

func TestSyncPlan_Bidirectional(t *testing.T) {
	now := time.Now().UTC()
	sourceFiles := map[string]*domain.FileEntry{
		"local_only.txt": {
			RelativePath: "local_only.txt",
			Size:         100,
			ModTime:      now,
		},
	}
	destFiles := map[string]*domain.FileEntry{
		"remote_only.txt": {
			RelativePath: "remote_only.txt",
			Size:         200,
			ModTime:      now,
		},
	}

	service := application.NewSyncService(nil)
	ctx := context.Background()

	opts := ports.SyncOptions{
		Mode: domain.ModeBidirectional,
	}

	plan, err := service.Plan(ctx, &mockScanner{files: sourceFiles}, &mockScanner{files: destFiles}, "src", "dst", opts)
	require.NoError(t, err)
	assert.Equal(t, 1, plan.TotalUploads)
	assert.Equal(t, 1, plan.TotalDownloads)
	assert.Equal(t, int64(300), plan.TotalBytes)
}
