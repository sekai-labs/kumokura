package application

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/sekai-labs/kumokura/internal/objects/domain"
	"github.com/sekai-labs/kumokura/internal/objects/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockObjectStorage struct {
	mock.Mock
}

func (m *mockObjectStorage) ListObjects(ctx context.Context, bucket string, filter domain.ObjectFilter) (ports.ListObjectsResult, error) {
	args := m.Called(ctx, bucket, filter)
	return args.Get(0).(ports.ListObjectsResult), args.Error(1)
}

func (m *mockObjectStorage) ListObjectVersions(ctx context.Context, bucket string, prefix string, keyMarker string, versionIDMarker string, maxKeys int32) (ports.ListObjectVersionsResult, error) {
	args := m.Called(ctx, bucket, prefix, keyMarker, versionIDMarker, maxKeys)
	return args.Get(0).(ports.ListObjectVersionsResult), args.Error(1)
}

func (m *mockObjectStorage) GetObject(ctx context.Context, bucket string, key string, versionID string) (ports.ObjectContent, error) {
	args := m.Called(ctx, bucket, key, versionID)
	return args.Get(0).(ports.ObjectContent), args.Error(1)
}

func (m *mockObjectStorage) PutObject(ctx context.Context, bucket string, key string, body io.Reader, size int64, metadata domain.ObjectMetadata) (domain.ObjectMetadata, error) {
	args := m.Called(ctx, bucket, key, body, size, metadata)
	return args.Get(0).(domain.ObjectMetadata), args.Error(1)
}

func (m *mockObjectStorage) DeleteObject(ctx context.Context, bucket string, key string, versionID string) error {
	args := m.Called(ctx, bucket, key, versionID)
	return args.Error(0)
}

func (m *mockObjectStorage) DeleteObjects(ctx context.Context, bucket string, keys []string) ([]string, error) {
	args := m.Called(ctx, bucket, keys)
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockObjectStorage) CopyObject(ctx context.Context, srcBucket, srcKey, srcVersionID, destBucket, destKey string) error {
	args := m.Called(ctx, srcBucket, srcKey, srcVersionID, destBucket, destKey)
	return args.Error(0)
}

func (m *mockObjectStorage) GetObjectMetadata(ctx context.Context, bucket string, key string, versionID string) (domain.ObjectMetadata, error) {
	args := m.Called(ctx, bucket, key, versionID)
	return args.Get(0).(domain.ObjectMetadata), args.Error(1)
}

func (m *mockObjectStorage) SetObjectMetadata(ctx context.Context, bucket string, key string, metadata domain.ObjectMetadata) error {
	args := m.Called(ctx, bucket, key, metadata)
	return args.Error(0)
}

func (m *mockObjectStorage) GeneratePresignedURL(ctx context.Context, bucket string, key string, method string, expiry time.Duration) (domain.PresignedURL, error) {
	args := m.Called(ctx, bucket, key, method, expiry)
	return args.Get(0).(domain.PresignedURL), args.Error(1)
}

func (m *mockObjectStorage) GetObjectTags(ctx context.Context, bucket string, key string, versionID string) ([]domain.ObjectTag, error) {
	args := m.Called(ctx, bucket, key, versionID)
	return args.Get(0).([]domain.ObjectTag), args.Error(1)
}

func (m *mockObjectStorage) SetObjectTags(ctx context.Context, bucket string, key string, versionID string, tags []domain.ObjectTag) error {
	args := m.Called(ctx, bucket, key, versionID, tags)
	return args.Error(0)
}
func (m *mockObjectStorage) DeleteObjectTags(ctx context.Context, bucket string, key string, versionID string) error {
	args := m.Called(ctx, bucket, key, versionID)
	return args.Error(0)
}

func (m *mockObjectStorage) GetObjectLegalHold(ctx context.Context, bucket string, key string, versionID string) (domain.LegalHoldStatus, error) {
	args := m.Called(ctx, bucket, key, versionID)
	return args.Get(0).(domain.LegalHoldStatus), args.Error(1)
}

func (m *mockObjectStorage) SetObjectLegalHold(ctx context.Context, bucket string, key string, versionID string, status domain.LegalHoldStatus) error {
	args := m.Called(ctx, bucket, key, versionID, status)
	return args.Error(0)
}

func (m *mockObjectStorage) GetObjectRetention(ctx context.Context, bucket string, key string, versionID string) (domain.Retention, error) {
	args := m.Called(ctx, bucket, key, versionID)
	return args.Get(0).(domain.Retention), args.Error(1)
}

func (m *mockObjectStorage) SetObjectRetention(ctx context.Context, bucket string, key string, versionID string, retention domain.Retention) error {
	args := m.Called(ctx, bucket, key, versionID, retention)
	return args.Error(0)
}

func (m *mockObjectStorage) RestoreObject(ctx context.Context, bucket string, key string, versionID string, req domain.RestoreRequest) error {
	args := m.Called(ctx, bucket, key, versionID, req)
	return args.Error(0)
}

func TestObjectService_KeyValidation(t *testing.T) {
	mockStore := new(mockObjectStorage)
	svc := NewObjectService(mockStore)
	ctx := context.Background()

	_, err := svc.GetObject(ctx, "b", "", "")
	assert.ErrorIs(t, err, domain.ErrEmptyObjectKey)

	err = svc.DeleteObject(ctx, "b", "   ", "")
	assert.ErrorIs(t, err, domain.ErrEmptyObjectKey)
}

func TestObjectService_BrowsePrefix(t *testing.T) {
	mockStore := new(mockObjectStorage)
	svc := NewObjectService(mockStore)
	ctx := context.Background()

	expected := ports.ListObjectsResult{
		Objects: []domain.Object{
			{Bucket: "b", Key: "photos/cat.jpg", Size: 1024},
		},
		CommonPrefixes: []domain.Prefix{
			{Bucket: "b", Prefix: "photos/2026/"},
		},
	}

	filter := domain.ObjectFilter{
		Prefix:       "photos/",
		Delimiter:    "/",
		MaxKeys:      1000,
		Continuation: "",
	}

	mockStore.On("ListObjects", ctx, "b", filter).Return(expected, nil)

	res, err := svc.BrowsePrefix(ctx, "b", "photos/", "")
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
	mockStore.AssertExpectations(t)
}

func TestObjectService_MoveAndRename(t *testing.T) {
	mockStore := new(mockObjectStorage)
	svc := NewObjectService(mockStore)
	ctx := context.Background()

	mockStore.On("CopyObject", ctx, "b", "old.txt", "", "b", "new.txt").Return(nil)
	mockStore.On("DeleteObject", ctx, "b", "old.txt", "").Return(nil)

	err := svc.RenameObject(ctx, "b", "old.txt", "new.txt")
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestObjectService_BatchDelete(t *testing.T) {
	mockStore := new(mockObjectStorage)
	svc := NewObjectService(mockStore)
	ctx := context.Background()

	keys := []string{"k1", "k2", "k3"}
	mockStore.On("DeleteObjects", ctx, "b", keys).Return(keys, nil)

	deleted, err := svc.BatchDeleteObjects(ctx, "b", keys)
	assert.NoError(t, err)
	assert.Equal(t, keys, deleted)
	mockStore.AssertExpectations(t)
}

func TestObjectService_SearchObjects(t *testing.T) {
	mockStore := new(mockObjectStorage)
	svc := NewObjectService(mockStore)
	ctx := context.Background()

	mockStore.On("ListObjects", ctx, "b", domain.ObjectFilter{
		Prefix:       "",
		MaxKeys:      1000,
		Continuation: "",
	}).Return(ports.ListObjectsResult{
		Objects: []domain.Object{
			{Key: "images/cat.png"},
			{Key: "images/dog.png"},
			{Key: "docs/readme.txt"},
		},
		IsTruncated: false,
	}, nil)

	found, err := svc.SearchObjects(ctx, "b", "", "cat")
	assert.NoError(t, err)
	assert.Len(t, found, 1)
	assert.Equal(t, "images/cat.png", found[0].Key)
	mockStore.AssertExpectations(t)
}

func TestObjectService_PutObject(t *testing.T) {
	mockStore := new(mockObjectStorage)
	svc := NewObjectService(mockStore)
	ctx := context.Background()

	buf := bytes.NewReader([]byte("hello world"))
	meta := domain.ObjectMetadata{ContentType: "text/plain"}

	mockStore.On("PutObject", ctx, "b", "test.txt", buf, int64(11), meta).Return(meta, nil)

	res, err := svc.PutObject(ctx, "b", "test.txt", buf, 11, meta)
	assert.NoError(t, err)
	assert.Equal(t, "text/plain", res.ContentType)
	mockStore.AssertExpectations(t)
}

func TestObjectService_TagsAndLegalHold(t *testing.T) {
	mockStore := new(mockObjectStorage)
	svc := NewObjectService(mockStore)
	ctx := context.Background()

	mockStore.On("DeleteObjectTags", ctx, "b", "obj.txt", "").Return(nil)
	err := svc.DeleteObjectTags(ctx, "b", "obj.txt", "")
	assert.NoError(t, err)

	mockStore.On("GetObjectLegalHold", ctx, "b", "obj.txt", "").Return(domain.LegalHoldStatusOn, nil)
	lh, err := svc.GetObjectLegalHold(ctx, "b", "obj.txt", "")
	assert.NoError(t, err)
	assert.Equal(t, domain.LegalHoldStatusOn, lh)

	mockStore.On("SetObjectLegalHold", ctx, "b", "obj.txt", "", domain.LegalHoldStatusOff).Return(nil)
	err = svc.SetObjectLegalHold(ctx, "b", "obj.txt", "", domain.LegalHoldStatusOff)
	assert.NoError(t, err)

	ret := domain.Retention{Mode: domain.RetentionModeCompliance, RetainUntilDate: time.Now()}
	mockStore.On("GetObjectRetention", ctx, "b", "obj.txt", "").Return(ret, nil)
	gotRet, err := svc.GetObjectRetention(ctx, "b", "obj.txt", "")
	assert.NoError(t, err)
	assert.Equal(t, domain.RetentionModeCompliance, gotRet.Mode)

	mockStore.AssertExpectations(t)
}
