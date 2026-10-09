package domain

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"
)

type SyncDirection string

const (
	DirectionLocalToS3 SyncDirection = "LocalToS3"
	DirectionS3ToLocal SyncDirection = "S3ToLocal"
	DirectionS3ToS3    SyncDirection = "S3ToS3"
)

type SyncMode string

const (
	ModeUploadOnly    SyncMode = "UploadOnly"
	ModeDownloadOnly  SyncMode = "DownloadOnly"
	ModeBidirectional SyncMode = "Bidirectional"
)

type ComparisonStrategy string

const (
	StrategyModTimeAndSize ComparisonStrategy = "modtime-and-size"
	StrategySizeOnly       ComparisonStrategy = "size-only"
	StrategyChecksum       ComparisonStrategy = "checksum"
)

type ConflictPolicy string

const (
	ConflictPolicyOverwrite ConflictPolicy = "Overwrite"
	ConflictPolicySkip      ConflictPolicy = "Skip"
	ConflictPolicyKeepNewer ConflictPolicy = "KeepNewer"
	ConflictPolicyRename    ConflictPolicy = "Rename"
	ConflictPolicyFail      ConflictPolicy = "Fail"
)

type SyncAction string

const (
	SyncActionCreate   SyncAction = "CREATE"
	SyncActionUpdate   SyncAction = "UPDATE"
	SyncActionDelete   SyncAction = "DELETE"
	SyncActionSkip     SyncAction = "SKIP"
	SyncActionConflict SyncAction = "CONFLICT"
)

type FileEntry struct {
	RelativePath string
	Size         int64
	ModTime      time.Time
	ETag         string
	Checksum     string
	IsDir        bool
}

type SyncItem struct {
	RelativePath string
	SourceSize   int64
	SourceMod    time.Time
	DestSize     int64
	DestMod      time.Time
	Action       SyncAction
	Reason       string
	SourceEntry  *FileEntry
	DestEntry    *FileEntry
}

type SyncPlan struct {
	Items          []SyncItem
	TotalUploads   int
	TotalDownloads int
	TotalDeletes   int
	TotalSkips     int
	TotalConflicts int
	TotalBytes     int64
	CreatedAt      time.Time
}

type DiffResult struct {
	OnlyInSource []*FileEntry
	OnlyInDest   []*FileEntry
	Different    []*SyncItem
	Identical    []*SyncItem
}

type Filter struct {
	Includes []string
	Excludes []string
}

func (f *Filter) Matches(relPath string) bool {
	normalized := strings.ReplaceAll(relPath, "\\", "/")
	normalized = strings.TrimPrefix(normalized, "/")

	if len(f.Excludes) > 0 {
		for _, pattern := range f.Excludes {
			if matchPattern(pattern, normalized) {
				return false
			}
		}
	}

	if len(f.Includes) > 0 {
		for _, pattern := range f.Includes {
			if matchPattern(pattern, normalized) {
				return true
			}
		}
		return false
	}

	return true
}

func matchPattern(pattern, text string) bool {
	p := strings.ReplaceAll(pattern, "\\", "/")
	p = strings.TrimPrefix(p, "/")
	t := strings.ReplaceAll(text, "\\", "/")
	t = strings.TrimPrefix(t, "/")

	if !strings.Contains(p, "/") {
		base := path.Base(t)
		matched, err := path.Match(p, base)
		if err == nil && matched {
			return true
		}
	}

	if strings.Contains(p, "**") {
		regexStr := fmt.Sprintf("^%s$", regexp.QuoteMeta(p))
		regexStr = strings.ReplaceAll(regexStr, `\*\*`, `.*`)
		regexStr = strings.ReplaceAll(regexStr, `\*`, `[^/]*`)
		regexStr = strings.ReplaceAll(regexStr, `\?`, `.`)
		matched, err := regexp.MatchString(regexStr, t)
		if err == nil && matched {
			return true
		}
	}

	matched, err := path.Match(p, t)
	if err == nil && matched {
		return true
	}

	if strings.HasSuffix(p, "/") && strings.HasPrefix(t, p) {
		return true
	}

	return false
}

func NormalizeETag(etag string) string {
	s := strings.Trim(etag, "\"")
	return strings.ToLower(s)
}

func IsMultipartETag(etag string) bool {
	s := strings.Trim(etag, "\"")
	return strings.Contains(s, "-")
}

func TimestampsMatchWithTolerance(t1, t2 time.Time, tolerance time.Duration) bool {
	diff := t1.Sub(t2)
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
