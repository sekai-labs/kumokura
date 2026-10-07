package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTransferJob_Transitions(t *testing.T) {
	job := TransferJob{
		ID:        "job-1",
		Status:    JobStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	assert.True(t, job.CanTransitionTo(JobStatusRunning))
	assert.True(t, job.CanTransitionTo(JobStatusCanceled))
	assert.False(t, job.CanTransitionTo(JobStatusCompleted))

	err := job.TransitionTo(JobStatusRunning)
	assert.NoError(t, err)
	assert.Equal(t, JobStatusRunning, job.Status)

	assert.True(t, job.CanTransitionTo(JobStatusPausing))
	assert.True(t, job.CanTransitionTo(JobStatusPaused))
	assert.True(t, job.CanTransitionTo(JobStatusCompleted))
	assert.True(t, job.CanTransitionTo(JobStatusFailed))

	err = job.TransitionTo(JobStatusPaused)
	assert.NoError(t, err)
	assert.Equal(t, JobStatusPaused, job.Status)

	err = job.TransitionTo(JobStatusRunning)
	assert.NoError(t, err)

	err = job.TransitionTo(JobStatusCompleted)
	assert.NoError(t, err)

	err = job.TransitionTo(JobStatusRunning)
	assert.ErrorIs(t, err, ErrInvalidJobStateTransition)
}

func TestCalculateAdaptivePartSize(t *testing.T) {
	tests := []struct {
		name      string
		totalSize int64
		minSize   int64
		maxSize   int64
	}{
		{
			name:      "tiny file 1MB",
			totalSize: 1 * 1024 * 1024,
			minSize:   5 * 1024 * 1024,
			maxSize:   5 * 1024 * 1024,
		},
		{
			name:      "medium file 50MB",
			totalSize: 50 * 1024 * 1024,
			minSize:   8 * 1024 * 1024,
			maxSize:   8 * 1024 * 1024,
		},
		{
			name:      "standard file 2GB",
			totalSize: 2 * 1024 * 1024 * 1024,
			minSize:   16 * 1024 * 1024,
			maxSize:   16 * 1024 * 1024,
		},
		{
			name:      "large file 20GB",
			totalSize: 20 * 1024 * 1024 * 1024,
			minSize:   32 * 1024 * 1024,
			maxSize:   32 * 1024 * 1024,
		},
		{
			name:      "huge file 100GB",
			totalSize: 100 * 1024 * 1024 * 1024,
			minSize:   10 * 1024 * 1024,
			maxSize:   64 * 1024 * 1024,
		},
		{
			name:      "massive file 1TB",
			totalSize: 1024 * 1024 * 1024 * 1024,
			minSize:   100 * 1024 * 1024,
			maxSize:   200 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			partSize := CalculateAdaptivePartSize(tt.totalSize)
			assert.GreaterOrEqual(t, partSize, tt.minSize)
			assert.LessOrEqual(t, partSize, tt.maxSize)
			numParts := (tt.totalSize + partSize - 1) / partSize
			assert.LessOrEqual(t, numParts, int64(10000))
		})
	}
}
