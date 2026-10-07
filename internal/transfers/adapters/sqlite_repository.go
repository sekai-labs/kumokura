package adapters

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/sekai-labs/kumokura/internal/platform/database"
	"github.com/sekai-labs/kumokura/internal/transfers/domain"
	"github.com/sekai-labs/kumokura/internal/transfers/ports"
)

type SQLiteTransferRepository struct {
	db *database.DB
}

func NewSQLiteTransferRepository(db *database.DB) ports.TransferRepository {
	return &SQLiteTransferRepository{
		db: db,
	}
}

func (r *SQLiteTransferRepository) SaveJob(ctx context.Context, job domain.TransferJob) error {
	query := `
INSERT INTO transfer_jobs (
    id, account_id, direction, source_path, destination_path,
    status, bytes_transferred, total_bytes, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    status = excluded.status,
    bytes_transferred = excluded.bytes_transferred,
    total_bytes = excluded.total_bytes,
    updated_at = excluded.updated_at;
`
	_, err := r.db.ExecContext(ctx, query,
		job.ID,
		job.AccountID,
		string(job.Type),
		job.SourcePath,
		job.DestinationPath,
		string(job.Status),
		job.BytesTransferred,
		job.TotalBytes,
		job.CreatedAt,
		job.UpdatedAt,
	)
	return err
}

func (r *SQLiteTransferRepository) GetJob(ctx context.Context, jobID string) (domain.TransferJob, error) {
	query := `
SELECT id, account_id, direction, source_path, destination_path, status, bytes_transferred, total_bytes, created_at, updated_at
FROM transfer_jobs WHERE id = ?
`
	var job domain.TransferJob
	var direction string
	var status string

	err := r.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID,
		&job.AccountID,
		&direction,
		&job.SourcePath,
		&job.DestinationPath,
		&status,
		&job.BytesTransferred,
		&job.TotalBytes,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.TransferJob{}, errors.New("job not found")
		}
		return domain.TransferJob{}, err
	}
	job.Type = domain.TransferType(direction)
	job.Status = domain.JobStatus(status)

	dest := job.DestinationPath
	if strings.HasPrefix(dest, "s3://") {
		trimmed := strings.TrimPrefix(dest, "s3://")
		parts := strings.SplitN(trimmed, "/", 2)
		if len(parts) > 0 {
			job.Bucket = parts[0]
		}
		if len(parts) > 1 {
			job.Key = parts[1]
		}
	}
	return job, nil
}
func (r *SQLiteTransferRepository) ListJobs(ctx context.Context, accountID string, status domain.JobStatus) ([]domain.TransferJob, error) {
	query := `
SELECT id, account_id, direction, source_path, destination_path, status, bytes_transferred, total_bytes, created_at, updated_at
FROM transfer_jobs
WHERE (account_id = ? OR ? = '')
  AND (status = ? OR ? = '')
ORDER BY created_at DESC
`
	rows, err := r.db.QueryContext(ctx, query, accountID, accountID, string(status), string(status))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []domain.TransferJob
	for rows.Next() {
		var job domain.TransferJob
		var direction string
		var stat string
		if err := rows.Scan(
			&job.ID,
			&job.AccountID,
			&direction,
			&job.SourcePath,
			&job.DestinationPath,
			&stat,
			&job.BytesTransferred,
			&job.TotalBytes,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, err
		}
		job.Type = domain.TransferType(direction)
		job.Status = domain.JobStatus(stat)
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *SQLiteTransferRepository) UpdateJobStatus(ctx context.Context, jobID string, status domain.JobStatus, errMsg string) error {
	query := `
UPDATE transfer_jobs
SET status = ?, updated_at = ?
WHERE id = ?
`
	_, err := r.db.ExecContext(ctx, query, string(status), time.Now(), jobID)
	return err
}

func (r *SQLiteTransferRepository) UpdateJobProgress(ctx context.Context, jobID string, bytesTransferred int64) error {
	query := `
UPDATE transfer_jobs
SET bytes_transferred = ?, updated_at = ?
WHERE id = ?
`
	_, err := r.db.ExecContext(ctx, query, bytesTransferred, time.Now(), jobID)
	return err
}

func (r *SQLiteTransferRepository) SaveCheckpoints(ctx context.Context, checkpoints []domain.TransferPart) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
INSERT INTO transfer_checkpoints (job_id, part_number, byte_offset, size, etag, status, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(job_id, part_number) DO UPDATE SET
    etag = excluded.etag,
    status = excluded.status,
    updated_at = excluded.updated_at;
`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, cp := range checkpoints {
		_, err := stmt.ExecContext(ctx,
			cp.JobID,
			cp.PartNumber,
			cp.Offset,
			cp.Size,
			cp.ETag,
			string(cp.Status),
			cp.UpdatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLiteTransferRepository) UpdateCheckpoint(ctx context.Context, cp domain.TransferPart) error {
	query := `
UPDATE transfer_checkpoints
SET etag = ?, status = ?, updated_at = ?
WHERE job_id = ? AND part_number = ?
`
	_, err := r.db.ExecContext(ctx, query, cp.ETag, string(cp.Status), time.Now(), cp.JobID, cp.PartNumber)
	return err
}

func (r *SQLiteTransferRepository) GetCheckpoints(ctx context.Context, jobID string) ([]domain.TransferPart, error) {
	query := `
SELECT job_id, part_number, byte_offset, size, etag, status, updated_at
FROM transfer_checkpoints
WHERE job_id = ?
ORDER BY part_number ASC
`
	rows, err := r.db.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parts []domain.TransferPart
	for rows.Next() {
		var p domain.TransferPart
		var status string
		if err := rows.Scan(
			&p.JobID,
			&p.PartNumber,
			&p.Offset,
			&p.Size,
			&p.ETag,
			&status,
			&p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		p.Status = domain.PartStatus(status)
		parts = append(parts, p)
	}
	return parts, rows.Err()
}

func (r *SQLiteTransferRepository) DeleteCheckpoints(ctx context.Context, jobID string) error {
	query := `DELETE FROM transfer_checkpoints WHERE job_id = ?`
	_, err := r.db.ExecContext(ctx, query, jobID)
	return err
}
