package adapters

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

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

	err = filepath.Walk(absRoot, func(p string, info os.FileInfo, err error) error {
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

		if info.IsDir() {
			return nil
		}

		if !filter.Matches(rel) {
			return nil
		}

		entry := &domain.FileEntry{
			RelativePath: rel,
			Size:         info.Size(),
			ModTime:      info.ModTime().UTC(),
			IsDir:        false,
		}

		if s.CalculateChecksums {
			hash, err := calculateLocalMD5(p)
			if err == nil {
				entry.Checksum = hash
				entry.ETag = hash
			}
		}

		results[rel] = entry
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk local directory %s: %w", rootDir, err)
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
