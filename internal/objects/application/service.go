package application

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/sekai-labs/kumokura/internal/objects/domain"
	"github.com/sekai-labs/kumokura/internal/objects/ports"
)

type ObjectService struct {
	storage ports.ObjectStorage
}

func NewObjectService(storage ports.ObjectStorage) *ObjectService {
	return &ObjectService{
		storage: storage,
	}
}

func (s *ObjectService) ListObjects(ctx context.Context, bucket string, filter domain.ObjectFilter) (ports.ListObjectsResult, error) {
	return s.storage.ListObjects(ctx, bucket, filter)
}

func (s *ObjectService) ListObjectVersions(ctx context.Context, bucket string, prefix string, keyMarker string, versionMarker string, maxKeys int32) (ports.ListObjectVersionsResult, error) {
	return s.storage.ListObjectVersions(ctx, bucket, prefix, keyMarker, versionMarker, maxKeys)
}

func (s *ObjectService) BrowsePrefix(ctx context.Context, bucket string, prefix string, continuationToken string) (ports.ListObjectsResult, error) {
	filter := domain.ObjectFilter{
		Prefix:       prefix,
		Delimiter:    "/",
		MaxKeys:      1000,
		Continuation: continuationToken,
	}
	return s.storage.ListObjects(ctx, bucket, filter)
}

func (s *ObjectService) SearchObjects(ctx context.Context, bucket string, prefix string, searchPattern string) ([]domain.Object, error) {
	var results []domain.Object
	continuation := ""

	for {
		res, err := s.storage.ListObjects(ctx, bucket, domain.ObjectFilter{
			Prefix:       prefix,
			MaxKeys:      1000,
			Continuation: continuation,
		})
		if err != nil {
			return nil, err
		}

		for _, obj := range res.Objects {
			if strings.Contains(strings.ToLower(obj.Key), strings.ToLower(searchPattern)) {
				results = append(results, obj)
			}
		}

		if !res.IsTruncated || res.NextContinuationToken == "" {
			break
		}
		continuation = res.NextContinuationToken
	}

	return results, nil
}

func (s *ObjectService) GetObject(ctx context.Context, bucket string, key string, versionID string) (ports.ObjectContent, error) {
	if err := domain.ValidateObjectKey(key); err != nil {
		return ports.ObjectContent{}, err
	}
	return s.storage.GetObject(ctx, bucket, key, versionID)
}

func (s *ObjectService) PutObject(ctx context.Context, bucket string, key string, body io.Reader, size int64, metadata domain.ObjectMetadata) (domain.ObjectMetadata, error) {
	if err := domain.ValidateObjectKey(key); err != nil {
		return domain.ObjectMetadata{}, err
	}
	return s.storage.PutObject(ctx, bucket, key, body, size, metadata)
}

func (s *ObjectService) DeleteObject(ctx context.Context, bucket string, key string, versionID string) error {
	if err := domain.ValidateObjectKey(key); err != nil {
		return err
	}
	return s.storage.DeleteObject(ctx, bucket, key, versionID)
}

func (s *ObjectService) BatchDeleteObjects(ctx context.Context, bucket string, keys []string) ([]string, error) {
	var deleted []string
	const batchSize = 1000

	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]
		d, err := s.storage.DeleteObjects(ctx, bucket, batch)
		if err != nil {
			return deleted, err
		}
		deleted = append(deleted, d...)
	}
	return deleted, nil
}

func (s *ObjectService) CopyObject(ctx context.Context, srcBucket, srcKey, srcVersionID, destBucket, destKey string) error {
	if err := domain.ValidateObjectKey(srcKey); err != nil {
		return err
	}
	if err := domain.ValidateObjectKey(destKey); err != nil {
		return err
	}
	return s.storage.CopyObject(ctx, srcBucket, srcKey, srcVersionID, destBucket, destKey)
}

func (s *ObjectService) MoveObject(ctx context.Context, srcBucket, srcKey, destBucket, destKey string) error {
	if err := s.CopyObject(ctx, srcBucket, srcKey, "", destBucket, destKey); err != nil {
		return err
	}
	return s.DeleteObject(ctx, srcBucket, srcKey, "")
}

func (s *ObjectService) RenameObject(ctx context.Context, bucket string, oldKey string, newKey string) error {
	return s.MoveObject(ctx, bucket, oldKey, bucket, newKey)
}

func (s *ObjectService) GetObjectMetadata(ctx context.Context, bucket string, key string, versionID string) (domain.ObjectMetadata, error) {
	if err := domain.ValidateObjectKey(key); err != nil {
		return domain.ObjectMetadata{}, err
	}
	return s.storage.GetObjectMetadata(ctx, bucket, key, versionID)
}

func (s *ObjectService) SetObjectMetadata(ctx context.Context, bucket string, key string, metadata domain.ObjectMetadata) error {
	if err := domain.ValidateObjectKey(key); err != nil {
		return err
	}
	return s.storage.SetObjectMetadata(ctx, bucket, key, metadata)
}

func (s *ObjectService) GeneratePresignedURL(ctx context.Context, bucket string, key string, method string, expiry time.Duration) (domain.PresignedURL, error) {
	if err := domain.ValidateObjectKey(key); err != nil {
		return domain.PresignedURL{}, err
	}
	return s.storage.GeneratePresignedURL(ctx, bucket, key, method, expiry)
}

func (s *ObjectService) GetObjectTags(ctx context.Context, bucket string, key string, versionID string) ([]domain.ObjectTag, error) {
	if err := domain.ValidateObjectKey(key); err != nil {
		return nil, err
	}
	return s.storage.GetObjectTags(ctx, bucket, key, versionID)
}

func (s *ObjectService) SetObjectTags(ctx context.Context, bucket string, key string, versionID string, tags []domain.ObjectTag) error {
	if err := domain.ValidateObjectKey(key); err != nil {
		return err
	}
	return s.storage.SetObjectTags(ctx, bucket, key, versionID, tags)
}
func (s *ObjectService) DeleteObjectTags(ctx context.Context, bucket string, key string, versionID string) error {
	if err := domain.ValidateObjectKey(key); err != nil {
		return err
	}
	return s.storage.DeleteObjectTags(ctx, bucket, key, versionID)
}

func (s *ObjectService) GetObjectLegalHold(ctx context.Context, bucket string, key string, versionID string) (domain.LegalHoldStatus, error) {
	if err := domain.ValidateObjectKey(key); err != nil {
		return "", err
	}
	return s.storage.GetObjectLegalHold(ctx, bucket, key, versionID)
}

func (s *ObjectService) SetObjectLegalHold(ctx context.Context, bucket string, key string, versionID string, status domain.LegalHoldStatus) error {
	if err := domain.ValidateObjectKey(key); err != nil {
		return err
	}
	return s.storage.SetObjectLegalHold(ctx, bucket, key, versionID, status)
}

func (s *ObjectService) GetObjectRetention(ctx context.Context, bucket string, key string, versionID string) (domain.Retention, error) {
	if err := domain.ValidateObjectKey(key); err != nil {
		return domain.Retention{}, err
	}
	return s.storage.GetObjectRetention(ctx, bucket, key, versionID)
}

func (s *ObjectService) SetObjectRetention(ctx context.Context, bucket string, key string, versionID string, retention domain.Retention) error {
	if err := domain.ValidateObjectKey(key); err != nil {
		return err
	}
	return s.storage.SetObjectRetention(ctx, bucket, key, versionID, retention)
}

func (s *ObjectService) RestoreObject(ctx context.Context, bucket string, key string, versionID string, req domain.RestoreRequest) error {
	if err := domain.ValidateObjectKey(key); err != nil {
		return err
	}
	return s.storage.RestoreObject(ctx, bucket, key, versionID, req)
}
