package application

import (
	"context"
	"errors"
	"math/rand/v2"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sekai-labs/kumokura/internal/transfers/adapters"
	"github.com/sekai-labs/kumokura/internal/transfers/domain"
	"github.com/sekai-labs/kumokura/internal/transfers/ports"
)

var (
	ErrJobNotFound = errors.New("transfer job not found")
)

type activeJobContext struct {
	cancel context.CancelFunc
}

type TransferService struct {
	repo       ports.TransferRepository
	publisher  ports.TransferEventPublisher
	worker     *adapters.S3TransferWorker
	bufferPool *adapters.TieredBufferPool

	maxConcurrency int
	mu             sync.Mutex
	activeJobs     map[string]*activeJobContext
	metrics        map[string]*domain.TransferMetrics
	metricsMu      sync.RWMutex
}

func NewTransferService(
	repo ports.TransferRepository,
	publisher ports.TransferEventPublisher,
	worker *adapters.S3TransferWorker,
	bufferPool *adapters.TieredBufferPool,
	maxConcurrency int,
) *TransferService {
	if maxConcurrency <= 0 {
		maxConcurrency = 100
	}
	return &TransferService{
		repo:           repo,
		publisher:      publisher,
		worker:         worker,
		bufferPool:     bufferPool,
		maxConcurrency: maxConcurrency,
		activeJobs:     make(map[string]*activeJobContext),
		metrics:        make(map[string]*domain.TransferMetrics),
	}
}

func (s *TransferService) SetMaxConcurrency(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n > 0 {
		s.maxConcurrency = n
	}
}

func (s *TransferService) GetMaxConcurrency() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.maxConcurrency
}

func (s *TransferService) SubmitJob(ctx context.Context, job domain.TransferJob) (string, error) {
	if job.Status == "" {
		job.Status = domain.JobStatusPending
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	job.UpdatedAt = time.Now()

	if err := s.repo.SaveJob(ctx, job); err != nil {
		return "", err
	}

	go s.executeJob(job.ID)
	return job.ID, nil
}

func (s *TransferService) PauseJob(ctx context.Context, jobID string) error {
	s.mu.Lock()
	act, exists := s.activeJobs[jobID]
	if exists && act.cancel != nil {
		act.cancel()
		delete(s.activeJobs, jobID)
	}
	s.mu.Unlock()

	return s.repo.UpdateJobStatus(ctx, jobID, domain.JobStatusPaused, "")
}

func (s *TransferService) ResumeJob(ctx context.Context, jobID string) error {
	job, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	if job.Status != domain.JobStatusPaused && job.Status != domain.JobStatusFailed {
		return domain.ErrInvalidJobStateTransition
	}

	if err := s.repo.UpdateJobStatus(ctx, jobID, domain.JobStatusPending, ""); err != nil {
		return err
	}

	go s.executeJob(jobID)
	return nil
}

func (s *TransferService) CancelJob(ctx context.Context, jobID string) error {
	s.mu.Lock()
	act, exists := s.activeJobs[jobID]
	if exists && act.cancel != nil {
		act.cancel()
		delete(s.activeJobs, jobID)
	}
	s.mu.Unlock()

	return s.repo.UpdateJobStatus(ctx, jobID, domain.JobStatusCanceled, "")
}

func (s *TransferService) GetJobMetrics(ctx context.Context, jobID string) (domain.TransferMetrics, error) {
	s.metricsMu.RLock()
	defer s.metricsMu.RUnlock()

	m, ok := s.metrics[jobID]
	if !ok {
		job, err := s.repo.GetJob(ctx, jobID)
		if err != nil {
			return domain.TransferMetrics{}, err
		}
		return domain.TransferMetrics{
			BytesTransferred: job.BytesTransferred,
			TotalBytes:       job.TotalBytes,
		}, nil
	}
	return *m, nil
}

func (s *TransferService) executeJob(jobID string) {
	ctx, cancel := context.WithCancel(context.Background())

	s.mu.Lock()
	s.activeJobs[jobID] = &activeJobContext{cancel: cancel}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.activeJobs, jobID)
		s.mu.Unlock()
	}()

	job, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		s.publishError(jobID, err.Error())
		return
	}

	currentJob, err := s.repo.GetJob(ctx, jobID)
	if err == nil && (currentJob.Status == domain.JobStatusPaused || currentJob.Status == domain.JobStatusCanceled) {
		return
	}
	if err := s.repo.UpdateJobStatus(ctx, jobID, domain.JobStatusRunning, ""); err != nil {
		s.publishError(jobID, err.Error())
		return
	}

	s.publisher.Publish(ports.TransferEvent{
		JobID:  jobID,
		Status: domain.JobStatusRunning,
		Metrics: domain.TransferMetrics{
			BytesTransferred: job.BytesTransferred,
			TotalBytes:       job.TotalBytes,
		},
	})

	if job.Bucket == "" || job.Key == "" {
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
	}

	switch job.Type {
	case domain.TransferTypeUpload:
		err = s.runMultipartUpload(ctx, job)
	case domain.TransferTypeDownload:
		err = s.runMultipartDownload(ctx, job)
	default:
		err = errors.New("unsupported transfer type")
	}

	if err != nil {
		currentJob, fetchErr := s.repo.GetJob(context.Background(), jobID)
		if fetchErr == nil && currentJob.Status == domain.JobStatusPaused {
			return
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			_ = s.repo.UpdateJobStatus(context.Background(), jobID, domain.JobStatusCanceled, "transfer canceled")
			s.publisher.Publish(ports.TransferEvent{
				JobID:  jobID,
				Status: domain.JobStatusCanceled,
			})
			return
		}
		_ = s.repo.UpdateJobStatus(context.Background(), jobID, domain.JobStatusFailed, err.Error())
		s.publishError(jobID, err.Error())
		return
	}

	_ = s.repo.UpdateJobStatus(context.Background(), jobID, domain.JobStatusCompleted, "")
	_ = s.repo.DeleteCheckpoints(context.Background(), jobID)
	s.publisher.Publish(ports.TransferEvent{
		JobID:  jobID,
		Status: domain.JobStatusCompleted,
		Metrics: domain.TransferMetrics{
			BytesTransferred: job.TotalBytes,
			TotalBytes:       job.TotalBytes,
		},
	})
}

func (s *TransferService) runMultipartUpload(ctx context.Context, job domain.TransferJob) error {
	file, err := os.Open(job.SourcePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	totalSize := info.Size()
	partSize := job.PartSize
	if partSize <= 0 {
		partSize = domain.CalculateAdaptivePartSize(totalSize)
	}

	checkpoints, err := s.repo.GetCheckpoints(ctx, job.ID)
	if err != nil {
		return err
	}

	var uploadID = job.UploadID
	if len(checkpoints) == 0 {
		if s.worker != nil {
			uID, err := s.worker.InitiateMultipartUpload(ctx, job.Bucket, job.Key)
			if err != nil {
				return err
			}
			uploadID = uID
		}

		var parts []domain.TransferPart
		var offset int64
		var partNum int32 = 1

		for offset < totalSize {
			currentPartSize := partSize
			if offset+currentPartSize > totalSize {
				currentPartSize = totalSize - offset
			}
			parts = append(parts, domain.TransferPart{
				JobID:      job.ID,
				PartNumber: partNum,
				Offset:     offset,
				Size:       currentPartSize,
				Status:     domain.PartStatusPending,
				UpdatedAt:  time.Now(),
			})
			offset += currentPartSize
			partNum++
		}

		if err := s.repo.SaveCheckpoints(ctx, parts); err != nil {
			return err
		}
		checkpoints = parts
	}

	var transferred int64
	for _, p := range checkpoints {
		if p.Status == domain.PartStatusCompleted {
			transferred += p.Size
		}
	}

	sem := make(chan struct{}, s.maxConcurrency)
	var wg sync.WaitGroup
	var uploadErr error
	var errMu sync.Mutex

	startTime := time.Now()

	for _, p := range checkpoints {
		if p.Status == domain.PartStatusCompleted {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(part domain.TransferPart) {
			defer func() {
				<-sem
				wg.Done()
			}()

			var etag string
			var partErr error

			for attempt := range 3 {
				if s.worker != nil {
					etag, partErr = s.worker.UploadPartChunk(ctx, job.Bucket, job.Key, uploadID, part, file)
				}
				if partErr == nil {
					break
				}
				jitter := time.Duration(rand.IntN(100)) * time.Millisecond
				backoff := (time.Duration(1<<attempt) * 100 * time.Millisecond) + jitter
				time.Sleep(backoff)
			}

			if partErr != nil {
				errMu.Lock()
				if uploadErr == nil {
					uploadErr = partErr
				}
				errMu.Unlock()
				return
			}

			part.Status = domain.PartStatusCompleted
			part.ETag = etag
			_ = s.repo.UpdateCheckpoint(context.Background(), part)

			curr := atomic.AddInt64(&transferred, part.Size)
			_ = s.repo.UpdateJobProgress(context.Background(), job.ID, curr)

			elapsed := time.Since(startTime)
			speed := float64(curr) / elapsed.Seconds()
			var eta time.Duration
			if speed > 0 {
				rem := totalSize - curr
				eta = time.Duration(float64(rem)/speed) * time.Second
			}

			metrics := domain.TransferMetrics{
				BytesTransferred: curr,
				TotalBytes:       totalSize,
				SpeedBps:         speed,
				ETA:              eta,
				Elapsed:          elapsed,
			}

			s.metricsMu.Lock()
			s.metrics[job.ID] = &metrics
			s.metricsMu.Unlock()

			s.publisher.Publish(ports.TransferEvent{
				JobID:   job.ID,
				Status:  domain.JobStatusRunning,
				Metrics: metrics,
			})
		}(p)
	}

	wg.Wait()

	if uploadErr != nil {
		return uploadErr
	}

	allParts, err := s.repo.GetCheckpoints(ctx, job.ID)
	if err != nil {
		return err
	}

	if s.worker != nil {
		if err := s.worker.CompleteMultipartUpload(ctx, job.Bucket, job.Key, uploadID, allParts); err != nil {
			return err
		}
	}

	return nil
}

func (s *TransferService) runMultipartDownload(ctx context.Context, job domain.TransferJob) error {
	destFile, err := os.OpenFile(job.DestinationPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer destFile.Close()

	totalSize := job.TotalBytes
	partSize := job.PartSize
	if partSize <= 0 {
		partSize = domain.CalculateAdaptivePartSize(totalSize)
	}

	checkpoints, err := s.repo.GetCheckpoints(ctx, job.ID)
	if err != nil {
		return err
	}

	if len(checkpoints) == 0 {
		var parts []domain.TransferPart
		var offset int64
		var partNum int32 = 1

		for offset < totalSize {
			currentPartSize := partSize
			if offset+currentPartSize > totalSize {
				currentPartSize = totalSize - offset
			}
			parts = append(parts, domain.TransferPart{
				JobID:      job.ID,
				PartNumber: partNum,
				Offset:     offset,
				Size:       currentPartSize,
				Status:     domain.PartStatusPending,
				UpdatedAt:  time.Now(),
			})
			offset += currentPartSize
			partNum++
		}

		if err := s.repo.SaveCheckpoints(ctx, parts); err != nil {
			return err
		}
		checkpoints = parts
	}

	sem := make(chan struct{}, s.maxConcurrency)
	var wg sync.WaitGroup
	var downloadErr error
	var errMu sync.Mutex

	for _, p := range checkpoints {
		if p.Status == domain.PartStatusCompleted {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(part domain.TransferPart) {
			defer func() {
				<-sem
				wg.Done()
			}()

			var partErr error
			if s.worker != nil {
				partErr = s.worker.DownloadRangeChunk(ctx, job.Bucket, job.Key, part, destFile)
			}

			if partErr != nil {
				errMu.Lock()
				if downloadErr == nil {
					downloadErr = partErr
				}
				errMu.Unlock()
				return
			}

			part.Status = domain.PartStatusCompleted
			_ = s.repo.UpdateCheckpoint(context.Background(), part)
		}(p)
	}

	wg.Wait()
	return downloadErr
}

func (s *TransferService) publishError(jobID string, msg string) {
	s.publisher.Publish(ports.TransferEvent{
		JobID:  jobID,
		Status: domain.JobStatusFailed,
		Error:  msg,
	})
}

func (s *TransferService) ListJobs(ctx context.Context, accountID string, status domain.JobStatus) ([]domain.TransferJob, error) {
	if s.repo == nil {
		return []domain.TransferJob{}, nil
	}
	return s.repo.ListJobs(ctx, accountID, status)
}
