package ports

import (
	"context"
	"io"
	"time"

	"github.com/sekai-labs/kumokura/internal/objects/domain"
)

type ListObjectsResult struct {
	Objects               []domain.Object
	CommonPrefixes        []domain.Prefix
	NextContinuationToken string
	IsTruncated           bool
}

type ListObjectVersionsResult struct {
	Versions              []domain.ObjectVersion
	NextKeyMarker         string
	NextVersionIDMarker   string
	IsTruncated           bool
}

type ObjectContent struct {
	Body          io.ReadCloser
	Metadata      domain.ObjectMetadata
}

type ObjectStorage interface {
	ListObjects(ctx context.Context, bucket string, filter domain.ObjectFilter) (ListObjectsResult, error)
	ListObjectVersions(ctx context.Context, bucket string, prefix string, keyMarker string, versionIDMarker string, maxKeys int32) (ListObjectVersionsResult, error)
	GetObject(ctx context.Context, bucket string, key string, versionID string) (ObjectContent, error)
	PutObject(ctx context.Context, bucket string, key string, body io.Reader, size int64, metadata domain.ObjectMetadata) (domain.ObjectMetadata, error)
	DeleteObject(ctx context.Context, bucket string, key string, versionID string) error
	DeleteObjects(ctx context.Context, bucket string, keys []string) ([]string, error)
	CopyObject(ctx context.Context, srcBucket, srcKey, srcVersionID, destBucket, destKey string) error
	GetObjectMetadata(ctx context.Context, bucket string, key string, versionID string) (domain.ObjectMetadata, error)
	SetObjectMetadata(ctx context.Context, bucket string, key string, metadata domain.ObjectMetadata) error
	GeneratePresignedURL(ctx context.Context, bucket string, key string, method string, expiry time.Duration) (domain.PresignedURL, error)
	GetObjectTags(ctx context.Context, bucket string, key string, versionID string) ([]domain.ObjectTag, error)
	SetObjectTags(ctx context.Context, bucket string, key string, versionID string, tags []domain.ObjectTag) error
	DeleteObjectTags(ctx context.Context, bucket string, key string, versionID string) error
	GetObjectLegalHold(ctx context.Context, bucket string, key string, versionID string) (domain.LegalHoldStatus, error)
	SetObjectLegalHold(ctx context.Context, bucket string, key string, versionID string, status domain.LegalHoldStatus) error
	GetObjectRetention(ctx context.Context, bucket string, key string, versionID string) (domain.Retention, error)
	SetObjectRetention(ctx context.Context, bucket string, key string, versionID string, retention domain.Retention) error
	RestoreObject(ctx context.Context, bucket string, key string, versionID string, req domain.RestoreRequest) error
}
