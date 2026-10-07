package application

import (
	"context"
	"testing"
	"time"

	"github.com/sekai-labs/kumokura/internal/buckets/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockBucketStorage struct {
	mock.Mock
}

func (m *mockBucketStorage) ListBuckets(ctx context.Context) ([]domain.Bucket, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.Bucket), args.Error(1)
}

func (m *mockBucketStorage) CreateBucket(ctx context.Context, bucket string, region string) error {
	args := m.Called(ctx, bucket, region)
	return args.Error(0)
}

func (m *mockBucketStorage) DeleteBucket(ctx context.Context, bucket string) error {
	args := m.Called(ctx, bucket)
	return args.Error(0)
}

func (m *mockBucketStorage) GetBucketLocation(ctx context.Context, bucket string) (string, error) {
	args := m.Called(ctx, bucket)
	return args.String(0), args.Error(1)
}

func (m *mockBucketStorage) GetBucketVersioning(ctx context.Context, bucket string) (domain.VersioningConfig, error) {
	args := m.Called(ctx, bucket)
	return args.Get(0).(domain.VersioningConfig), args.Error(1)
}

func (m *mockBucketStorage) SetBucketVersioning(ctx context.Context, bucket string, config domain.VersioningConfig) error {
	args := m.Called(ctx, bucket, config)
	return args.Error(0)
}

func (m *mockBucketStorage) GetBucketLifecycle(ctx context.Context, bucket string) ([]domain.LifecycleRule, error) {
	args := m.Called(ctx, bucket)
	return args.Get(0).([]domain.LifecycleRule), args.Error(1)
}

func (m *mockBucketStorage) SetBucketLifecycle(ctx context.Context, bucket string, rules []domain.LifecycleRule) error {
	args := m.Called(ctx, bucket, rules)
	return args.Error(0)
}

func (m *mockBucketStorage) GetBucketPolicy(ctx context.Context, bucket string) (string, error) {
	args := m.Called(ctx, bucket)
	return args.String(0), args.Error(1)
}

func (m *mockBucketStorage) SetBucketPolicy(ctx context.Context, bucket string, policy string) error {
	args := m.Called(ctx, bucket, policy)
	return args.Error(0)
}

func (m *mockBucketStorage) DeleteBucketPolicy(ctx context.Context, bucket string) error {
	args := m.Called(ctx, bucket)
	return args.Error(0)
}

func (m *mockBucketStorage) GetBucketEncryption(ctx context.Context, bucket string) (domain.EncryptionConfig, error) {
	args := m.Called(ctx, bucket)
	return args.Get(0).(domain.EncryptionConfig), args.Error(1)
}

func (m *mockBucketStorage) SetBucketEncryption(ctx context.Context, bucket string, config domain.EncryptionConfig) error {
	args := m.Called(ctx, bucket, config)
	return args.Error(0)
}

func (m *mockBucketStorage) GetPublicAccessBlock(ctx context.Context, bucket string) (domain.PublicAccessBlock, error) {
	args := m.Called(ctx, bucket)
	return args.Get(0).(domain.PublicAccessBlock), args.Error(1)
}

func (m *mockBucketStorage) SetPublicAccessBlock(ctx context.Context, bucket string, block domain.PublicAccessBlock) error {
	args := m.Called(ctx, bucket, block)
	return args.Error(0)
}

func (m *mockBucketStorage) GetBucketTags(ctx context.Context, bucket string) (domain.TagSet, error) {
	args := m.Called(ctx, bucket)
	return args.Get(0).(domain.TagSet), args.Error(1)
}

func (m *mockBucketStorage) SetBucketTags(ctx context.Context, bucket string, tags domain.TagSet) error {
	args := m.Called(ctx, bucket, tags)
	return args.Error(0)
}

func (m *mockBucketStorage) GetBucketCORS(ctx context.Context, bucket string) ([]domain.CORSRule, error) {
	args := m.Called(ctx, bucket)
	return args.Get(0).([]domain.CORSRule), args.Error(1)
}

func (m *mockBucketStorage) SetBucketCORS(ctx context.Context, bucket string, rules []domain.CORSRule) error {
	args := m.Called(ctx, bucket, rules)
	return args.Error(0)
}

func TestBucketService_CreateBucket(t *testing.T) {
	mockStore := new(mockBucketStorage)
	svc := NewBucketService(mockStore)

	ctx := context.Background()

	err := svc.CreateBucket(ctx, "INVALID_NAME", "us-east-1")
	assert.Error(t, err)

	mockStore.On("CreateBucket", ctx, "valid-name", "us-east-1").Return(nil)
	err = svc.CreateBucket(ctx, "valid-name", "us-east-1")
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestBucketService_ListBuckets(t *testing.T) {
	mockStore := new(mockBucketStorage)
	svc := NewBucketService(mockStore)

	ctx := context.Background()
	expected := []domain.Bucket{
		{Name: "b1", Region: "us-east-1", CreationDate: time.Now()},
	}

	mockStore.On("ListBuckets", ctx).Return(expected, nil)
	result, err := svc.ListBuckets(ctx)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	mockStore.AssertExpectations(t)
}

func TestBucketService_Versioning(t *testing.T) {
	mockStore := new(mockBucketStorage)
	svc := NewBucketService(mockStore)
	ctx := context.Background()

	cfg := domain.VersioningConfig{Status: domain.VersioningStatusEnabled}
	mockStore.On("SetBucketVersioning", ctx, "valid-bucket", cfg).Return(nil)
	err := svc.SetBucketVersioning(ctx, "valid-bucket", cfg)
	assert.NoError(t, err)

	mockStore.On("GetBucketVersioning", ctx, "valid-bucket").Return(cfg, nil)
	res, err := svc.GetBucketVersioning(ctx, "valid-bucket")
	assert.NoError(t, err)
	assert.Equal(t, cfg, res)
	mockStore.AssertExpectations(t)
}
