package s3

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sekai-labs/kumokura/internal/objects/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockS3ObjectClientAPI struct {
	mock.Mock
}

func (m *mockS3ObjectClientAPI) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.ListObjectsV2Output), args.Error(1)
}

func (m *mockS3ObjectClientAPI) ListObjectVersions(ctx context.Context, params *s3.ListObjectVersionsInput, optFns ...func(*s3.Options)) (*s3.ListObjectVersionsOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.ListObjectVersionsOutput), args.Error(1)
}

func (m *mockS3ObjectClientAPI) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

func (m *mockS3ObjectClientAPI) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

func (m *mockS3ObjectClientAPI) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

func (m *mockS3ObjectClientAPI) DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.DeleteObjectsOutput), args.Error(1)
}

func (m *mockS3ObjectClientAPI) CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

func (m *mockS3ObjectClientAPI) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.HeadObjectOutput), args.Error(1)
}

func (m *mockS3ObjectClientAPI) GetObjectTagging(ctx context.Context, params *s3.GetObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.GetObjectTaggingOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.GetObjectTaggingOutput), args.Error(1)
}

func (m *mockS3ObjectClientAPI) PutObjectTagging(ctx context.Context, params *s3.PutObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.PutObjectTaggingOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

func (m *mockS3ObjectClientAPI) DeleteObjectTagging(ctx context.Context, params *s3.DeleteObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectTaggingOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

func (m *mockS3ObjectClientAPI) GetObjectLegalHold(ctx context.Context, params *s3.GetObjectLegalHoldInput, optFns ...func(*s3.Options)) (*s3.GetObjectLegalHoldOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.GetObjectLegalHoldOutput), args.Error(1)
}

func (m *mockS3ObjectClientAPI) PutObjectLegalHold(ctx context.Context, params *s3.PutObjectLegalHoldInput, optFns ...func(*s3.Options)) (*s3.PutObjectLegalHoldOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

func (m *mockS3ObjectClientAPI) GetObjectRetention(ctx context.Context, params *s3.GetObjectRetentionInput, optFns ...func(*s3.Options)) (*s3.GetObjectRetentionOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.GetObjectRetentionOutput), args.Error(1)
}

func (m *mockS3ObjectClientAPI) PutObjectRetention(ctx context.Context, params *s3.PutObjectRetentionInput, optFns ...func(*s3.Options)) (*s3.PutObjectRetentionOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

func (m *mockS3ObjectClientAPI) RestoreObject(ctx context.Context, params *s3.RestoreObjectInput, optFns ...func(*s3.Options)) (*s3.RestoreObjectOutput, error) {
	args := m.Called(ctx, params)
	return nil, args.Error(0)
}

type mockPresignClientAPI struct {
	mock.Mock
}

func (m *mockPresignClientAPI) PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v4.PresignedHTTPRequest), args.Error(1)
}

func (m *mockPresignClientAPI) PresignPutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v4.PresignedHTTPRequest), args.Error(1)
}

func TestS3ObjectStorage_ListObjects(t *testing.T) {
	clientMock := new(mockS3ObjectClientAPI)
	storage := NewS3ObjectStorage(clientMock, nil)
	ctx := context.Background()

	now := time.Now()
	clientMock.On("ListObjectsV2", ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String("b"),
		Prefix:    aws.String("photos/"),
		Delimiter: aws.String("/"),
	}).Return(&s3.ListObjectsV2Output{
		Contents: []s3types.Object{
			{
				Key:          aws.String("photos/cat.jpg"),
				Size:         aws.Int64(2048),
				ETag:         aws.String("\"etag123\""),
				LastModified: aws.Time(now),
				StorageClass: s3types.ObjectStorageClassStandard,
			},
		},
		CommonPrefixes: []s3types.CommonPrefix{
			{Prefix: aws.String("photos/2026/")},
		},
		IsTruncated: aws.Bool(false),
	}, nil)

	res, err := storage.ListObjects(ctx, "b", domain.ObjectFilter{
		Prefix:    "photos/",
		Delimiter: "/",
	})

	assert.NoError(t, err)
	assert.Len(t, res.Objects, 1)
	assert.Equal(t, "photos/cat.jpg", res.Objects[0].Key)
	assert.Equal(t, int64(2048), res.Objects[0].Size)
	assert.Len(t, res.CommonPrefixes, 1)
	assert.Equal(t, "photos/2026/", res.CommonPrefixes[0].Prefix)
}

func TestS3ObjectStorage_GetObject(t *testing.T) {
	clientMock := new(mockS3ObjectClientAPI)
	storage := NewS3ObjectStorage(clientMock, nil)
	ctx := context.Background()

	content := "test content"
	body := io.NopCloser(bytes.NewReader([]byte(content)))

	clientMock.On("GetObject", ctx, &s3.GetObjectInput{
		Bucket: aws.String("b"),
		Key:    aws.String("file.txt"),
	}).Return(&s3.GetObjectOutput{
		Body:          body,
		ContentType:   aws.String("text/plain"),
		ContentLength: aws.Int64(int64(len(content))),
		ETag:          aws.String("\"etag\""),
	}, nil)

	obj, err := storage.GetObject(ctx, "b", "file.txt", "")
	assert.NoError(t, err)
	defer obj.Body.Close()

	readData, err := io.ReadAll(obj.Body)
	assert.NoError(t, err)
	assert.Equal(t, content, string(readData))
	assert.Equal(t, "text/plain", obj.Metadata.ContentType)
}

func TestS3ObjectStorage_Presign(t *testing.T) {
	clientMock := new(mockS3ObjectClientAPI)
	presignMock := new(mockPresignClientAPI)
	storage := NewS3ObjectStorage(clientMock, presignMock)
	ctx := context.Background()

	presignMock.On("PresignGetObject", ctx, &s3.GetObjectInput{
		Bucket: aws.String("b"),
		Key:    aws.String("test.jpg"),
	}, mock.Anything).Return(&v4.PresignedHTTPRequest{
		URL:    "https://b.s3.amazonaws.com/test.jpg?signed=true",
		Method: "GET",
	}, nil)

	pURL, err := storage.GeneratePresignedURL(ctx, "b", "test.jpg", "GET", 15*time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, "https://b.s3.amazonaws.com/test.jpg?signed=true", pURL.URL)
	assert.Equal(t, "GET", pURL.Method)
}
