package ports

import (
	"context"

	"github.com/sekai-labs/kumokura/internal/transfers/domain"
)

type TransferRepository interface {
	SaveJob(ctx context.Context, job domain.TransferJob) error
	GetJob(ctx context.Context, jobID string) (domain.TransferJob, error)
	ListJobs(ctx context.Context, accountID string, status domain.JobStatus) ([]domain.TransferJob, error)
	UpdateJobStatus(ctx context.Context, jobID string, status domain.JobStatus, errMsg string) error
	UpdateJobProgress(ctx context.Context, jobID string, bytesTransferred int64) error
	SaveCheckpoints(ctx context.Context, checkpoints []domain.TransferPart) error
	UpdateCheckpoint(ctx context.Context, checkpoint domain.TransferPart) error
	GetCheckpoints(ctx context.Context, jobID string) ([]domain.TransferPart, error)
	DeleteCheckpoints(ctx context.Context, jobID string) error
}

type TransferEvent struct {
	JobID   string
	Status  domain.JobStatus
	Metrics domain.TransferMetrics
	Error   string
}

type TransferEventPublisher interface {
	Publish(event TransferEvent)
	Subscribe(ctx context.Context, jobID string) (<-chan TransferEvent, func())
}

type TransferCoordinator interface {
	SubmitJob(ctx context.Context, job domain.TransferJob) (string, error)
	PauseJob(ctx context.Context, jobID string) error
	ResumeJob(ctx context.Context, jobID string) error
	CancelJob(ctx context.Context, jobID string) error
	GetJobMetrics(ctx context.Context, jobID string) (domain.TransferMetrics, error)
}
