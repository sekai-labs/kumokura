package s3

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sekai-labs/kumokura/internal/buckets/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockS3ClientAPI struct {
	mock.Mock
}

func (m *mockS3ClientAPI) ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.ListBucketsOutput), args.Error(1)
}

func (m *mockS3ClientAPI) CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

func (m *mockS3ClientAPI) DeleteBucket(ctx context.Context, params *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

func (m *mockS3ClientAPI) GetBucketLocation(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.GetBucketLocationOutput), args.Error(1)
}

func (m *mockS3ClientAPI) GetBucketVersioning(ctx context.Context, params *s3.GetBucketVersioningInput, optFns ...func(*s3.Options)) (*s3.GetBucketVersioningOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.GetBucketVersioningOutput), args.Error(1)
}

func (m *mockS3ClientAPI) PutBucketVersioning(ctx context.Context, params *s3.PutBucketVersioningInput, optFns ...func(*s3.Options)) (*s3.PutBucketVersioningOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

func (m *mockS3ClientAPI) GetBucketLifecycleConfiguration(ctx context.Context, params *s3.GetBucketLifecycleConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLifecycleConfigurationOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.GetBucketLifecycleConfigurationOutput), args.Error(1)
}

func (m *mockS3ClientAPI) PutBucketLifecycleConfiguration(ctx context.Context, params *s3.PutBucketLifecycleConfigurationInput, optFns ...func(*s3.Options)) (*s3.PutBucketLifecycleConfigurationOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

func (m *mockS3ClientAPI) GetBucketPolicy(ctx context.Context, params *s3.GetBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.GetBucketPolicyOutput), args.Error(1)
}

func (m *mockS3ClientAPI) PutBucketPolicy(ctx context.Context, params *s3.PutBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.PutBucketPolicyOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

func (m *mockS3ClientAPI) DeleteBucketPolicy(ctx context.Context, params *s3.DeleteBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketPolicyOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

func (m *mockS3ClientAPI) GetBucketEncryption(ctx context.Context, params *s3.GetBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.GetBucketEncryptionOutput), args.Error(1)
}

func (m *mockS3ClientAPI) PutBucketEncryption(ctx context.Context, params *s3.PutBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.PutBucketEncryptionOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

func (m *mockS3ClientAPI) GetPublicAccessBlock(ctx context.Context, params *s3.GetPublicAccessBlockInput, optFns ...func(*s3.Options)) (*s3.GetPublicAccessBlockOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.GetPublicAccessBlockOutput), args.Error(1)
}

func (m *mockS3ClientAPI) PutPublicAccessBlock(ctx context.Context, params *s3.PutPublicAccessBlockInput, optFns ...func(*s3.Options)) (*s3.PutPublicAccessBlockOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

func (m *mockS3ClientAPI) GetBucketTagging(ctx context.Context, params *s3.GetBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.GetBucketTaggingOutput), args.Error(1)
}

func (m *mockS3ClientAPI) PutBucketTagging(ctx context.Context, params *s3.PutBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.PutBucketTaggingOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

func (m *mockS3ClientAPI) GetBucketCors(ctx context.Context, params *s3.GetBucketCorsInput, optFns ...func(*s3.Options)) (*s3.GetBucketCorsOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.GetBucketCorsOutput), args.Error(1)
}

func (m *mockS3ClientAPI) PutBucketCors(ctx context.Context, params *s3.PutBucketCorsInput, optFns ...func(*s3.Options)) (*s3.PutBucketCorsOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

func TestS3BucketStorage_ListBuckets(t *testing.T) {
	clientMock := new(mockS3ClientAPI)
	storage := NewS3BucketStorage(clientMock)
	ctx := context.Background()

	now := time.Now()
	clientMock.On("ListBuckets", ctx, &s3.ListBucketsInput{}).Return(&s3.ListBucketsOutput{
		Buckets: []s3types.Bucket{
			{
				Name:         aws.String("my-bucket"),
				CreationDate: aws.Time(now),
			},
		},
	}, nil)

	buckets, err := storage.ListBuckets(ctx)
	assert.NoError(t, err)
	assert.Len(t, buckets, 1)
	assert.Equal(t, "my-bucket", buckets[0].Name)
	assert.Equal(t, now, buckets[0].CreationDate)
}

func TestS3BucketStorage_CreateBucket(t *testing.T) {
	clientMock := new(mockS3ClientAPI)
	storage := NewS3BucketStorage(clientMock)
	ctx := context.Background()

	clientMock.On("CreateBucket", ctx, &s3.CreateBucketInput{
		Bucket: aws.String("my-bucket"),
	}).Return(nil)

	err := storage.CreateBucket(ctx, "my-bucket", "us-east-1")
	assert.NoError(t, err)

	clientMock.On("CreateBucket", ctx, &s3.CreateBucketInput{
		Bucket: aws.String("eu-bucket"),
		CreateBucketConfiguration: &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint("eu-west-1"),
		},
	}).Return(nil)

	err = storage.CreateBucket(ctx, "eu-bucket", "eu-west-1")
	assert.NoError(t, err)
}

func TestS3BucketStorage_GetSetVersioning(t *testing.T) {
	clientMock := new(mockS3ClientAPI)
	storage := NewS3BucketStorage(clientMock)
	ctx := context.Background()

	clientMock.On("GetBucketVersioning", ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String("test-bucket"),
	}).Return(&s3.GetBucketVersioningOutput{
		Status: s3types.BucketVersioningStatusEnabled,
	}, nil)

	cfg, err := storage.GetBucketVersioning(ctx, "test-bucket")
	assert.NoError(t, err)
	assert.Equal(t, domain.VersioningStatusEnabled, cfg.Status)

	clientMock.On("PutBucketVersioning", ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String("test-bucket"),
		VersioningConfiguration: &s3types.VersioningConfiguration{
			Status: s3types.BucketVersioningStatusSuspended,
		},
	}).Return(nil)

	err = storage.SetBucketVersioning(ctx, "test-bucket", domain.VersioningConfig{
		Status: domain.VersioningStatusSuspended,
	})
	assert.NoError(t, err)
}
