package ports

import (
	"context"
	"time"

	"github.com/sekai-labs/kumokura/internal/synchronization/domain"
)

type SyncJobRecord struct {
	ID               string
	AccountID        string
	SourceBucket     string
	SourcePrefix     string
	DestinationPath  string
	SyncDirection    domain.SyncDirection
	DeleteExtraneous bool
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type SyncRepository interface {
	SaveJob(ctx context.Context, job *SyncJobRecord) error
	GetJob(ctx context.Context, id string) (*SyncJobRecord, error)
	ListJobs(ctx context.Context, accountID string) ([]*SyncJobRecord, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	DeleteJob(ctx context.Context, id string) error
}

type SyncScanner interface {
	Scan(ctx context.Context, target string, filter domain.Filter) (map[string]*domain.FileEntry, error)
}

type SyncOptions struct {
	Direction        domain.SyncDirection
	Mode             domain.SyncMode
	Strategy         domain.ComparisonStrategy
	ConflictPolicy   domain.ConflictPolicy
	DeleteExtraneous bool
	DryRun           bool
	TimeTolerance    time.Duration
	Filter           domain.Filter
	MaxConcurrency   int
}

type SyncEngine interface {
	Plan(ctx context.Context, sourceScanner, destScanner SyncScanner, sourceTarget, destTarget string, opts SyncOptions) (*domain.SyncPlan, error)
	Execute(ctx context.Context, plan *domain.SyncPlan, opts SyncOptions) error
	Cancel(ctx context.Context, jobID string) error
}
