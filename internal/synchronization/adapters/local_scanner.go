package adapters

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/sekai-labs/kumokura/internal/synchronization/domain"
	"github.com/sekai-labs/kumokura/internal/synchronization/ports"
)

type LocalScanner struct {
	CalculateChecksums bool
}

func NewLocalScanner(calcChecksums bool) *LocalScanner {
	return &LocalScanner{
		CalculateChecksums: calcChecksums,
	}
}

var _ ports.SyncScanner = (*LocalScanner)(nil)

func (s *LocalScanner) Scan(ctx context.Context, rootDir string, filter domain.Filter) (map[string]*domain.FileEntry, error) {
	results := make(map[string]*domain.FileEntry)
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("resolve local root %s: %w", rootDir, err)
	}

	type pendingHashEntry struct {
		absPath string
		entry   *domain.FileEntry
	}

	var pendingHashing []pendingHashEntry

	err = filepath.WalkDir(absRoot, func(p string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			return err
		}

		if p == absRoot {
			return nil
		}

		rel, err := filepath.Rel(absRoot, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == ".." || strings.HasPrefix(rel, "../") || strings.Contains(rel, "/../") {
			return nil
		}
		if d.IsDir() {
			return nil
		}

		if !filter.Matches(rel) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		entry := &domain.FileEntry{
			RelativePath: rel,
			Size:         info.Size(),
			ModTime:      info.ModTime().UTC(),
			IsDir:        false,
		}

		if s.CalculateChecksums {
			pendingHashing = append(pendingHashing, pendingHashEntry{
				absPath: p,
				entry:   entry,
			})
		}

		results[rel] = entry
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk local directory %s: %w", rootDir, err)
	}

	if s.CalculateChecksums && len(pendingHashing) > 0 {
		numWorkers := runtime.NumCPU() * 2
		if numWorkers < 1 {
			numWorkers = 1
		}
		if numWorkers > len(pendingHashing) {
			numWorkers = len(pendingHashing)
		}

		jobsCh := make(chan pendingHashEntry, numWorkers*2)
		var wg sync.WaitGroup
		var hashErr error
		var errMu sync.Mutex

		for range numWorkers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for item := range jobsCh {
					select {
					case <-ctx.Done():
						errMu.Lock()
						if hashErr == nil {
							hashErr = ctx.Err()
						}
						errMu.Unlock()
						return
					default:
					}

					hash, err := calculateLocalMD5(item.absPath)
					if err == nil {
						item.entry.Checksum = hash
						item.entry.ETag = hash
					}
				}
			}()
		}

		for _, item := range pendingHashing {
			select {
			case <-ctx.Done():
				errMu.Lock()
				if hashErr == nil {
					hashErr = ctx.Err()
				}
				errMu.Unlock()
				break
			case jobsCh <- item:
			}
		}
		close(jobsCh)
		wg.Wait()

		if hashErr != nil {
			return nil, hashErr
		}
	}

	return results, nil
}

func calculateLocalMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
