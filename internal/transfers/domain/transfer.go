package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidJobStateTransition = errors.New("invalid job state transition")
	ErrEmptyJobID                = errors.New("job ID cannot be empty")
	ErrZeroTotalBytes            = errors.New("total bytes cannot be negative")
)

type TransferType string

const (
	TransferTypeUpload   TransferType = "Upload"
	TransferTypeDownload TransferType = "Download"
	TransferTypeCopy     TransferType = "Copy"
)

type JobStatus string

const (
	JobStatusPending   JobStatus = "Pending"
	JobStatusRunning   JobStatus = "Running"
	JobStatusPausing   JobStatus = "Pausing"
	JobStatusPaused    JobStatus = "Paused"
	JobStatusCompleted JobStatus = "Completed"
	JobStatusFailed    JobStatus = "Failed"
	JobStatusCanceled  JobStatus = "Canceled"
)

type PartStatus string

const (
	PartStatusPending   PartStatus = "Pending"
	PartStatusRunning   PartStatus = "Running"
	PartStatusCompleted PartStatus = "Completed"
	PartStatusFailed    PartStatus = "Failed"
)

type TransferPart struct {
	JobID      string
	PartNumber int32
	Offset     int64
	Size       int64
	ETag       string
	Status     PartStatus
	UpdatedAt  time.Time
}

type TransferMetrics struct {
	BytesTransferred int64
	TotalBytes       int64
	SpeedBps         float64
	ETA              time.Duration
	Elapsed          time.Duration
}

type TransferJob struct {
	ID               string
	AccountID        string
	Type             TransferType
	SourcePath       string
	DestinationPath  string
	Bucket           string
	Key              string
	UploadID         string
	Status           JobStatus
	BytesTransferred int64
	TotalBytes       int64
	PartSize         int64
	ErrorMessage     string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (j *TransferJob) CanTransitionTo(target JobStatus) bool {
	switch j.Status {
	case JobStatusPending:
		return target == JobStatusRunning || target == JobStatusCanceled
	case JobStatusRunning:
		return target == JobStatusPausing || target == JobStatusPaused || target == JobStatusCompleted || target == JobStatusFailed || target == JobStatusCanceled
	case JobStatusPausing:
		return target == JobStatusPaused || target == JobStatusFailed || target == JobStatusCanceled
	case JobStatusPaused:
		return target == JobStatusRunning || target == JobStatusCanceled
	case JobStatusCompleted, JobStatusFailed, JobStatusCanceled:
		return false
	default:
		return false
	}
}

func (j *TransferJob) TransitionTo(target JobStatus) error {
	if !j.CanTransitionTo(target) {
		return ErrInvalidJobStateTransition
	}
	j.Status = target
	j.UpdatedAt = time.Now()
	return nil
}

func CalculateAdaptivePartSize(totalSize int64) int64 {
	const (
		minPartSize  int64 = 5 * 1024 * 1024
		maxPartSize  int64 = 5 * 1024 * 1024 * 1024
		maxParts     int64 = 10000
		basePartSize int64 = 16 * 1024 * 1024
	)

	if totalSize <= minPartSize {
		return minPartSize
	}

	requiredPartSize := (totalSize + maxParts - 1) / maxParts
	if requiredPartSize < basePartSize {
		if totalSize <= 100*1024*1024 {
			return 8 * 1024 * 1024
		}
		if totalSize <= 5*1024*1024*1024 {
			return 16 * 1024 * 1024
		}
		if totalSize <= 50*1024*1024*1024 {
			return 32 * 1024 * 1024
		}
		return 64 * 1024 * 1024
	}

	aligned := ((requiredPartSize + minPartSize - 1) / minPartSize) * minPartSize
	if aligned > maxPartSize {
		return maxPartSize
	}
	return aligned
}
