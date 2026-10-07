package performance

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sekai-labs/kumokura/internal/platform/database"
	transferAdapters "github.com/sekai-labs/kumokura/internal/transfers/adapters"
	transferApp "github.com/sekai-labs/kumokura/internal/transfers/application"
	transferDomain "github.com/sekai-labs/kumokura/internal/transfers/domain"
	transferPorts "github.com/sekai-labs/kumokura/internal/transfers/ports"
	"github.com/stretchr/testify/require"
)

type mockPerformanceS3Server struct {
	server    *httptest.Server
	mu        sync.Mutex
	uploadIDs map[string]bool
	objects   map[string][]byte
}

func newMockPerformanceS3Server() *mockPerformanceS3Server {
	m := &mockPerformanceS3Server{
		uploadIDs: make(map[string]bool),
		objects:   make(map[string][]byte),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()

		if r.Method == http.MethodPost && r.URL.Query().Has("uploads") {
			uploadID := fmt.Sprintf("perf-up-%d", time.Now().UnixNano())
			m.uploadIDs[uploadID] = true
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `<InitiateMultipartUploadResult><UploadId>%s</UploadId></InitiateMultipartUploadResult>`, uploadID)
			return
		}

		if r.Method == http.MethodPut && r.URL.Query().Has("uploadId") && r.URL.Query().Has("partNumber") {
			_, _ = io.Copy(io.Discard, r.Body)
			w.Header().Set("ETag", `"etag-perf-part"`)
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodPost && r.URL.Query().Has("uploadId") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `<CompleteMultipartUploadResult><ETag>"etag-perf-complete"</ETag></CompleteMultipartUploadResult>`)
			return
		}

		if r.Method == http.MethodDelete && r.URL.Query().Has("uploadId") {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method == http.MethodPut {
			data, _ := io.ReadAll(r.Body)
			m.objects[r.URL.Path] = data
			w.Header().Set("ETag", `"etag-perf-single"`)
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodGet {
			data := m.objects[r.URL.Path]
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	m.server = httptest.NewServer(handler)
	return m
}

func (m *mockPerformanceS3Server) Close() {
	m.server.Close()
}

func (m *mockPerformanceS3Server) Client() *s3.Client {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               m.server.URL,
			HostnameImmutable: true,
			SigningRegion:     "us-east-1",
		}, nil
	})

	cfg := aws.Config{
		Region: "us-east-1",
		Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     "mock-ak",
				SecretAccessKey: "mock-sk",
			}, nil
		}),
		EndpointResolverWithOptions: customResolver,
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
}

func setupTestTransferService(t *testing.T, s3Client *s3.Client, bufferPool *transferAdapters.TieredBufferPool, concurrency int) (*transferApp.TransferService, transferPorts.TransferRepository, transferPorts.TransferEventPublisher) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "perf_transfer.db")
	db, err := database.Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	ctx := context.Background()
	err = db.Migrate(ctx)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO accounts (id, name, account_type) VALUES ('acc-perf', 'Perf Acc', 'aws')`)
	require.NoError(t, err)

	repo := transferAdapters.NewSQLiteTransferRepository(db)
	pub := transferAdapters.NewInMemoryEventPublisher()
	worker := transferAdapters.NewS3TransferWorker(s3Client, bufferPool)
	svc := transferApp.NewTransferService(repo, pub, worker, bufferPool, concurrency)

	return svc, repo, pub
}

func TestAdaptivePartSizeCalculationScaling(t *testing.T) {
	sizes := []int64{
		1024 * 1024,
		10 * 1024 * 1024,
		50 * 1024 * 1024,
		100 * 1024 * 1024,
		1024 * 1024 * 1024,
		10 * 1024 * 1024 * 1024,
		50 * 1024 * 1024 * 1024,
		500 * 1024 * 1024 * 1024,
		5 * 1024 * 1024 * 1024 * 1024,
	}

	const iterations = 500000

	for _, size := range sizes {
		t.Run(fmt.Sprintf("Size_%dMB", size/(1024*1024)), func(t *testing.T) {
			var totalAllocBytes uint64
			var m1, m2 runtime.MemStats

			runtime.GC()
			runtime.ReadMemStats(&m1)
			start := time.Now()

			var dummy int64
			for range iterations {
				dummy += transferDomain.CalculateAdaptivePartSize(size)
			}

			elapsed := time.Since(start)
			runtime.ReadMemStats(&m2)
			totalAllocBytes = m2.TotalAlloc - m1.TotalAlloc

			require.NotZero(t, dummy)
			require.Less(t, totalAllocBytes, uint64(1024*1024))

			opsPerSec := float64(iterations) / elapsed.Seconds()
			require.Greater(t, opsPerSec, float64(100000))
		})
	}
}

func TestTieredBufferPoolRecyclingUnderMemoryPressure(t *testing.T) {
	tiered := transferAdapters.NewTieredBufferPool()
	sizes := []int{
		5 * 1024 * 1024,
		8 * 1024 * 1024,
		16 * 1024 * 1024,
		32 * 1024 * 1024,
	}

	const warmupRounds = 50
	for _, sz := range sizes {
		for range warmupRounds {
			buf := tiered.Get(sz)
			tiered.Put(sz, buf)
		}
	}

	const goroutines = 32
	const operationsPerGoroutine = 500

	runtime.GC()
	var mBefore, mAfter runtime.MemStats
	runtime.ReadMemStats(&mBefore)

	var wg sync.WaitGroup
	var activeOps atomic.Int64

	startSignal := make(chan struct{})

	for g := range goroutines {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			<-startSignal

			for i := range operationsPerGoroutine {
				sz := sizes[(routineID+i)%len(sizes)]
				bufPtr := tiered.Get(sz)
				if bufPtr == nil || len(*bufPtr) != sz {
					return
				}

				(*bufPtr)[0] = byte(routineID)
				(*bufPtr)[sz-1] = byte(i)

				activeOps.Add(1)
				tiered.Put(sz, bufPtr)
			}
		}(g)
	}

	start := time.Now()
	close(startSignal)
	wg.Wait()
	duration := time.Since(start)

	runtime.ReadMemStats(&mAfter)

	totalOperations := int64(goroutines * operationsPerGoroutine)
	require.Equal(t, totalOperations, activeOps.Load())

	throughputOps := float64(totalOperations) / duration.Seconds()
	require.Greater(t, throughputOps, float64(1000))

	for _, sz := range sizes {
		allocs := testing.AllocsPerRun(100, func() {
			b := tiered.Get(sz)
			tiered.Put(sz, b)
		})
		require.Equal(t, float64(0), allocs)
	}
}

func TestSmallObjectThroughputSimulation(t *testing.T) {
	mockServer := newMockPerformanceS3Server()
	defer mockServer.Close()

	s3Client := mockServer.Client()
	tieredPool := transferAdapters.NewTieredBufferPool()
	concurrency := 32
	svc, repo, _ := setupTestTransferService(t, s3Client, tieredPool, concurrency)

	tmpDir := t.TempDir()
	const numJobs = 200
	const fileSize = 128 * 1024

	dataBuf := make([]byte, fileSize)
	_, err := rand.Read(dataBuf)
	require.NoError(t, err)

	sharedSourceFile := filepath.Join(tmpDir, "shared_small.dat")
	err = os.WriteFile(sharedSourceFile, dataBuf, 0644)
	require.NoError(t, err)

	ctx := context.Background()
	start := time.Now()

	var wg sync.WaitGroup
	errChan := make(chan error, numJobs)

	for i := range numJobs {
		wg.Add(1)
		go func(jobIndex int) {
			defer wg.Done()

			jobID := fmt.Sprintf("perf-small-job-%d", jobIndex)
			job := transferDomain.TransferJob{
				ID:              jobID,
				AccountID:       "acc-perf",
				Type:            transferDomain.TransferTypeUpload,
				SourcePath:      sharedSourceFile,
				DestinationPath: fmt.Sprintf("s3://test-perf-bucket/small_%d.dat", jobIndex),
				Bucket:          "test-perf-bucket",
				Key:             fmt.Sprintf("small_%d.dat", jobIndex),
				TotalBytes:      int64(fileSize),
				PartSize:        int64(fileSize),
			}

			_, submitErr := svc.SubmitJob(ctx, job)
			if submitErr != nil {
				errChan <- submitErr
				return
			}

			deadline := time.After(30 * time.Second)
			ticker := time.NewTicker(5 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-deadline:
					errChan <- fmt.Errorf("timeout waiting for job %s", jobID)
					return
				case <-ticker.C:
					j, getErr := repo.GetJob(ctx, jobID)
					if getErr == nil {
						if j.Status == transferDomain.JobStatusCompleted {
							return
						}
						if j.Status == transferDomain.JobStatusFailed {
							errChan <- fmt.Errorf("job %s failed: %s", jobID, j.ErrorMessage)
							return
						}
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		require.NoError(t, err)
	}

	elapsed := time.Since(start)
	totalBytes := int64(numJobs * fileSize)
	throughputMBps := (float64(totalBytes) / (1024 * 1024)) / elapsed.Seconds()
	jobsPerSec := float64(numJobs) / elapsed.Seconds()

	require.Greater(t, throughputMBps, float64(0.5))
	require.Greater(t, jobsPerSec, float64(5.0))
}

func TestLargeObjectMultipartProgressAndCancellation(t *testing.T) {
	mockServer := newMockPerformanceS3Server()
	defer mockServer.Close()

	s3Client := mockServer.Client()
	tieredPool := transferAdapters.NewTieredBufferPool()
	svc, repo, pub := setupTestTransferService(t, s3Client, tieredPool, 8)

	tmpDir := t.TempDir()

	t.Run("ProgressTelemetryVerification", func(t *testing.T) {
		const partSize = 5 * 1024 * 1024
		const partCount = 4
		const totalSize = int64(partCount * partSize)

		largeFilePath := filepath.Join(tmpDir, "large_telemetry.dat")
		file, err := os.Create(largeFilePath)
		require.NoError(t, err)

		dummyChunk := make([]byte, 1024*1024)
		for range partCount * 5 {
			_, _ = file.Write(dummyChunk)
		}
		_ = file.Close()

		ctx := context.Background()
		jobID := "perf-large-telemetry"
		eventsCh, cleanupSub := pub.Subscribe(ctx, jobID)
		defer cleanupSub()

		job := transferDomain.TransferJob{
			ID:              jobID,
			AccountID:       "acc-perf",
			Type:            transferDomain.TransferTypeUpload,
			SourcePath:      largeFilePath,
			DestinationPath: "s3://test-perf-bucket/large_telemetry.dat",
			Bucket:          "test-perf-bucket",
			Key:             "large_telemetry.dat",
			TotalBytes:      totalSize,
			PartSize:        partSize,
		}

		startTime := time.Now()
		_, err = svc.SubmitJob(ctx, job)
		require.NoError(t, err)

		var receivedEvents []transferPorts.TransferEvent
		timeout := time.After(30 * time.Second)
		finished := false

		for !finished {
			select {
			case <-timeout:
				t.Fatalf("timed out waiting for multipart transfer to complete")
			case ev := <-eventsCh:
				receivedEvents = append(receivedEvents, ev)
				if ev.Status == transferDomain.JobStatusCompleted {
					finished = true
				}
			}
		}

		elapsed := time.Since(startTime)
		throughputMBps := (float64(totalSize) / (1024 * 1024)) / elapsed.Seconds()
		require.Greater(t, throughputMBps, float64(1.0))

		require.NotEmpty(t, receivedEvents)
		hasProgressUpdates := false
		for _, ev := range receivedEvents {
			if ev.Metrics.BytesTransferred > 0 {
				hasProgressUpdates = true
			}
		}
		require.True(t, hasProgressUpdates)

		finalJob, err := repo.GetJob(ctx, jobID)
		require.NoError(t, err)
		require.Equal(t, transferDomain.JobStatusCompleted, finalJob.Status)
		require.Equal(t, totalSize, finalJob.BytesTransferred)
	})

	t.Run("CancellationSimulation", func(t *testing.T) {
		const partSize = 5 * 1024 * 1024
		const partCount = 8
		const totalSize = int64(partCount * partSize)

		cancelFilePath := filepath.Join(tmpDir, "large_cancel.dat")
		file, err := os.Create(cancelFilePath)
		require.NoError(t, err)

		dummyChunk := make([]byte, 1024*1024)
		for range partCount * 5 {
			_, _ = file.Write(dummyChunk)
		}
		_ = file.Close()

		ctx := context.Background()
		jobID := "perf-large-cancellation"
		eventsCh, cleanupSub := pub.Subscribe(ctx, jobID)
		defer cleanupSub()

		job := transferDomain.TransferJob{
			ID:              jobID,
			AccountID:       "acc-perf",
			Type:            transferDomain.TransferTypeUpload,
			SourcePath:      cancelFilePath,
			DestinationPath: "s3://test-perf-bucket/large_cancel.dat",
			Bucket:          "test-perf-bucket",
			Key:             "large_cancel.dat",
			TotalBytes:      totalSize,
			PartSize:        partSize,
		}

		_, err = svc.SubmitJob(ctx, job)
		require.NoError(t, err)

		cancelled := false
		timeout := time.After(15 * time.Second)

		for !cancelled {
			select {
			case <-timeout:
				t.Fatalf("timeout waiting for running event before cancellation")
			case ev := <-eventsCh:
				if ev.Status == transferDomain.JobStatusRunning {
					err := svc.CancelJob(ctx, jobID)
					require.NoError(t, err)
					cancelled = true
				}
			}
		}

		require.Eventually(t, func() bool {
			j, err := repo.GetJob(ctx, jobID)
			return err == nil && j.Status == transferDomain.JobStatusCanceled
		}, 10*time.Second, 10*time.Millisecond)
	})
}
