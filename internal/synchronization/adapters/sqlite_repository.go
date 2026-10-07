package adapters

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/sekai-labs/kumokura/internal/synchronization/domain"
	"github.com/sekai-labs/kumokura/internal/synchronization/ports"
)

type SQLiteSyncRepository struct {
	db *sql.DB
}

func NewSQLiteSyncRepository(db *sql.DB) *SQLiteSyncRepository {
	return &SQLiteSyncRepository{db: db}
}

var _ ports.SyncRepository = (*SQLiteSyncRepository)(nil)

func (r *SQLiteSyncRepository) SaveJob(ctx context.Context, job *ports.SyncJobRecord) error {
	query := `
INSERT INTO sync_jobs (
    id, account_id, source_bucket, source_prefix, destination_path,
    sync_direction, delete_extraneous, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	deleteExtraneous := 0
	if job.DeleteExtraneous {
		deleteExtraneous = 1
	}

	_, err := r.db.ExecContext(ctx, query,
		job.ID,
		job.AccountID,
		job.SourceBucket,
		job.SourcePrefix,
		job.DestinationPath,
		string(job.SyncDirection),
		deleteExtraneous,
		job.Status,
		job.CreatedAt.Format(time.RFC3339Nano),
		job.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert sync job: %w", err)
	}
	return nil
}

func (r *SQLiteSyncRepository) GetJob(ctx context.Context, id string) (*ports.SyncJobRecord, error) {
	query := `
SELECT id, account_id, source_bucket, source_prefix, destination_path,
       sync_direction, delete_extraneous, status, created_at, updated_at
FROM sync_jobs
WHERE id = ?
`
	var (
		jobID            string
		accountID        string
		sourceBucket     string
		sourcePrefix     string
		destinationPath  string
		syncDirection    string
		deleteExtraneous int
		status           string
		createdAtStr     string
		updatedAtStr     string
	)

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&jobID,
		&accountID,
		&sourceBucket,
		&sourcePrefix,
		&destinationPath,
		&syncDirection,
		&deleteExtraneous,
		&status,
		&createdAtStr,
		&updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("sync job not found: %s", id)
		}
		return nil, fmt.Errorf("query sync job %s: %w", id, err)
	}

	createdAt, _ := time.Parse(time.RFC3339Nano, createdAtStr)
	if createdAt.IsZero() {
		createdAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
	}
	updatedAt, _ := time.Parse(time.RFC3339Nano, updatedAtStr)
	if updatedAt.IsZero() {
		updatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAtStr)
	}

	return &ports.SyncJobRecord{
		ID:               jobID,
		AccountID:        accountID,
		SourceBucket:     sourceBucket,
		SourcePrefix:     sourcePrefix,
		DestinationPath:  destinationPath,
		SyncDirection:    domain.SyncDirection(syncDirection),
		DeleteExtraneous: deleteExtraneous == 1,
		Status:           status,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}, nil
}

func (r *SQLiteSyncRepository) ListJobs(ctx context.Context, accountID string) ([]*ports.SyncJobRecord, error) {
	query := `
SELECT id, account_id, source_bucket, source_prefix, destination_path,
       sync_direction, delete_extraneous, status, created_at, updated_at
FROM sync_jobs
WHERE account_id = ?
ORDER BY created_at DESC
`
	rows, err := r.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("list sync jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*ports.SyncJobRecord
	for rows.Next() {
		var (
			jobID            string
			accID            string
			sourceBucket     string
			sourcePrefix     string
			destinationPath  string
			syncDirection    string
			deleteExtraneous int
			status           string
			createdAtStr     string
			updatedAtStr     string
		)

		if err := rows.Scan(
			&jobID,
			&accID,
			&sourceBucket,
			&sourcePrefix,
			&destinationPath,
			&syncDirection,
			&deleteExtraneous,
			&status,
			&createdAtStr,
			&updatedAtStr,
		); err != nil {
			return nil, fmt.Errorf("scan sync job: %w", err)
		}

		createdAt, _ := time.Parse(time.RFC3339Nano, createdAtStr)
		if createdAt.IsZero() {
			createdAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		}
		updatedAt, _ := time.Parse(time.RFC3339Nano, updatedAtStr)
		if updatedAt.IsZero() {
			updatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAtStr)
		}

		jobs = append(jobs, &ports.SyncJobRecord{
			ID:               jobID,
			AccountID:        accID,
			SourceBucket:     sourceBucket,
			SourcePrefix:     sourcePrefix,
			DestinationPath:  destinationPath,
			SyncDirection:    domain.SyncDirection(syncDirection),
			DeleteExtraneous: deleteExtraneous == 1,
			Status:           status,
			CreatedAt:        createdAt,
			UpdatedAt:        updatedAt,
		})
	}

	return jobs, rows.Err()
}

func (r *SQLiteSyncRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE sync_jobs SET status = ?, updated_at = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, status, time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}

func (r *SQLiteSyncRepository) DeleteJob(ctx context.Context, id string) error {
	query := `DELETE FROM sync_jobs WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
