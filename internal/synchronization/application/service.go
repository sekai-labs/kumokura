package application

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sekai-labs/kumokura/internal/synchronization/domain"
	"github.com/sekai-labs/kumokura/internal/synchronization/ports"
)

type SyncService struct {
	repo ports.SyncRepository
}

func NewSyncService(repo ports.SyncRepository) *SyncService {
	return &SyncService{repo: repo}
}

var _ ports.SyncEngine = (*SyncService)(nil)

func (s *SyncService) Plan(
	ctx context.Context,
	sourceScanner, destScanner ports.SyncScanner,
	sourceTarget, destTarget string,
	opts ports.SyncOptions,
) (*domain.SyncPlan, error) {
	if opts.TimeTolerance == 0 {
		opts.TimeTolerance = 2 * time.Second
	}
	if opts.ConflictPolicy == "" {
		opts.ConflictPolicy = domain.ConflictPolicyKeepNewer
	}
	if opts.Strategy == "" {
		opts.Strategy = domain.StrategyModTimeAndSize
	}

	sourceFiles, err := sourceScanner.Scan(ctx, sourceTarget, opts.Filter)
	if err != nil {
		return nil, fmt.Errorf("scan source %s: %w", sourceTarget, err)
	}

	destFiles, err := destScanner.Scan(ctx, destTarget, opts.Filter)
	if err != nil {
		return nil, fmt.Errorf("scan destination %s: %w", destTarget, err)
	}

	allKeysMap := make(map[string]struct{})
	for k := range sourceFiles {
		allKeysMap[k] = struct{}{}
	}
	for k := range destFiles {
		allKeysMap[k] = struct{}{}
	}

	var allKeys []string
	for k := range allKeysMap {
		allKeys = append(allKeys, k)
	}
	sort.Strings(allKeys)

	var (
		items          []domain.SyncItem
		totalUploads   int
		totalDownloads int
		totalDeletes   int
		totalSkips     int
		totalConflicts int
		totalBytes     int64
	)

	for _, key := range allKeys {
		if err := domain.ValidateSyncPath(key); err != nil {
			continue
		}
		src, hasSrc := sourceFiles[key]
		dst, hasDst := destFiles[key]
		item := domain.SyncItem{
			RelativePath: key,
			SourceEntry:  src,
			DestEntry:    dst,
		}

		if hasSrc {
			item.SourceSize = src.Size
			item.SourceMod = src.ModTime
		}
		if hasDst {
			item.DestSize = dst.Size
			item.DestMod = dst.ModTime
		}

		if hasSrc && !hasDst {
			item.Action = domain.SyncActionCreate
			item.Reason = "file exists only in source"
			totalUploads++
			totalBytes += src.Size
			items = append(items, item)
			continue
		}

		if !hasSrc && hasDst {
			if opts.Mode == domain.ModeBidirectional {
				item.Action = domain.SyncActionCreate
				item.Reason = "file exists only in destination (bidirectional pull)"
				totalDownloads++
				totalBytes += dst.Size
				items = append(items, item)
				continue
			}

			if opts.DeleteExtraneous {
				item.Action = domain.SyncActionDelete
				item.Reason = "file missing in source; extraneous deletion"
				totalDeletes++
				items = append(items, item)
			} else {
				item.Action = domain.SyncActionSkip
				item.Reason = "file missing in source; deletion disabled"
				totalSkips++
				items = append(items, item)
			}
			continue
		}

		if hasSrc && hasDst {
			isSame := compareFiles(src, dst, opts)
			if isSame {
				item.Action = domain.SyncActionSkip
				item.Reason = "source and destination files are identical"
				totalSkips++
				items = append(items, item)
				continue
			}

			resolveConflictOrUpdate(&item, src, dst, opts)
			switch item.Action {
			case domain.SyncActionUpdate:
				totalUploads++
				totalBytes += src.Size
			case domain.SyncActionConflict:
				totalConflicts++
			case domain.SyncActionSkip:
				totalSkips++
			}
			items = append(items, item)
		}
	}

	plan := &domain.SyncPlan{
		Items:          items,
		TotalUploads:   totalUploads,
		TotalDownloads: totalDownloads,
		TotalDeletes:   totalDeletes,
		TotalSkips:     totalSkips,
		TotalConflicts: totalConflicts,
		TotalBytes:     totalBytes,
		CreatedAt:      time.Now().UTC(),
	}

	return plan, nil
}

func compareFiles(src, dst *domain.FileEntry, opts ports.SyncOptions) bool {
	switch opts.Strategy {
	case domain.StrategySizeOnly:
		return src.Size == dst.Size

	case domain.StrategyChecksum:
		if src.Checksum != "" && dst.Checksum != "" && !domain.IsMultipartETag(src.ETag) && !domain.IsMultipartETag(dst.ETag) {
			return domain.NormalizeETag(src.Checksum) == domain.NormalizeETag(dst.Checksum)
		}
		if src.Size != dst.Size {
			return false
		}
		return domain.TimestampsMatchWithTolerance(src.ModTime, dst.ModTime, opts.TimeTolerance)

	case domain.StrategyModTimeAndSize:
		fallthrough
	default:
		if src.Size != dst.Size {
			return false
		}
		return domain.TimestampsMatchWithTolerance(src.ModTime, dst.ModTime, opts.TimeTolerance)
	}
}

func resolveConflictOrUpdate(item *domain.SyncItem, src, dst *domain.FileEntry, opts ports.SyncOptions) {
	switch opts.ConflictPolicy {
	case domain.ConflictPolicyOverwrite:
		item.Action = domain.SyncActionUpdate
		item.Reason = "overwrite destination with source file"

	case domain.ConflictPolicySkip:
		item.Action = domain.SyncActionSkip
		item.Reason = "file difference detected; conflict policy skip"

	case domain.ConflictPolicyFail:
		item.Action = domain.SyncActionConflict
		item.Reason = "file difference detected; conflict policy fail"

	case domain.ConflictPolicyKeepNewer:
		fallthrough
	default:
		if src.ModTime.After(dst.ModTime) {
			item.Action = domain.SyncActionUpdate
			item.Reason = "source file is newer"
		} else if dst.ModTime.After(src.ModTime) {
			if opts.Mode == domain.ModeBidirectional {
				item.Action = domain.SyncActionUpdate
				item.Reason = "destination file is newer (bidirectional update)"
			} else {
				item.Action = domain.SyncActionSkip
				item.Reason = "destination file is newer than source; skipping update"
			}
		} else {
			item.Action = domain.SyncActionUpdate
			item.Reason = "files differ with same timestamp; overwriting"
		}
	}
}

func (s *SyncService) Execute(ctx context.Context, plan *domain.SyncPlan, opts ports.SyncOptions) error {
	if opts.DryRun || plan == nil || len(plan.Items) == 0 {
		return nil
	}
	concurrency := opts.MaxConcurrency
	if concurrency <= 0 {
		concurrency = 100
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var execErr error
	for _, item := range plan.Items {
		if item.Action == domain.SyncActionSkip {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(it domain.SyncItem) {
			defer func() {
				<-sem
				wg.Done()
			}()
		}(item)
	}
	wg.Wait()
	return execErr
}

func (s *SyncService) Cancel(ctx context.Context, jobID string) error {
	if s.repo != nil {
		return s.repo.UpdateStatus(ctx, jobID, "CANCELLED")
	}
	return nil
}
