package s3

import (
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/sekai-labs/kumokura/internal/objects/domain"
	"github.com/sekai-labs/kumokura/internal/objects/ports"
)

type S3ObjectClientAPI interface {
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	ListObjectVersions(ctx context.Context, params *s3.ListObjectVersionsInput, optFns ...func(*s3.Options)) (*s3.ListObjectVersionsOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	DeleteObjects(ctx context.Context, params *s3.DeleteObjectsInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectsOutput, error)
	CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	GetObjectTagging(ctx context.Context, params *s3.GetObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.GetObjectTaggingOutput, error)
	PutObjectTagging(ctx context.Context, params *s3.PutObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.PutObjectTaggingOutput, error)
	DeleteObjectTagging(ctx context.Context, params *s3.DeleteObjectTaggingInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectTaggingOutput, error)
	GetObjectLegalHold(ctx context.Context, params *s3.GetObjectLegalHoldInput, optFns ...func(*s3.Options)) (*s3.GetObjectLegalHoldOutput, error)
	PutObjectLegalHold(ctx context.Context, params *s3.PutObjectLegalHoldInput, optFns ...func(*s3.Options)) (*s3.PutObjectLegalHoldOutput, error)
	GetObjectRetention(ctx context.Context, params *s3.GetObjectRetentionInput, optFns ...func(*s3.Options)) (*s3.GetObjectRetentionOutput, error)
	PutObjectRetention(ctx context.Context, params *s3.PutObjectRetentionInput, optFns ...func(*s3.Options)) (*s3.PutObjectRetentionOutput, error)
	RestoreObject(ctx context.Context, params *s3.RestoreObjectInput, optFns ...func(*s3.Options)) (*s3.RestoreObjectOutput, error)
}

type S3PresignClientAPI interface {
	PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
	PresignPutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

type S3ObjectStorage struct {
	client        S3ObjectClientAPI
	presignClient S3PresignClientAPI
}

func NewS3ObjectStorage(client S3ObjectClientAPI, presignClient S3PresignClientAPI) ports.ObjectStorage {
	return &S3ObjectStorage{
		client:        client,
		presignClient: presignClient,
	}
}

func (s *S3ObjectStorage) ListObjects(ctx context.Context, bucket string, filter domain.ObjectFilter) (ports.ListObjectsResult, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}
	if filter.Prefix != "" {
		input.Prefix = aws.String(filter.Prefix)
	}
	if filter.Delimiter != "" {
		input.Delimiter = aws.String(filter.Delimiter)
	}
	if filter.MaxKeys > 0 {
		input.MaxKeys = aws.Int32(filter.MaxKeys)
	}
	if filter.Continuation != "" {
		input.ContinuationToken = aws.String(filter.Continuation)
	}

	out, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return ports.ListObjectsResult{}, err
	}

	result := ports.ListObjectsResult{
		IsTruncated:           aws.ToBool(out.IsTruncated),
		NextContinuationToken: aws.ToString(out.NextContinuationToken),
		Objects:               make([]domain.Object, 0, len(out.Contents)),
		CommonPrefixes:        make([]domain.Prefix, 0, len(out.CommonPrefixes)),
	}

	for _, item := range out.Contents {
		var lastMod time.Time
		if item.LastModified != nil {
			lastMod = *item.LastModified
		}
		result.Objects = append(result.Objects, domain.Object{
			Bucket:       bucket,
			Key:          aws.ToString(item.Key),
			Size:         aws.ToInt64(item.Size),
			ETag:         aws.ToString(item.ETag),
			LastModified: lastMod,
			StorageClass: domain.StorageClass(item.StorageClass),
		})
	}

	for _, p := range out.CommonPrefixes {
		result.CommonPrefixes = append(result.CommonPrefixes, domain.Prefix{
			Bucket: bucket,
			Prefix: aws.ToString(p.Prefix),
		})
	}

	return result, nil
}

func (s *S3ObjectStorage) ListObjectVersions(ctx context.Context, bucket string, prefix string, keyMarker string, versionIDMarker string, maxKeys int32) (ports.ListObjectVersionsResult, error) {
	input := &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}
	if keyMarker != "" {
		input.KeyMarker = aws.String(keyMarker)
	}
	if versionIDMarker != "" {
		input.VersionIdMarker = aws.String(versionIDMarker)
	}
	if maxKeys > 0 {
		input.MaxKeys = aws.Int32(maxKeys)
	}

	out, err := s.client.ListObjectVersions(ctx, input)
	if err != nil {
		return ports.ListObjectVersionsResult{}, err
	}

	result := ports.ListObjectVersionsResult{
		IsTruncated:         aws.ToBool(out.IsTruncated),
		NextKeyMarker:       aws.ToString(out.NextKeyMarker),
		NextVersionIDMarker: aws.ToString(out.NextVersionIdMarker),
		Versions:            make([]domain.ObjectVersion, 0, len(out.Versions)+len(out.DeleteMarkers)),
	}

	for _, v := range out.Versions {
		var lastMod time.Time
		if v.LastModified != nil {
			lastMod = *v.LastModified
		}
		result.Versions = append(result.Versions, domain.ObjectVersion{
			Bucket:       bucket,
			Key:          aws.ToString(v.Key),
			VersionID:    aws.ToString(v.VersionId),
			IsLatest:     aws.ToBool(v.IsLatest),
			IsDelete:     false,
			LastModified: lastMod,
			Size:         aws.ToInt64(v.Size),
			ETag:         aws.ToString(v.ETag),
		})
	}

	for _, d := range out.DeleteMarkers {
		var lastMod time.Time
		if d.LastModified != nil {
			lastMod = *d.LastModified
		}
		result.Versions = append(result.Versions, domain.ObjectVersion{
			Bucket:       bucket,
			Key:          aws.ToString(d.Key),
			VersionID:    aws.ToString(d.VersionId),
			IsLatest:     aws.ToBool(d.IsLatest),
			IsDelete:     true,
			LastModified: lastMod,
		})
	}

	return result, nil
}

func (s *S3ObjectStorage) GetObject(ctx context.Context, bucket string, key string, versionID string) (ports.ObjectContent, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if versionID != "" {
		input.VersionId = aws.String(versionID)
	}

	out, err := s.client.GetObject(ctx, input)
	if err != nil {
		return ports.ObjectContent{}, err
	}

	var lastMod time.Time
	if out.LastModified != nil {
		lastMod = *out.LastModified
	}

	metadata := domain.ObjectMetadata{
		ContentType:     aws.ToString(out.ContentType),
		ContentLength:   aws.ToInt64(out.ContentLength),
		ETag:            aws.ToString(out.ETag),
		LastModified:    lastMod,
		StorageClass:    domain.StorageClass(out.StorageClass),
		VersionID:       aws.ToString(out.VersionId),
		IsDeleteMarker:  aws.ToBool(out.DeleteMarker),
		CacheControl:    aws.ToString(out.CacheControl),
		ContentEncoding: aws.ToString(out.ContentEncoding),
		UserMetadata:    out.Metadata,
	}

	return ports.ObjectContent{
		Body:     out.Body,
		Metadata: metadata,
	}, nil
}

func (s *S3ObjectStorage) PutObject(ctx context.Context, bucket string, key string, body io.Reader, size int64, metadata domain.ObjectMetadata) (domain.ObjectMetadata, error) {
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
	}
	if size >= 0 {
		input.ContentLength = aws.Int64(size)
	}
	if metadata.ContentType != "" {
		input.ContentType = aws.String(metadata.ContentType)
	}
	if metadata.CacheControl != "" {
		input.CacheControl = aws.String(metadata.CacheControl)
	}
	if metadata.ContentEncoding != "" {
		input.ContentEncoding = aws.String(metadata.ContentEncoding)
	}
	if metadata.StorageClass != "" {
		input.StorageClass = s3types.StorageClass(metadata.StorageClass)
	}
	if len(metadata.UserMetadata) > 0 {
		input.Metadata = metadata.UserMetadata
	}

	out, err := s.client.PutObject(ctx, input)
	if err != nil {
		return domain.ObjectMetadata{}, err
	}

	return domain.ObjectMetadata{
		ETag:      aws.ToString(out.ETag),
		VersionID: aws.ToString(out.VersionId),
	}, nil
}

func (s *S3ObjectStorage) DeleteObject(ctx context.Context, bucket string, key string, versionID string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if versionID != "" {
		input.VersionId = aws.String(versionID)
	}
	_, err := s.client.DeleteObject(ctx, input)
	return err
}

func (s *S3ObjectStorage) DeleteObjects(ctx context.Context, bucket string, keys []string) ([]string, error) {
	objects := make([]s3types.ObjectIdentifier, 0, len(keys))
	for _, k := range keys {
		objects = append(objects, s3types.ObjectIdentifier{
			Key: aws.String(k),
		})
	}

	out, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket),
		Delete: &s3types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(false),
		},
	})
	if err != nil {
		return nil, err
	}

	deleted := make([]string, 0, len(out.Deleted))
	for _, d := range out.Deleted {
		deleted = append(deleted, aws.ToString(d.Key))
	}
	return deleted, nil
}

func (s *S3ObjectStorage) CopyObject(ctx context.Context, srcBucket, srcKey, srcVersionID, destBucket, destKey string) error {
	copySource := srcBucket + "/" + srcKey
	if srcVersionID != "" {
		copySource += "?versionId=" + srcVersionID
	}

	input := &s3.CopyObjectInput{
		Bucket:     aws.String(destBucket),
		Key:        aws.String(destKey),
		CopySource: aws.String(copySource),
	}
	_, err := s.client.CopyObject(ctx, input)
	return err
}

func (s *S3ObjectStorage) GetObjectMetadata(ctx context.Context, bucket string, key string, versionID string) (domain.ObjectMetadata, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if versionID != "" {
		input.VersionId = aws.String(versionID)
	}

	out, err := s.client.HeadObject(ctx, input)
	if err != nil {
		return domain.ObjectMetadata{}, err
	}

	var lastMod time.Time
	if out.LastModified != nil {
		lastMod = *out.LastModified
	}

	return domain.ObjectMetadata{
		ContentType:     aws.ToString(out.ContentType),
		ContentLength:   aws.ToInt64(out.ContentLength),
		ETag:            aws.ToString(out.ETag),
		LastModified:    lastMod,
		StorageClass:    domain.StorageClass(out.StorageClass),
		VersionID:       aws.ToString(out.VersionId),
		IsDeleteMarker:  aws.ToBool(out.DeleteMarker),
		CacheControl:    aws.ToString(out.CacheControl),
		ContentEncoding: aws.ToString(out.ContentEncoding),
		UserMetadata:    out.Metadata,
	}, nil
}

func (s *S3ObjectStorage) SetObjectMetadata(ctx context.Context, bucket string, key string, metadata domain.ObjectMetadata) error {
	copySource := bucket + "/" + key
	input := &s3.CopyObjectInput{
		Bucket:            aws.String(bucket),
		Key:               aws.String(key),
		CopySource:        aws.String(copySource),
		MetadataDirective: s3types.MetadataDirectiveReplace,
		Metadata:          metadata.UserMetadata,
	}
	if metadata.ContentType != "" {
		input.ContentType = aws.String(metadata.ContentType)
	}
	if metadata.CacheControl != "" {
		input.CacheControl = aws.String(metadata.CacheControl)
	}
	if metadata.ContentEncoding != "" {
		input.ContentEncoding = aws.String(metadata.ContentEncoding)
	}
	if metadata.StorageClass != "" {
		input.StorageClass = s3types.StorageClass(metadata.StorageClass)
	}

	_, err := s.client.CopyObject(ctx, input)
	return err
}

func (s *S3ObjectStorage) GeneratePresignedURL(ctx context.Context, bucket string, key string, method string, expiry time.Duration) (domain.PresignedURL, error) {
	if s.presignClient == nil {
		return domain.PresignedURL{}, nil
	}

	if method == "PUT" {
		req, err := s.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}, func(opts *s3.PresignOptions) {
			opts.Expires = expiry
		})
		if err != nil {
			return domain.PresignedURL{}, err
		}
		return domain.PresignedURL{
			URL:        req.URL,
			Method:     req.Method,
			Expiration: time.Now().Add(expiry),
		}, nil
	}

	req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return domain.PresignedURL{}, err
	}
	return domain.PresignedURL{
		URL:        req.URL,
		Method:     req.Method,
		Expiration: time.Now().Add(expiry),
	}, nil
}

func (s *S3ObjectStorage) GetObjectTags(ctx context.Context, bucket string, key string, versionID string) ([]domain.ObjectTag, error) {
	input := &s3.GetObjectTaggingInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if versionID != "" {
		input.VersionId = aws.String(versionID)
	}

	out, err := s.client.GetObjectTagging(ctx, input)
	if err != nil {
		return nil, err
	}

	tags := make([]domain.ObjectTag, 0, len(out.TagSet))
	for _, t := range out.TagSet {
		tags = append(tags, domain.ObjectTag{
			Key:   aws.ToString(t.Key),
			Value: aws.ToString(t.Value),
		})
	}
	return tags, nil
}

func (s *S3ObjectStorage) SetObjectTags(ctx context.Context, bucket string, key string, versionID string, tags []domain.ObjectTag) error {
	s3Tags := make([]s3types.Tag, 0, len(tags))
	for _, t := range tags {
		s3Tags = append(s3Tags, s3types.Tag{
			Key:   aws.String(t.Key),
			Value: aws.String(t.Value),
		})
	}

	input := &s3.PutObjectTaggingInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Tagging: &s3types.Tagging{
			TagSet: s3Tags,
		},
	}
	if versionID != "" {
		input.VersionId = aws.String(versionID)
	}

	_, err := s.client.PutObjectTagging(ctx, input)
	return err
}
func (s *S3ObjectStorage) DeleteObjectTags(ctx context.Context, bucket string, key string, versionID string) error {
	input := &s3.DeleteObjectTaggingInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if versionID != "" {
		input.VersionId = aws.String(versionID)
	}
	_, err := s.client.DeleteObjectTagging(ctx, input)
	return err
}

func (s *S3ObjectStorage) GetObjectLegalHold(ctx context.Context, bucket string, key string, versionID string) (domain.LegalHoldStatus, error) {
	input := &s3.GetObjectLegalHoldInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if versionID != "" {
		input.VersionId = aws.String(versionID)
	}
	out, err := s.client.GetObjectLegalHold(ctx, input)
	if err != nil {
		return "", err
	}
	if out.LegalHold == nil {
		return domain.LegalHoldStatusOff, nil
	}
	return domain.LegalHoldStatus(out.LegalHold.Status), nil
}

func (s *S3ObjectStorage) SetObjectLegalHold(ctx context.Context, bucket string, key string, versionID string, status domain.LegalHoldStatus) error {
	input := &s3.PutObjectLegalHoldInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		LegalHold: &s3types.ObjectLockLegalHold{
			Status: s3types.ObjectLockLegalHoldStatus(status),
		},
	}
	if versionID != "" {
		input.VersionId = aws.String(versionID)
	}
	_, err := s.client.PutObjectLegalHold(ctx, input)
	return err
}

func (s *S3ObjectStorage) GetObjectRetention(ctx context.Context, bucket string, key string, versionID string) (domain.Retention, error) {
	input := &s3.GetObjectRetentionInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if versionID != "" {
		input.VersionId = aws.String(versionID)
	}
	out, err := s.client.GetObjectRetention(ctx, input)
	if err != nil {
		return domain.Retention{}, err
	}
	if out.Retention == nil {
		return domain.Retention{}, nil
	}
	var retainDate time.Time
	if out.Retention.RetainUntilDate != nil {
		retainDate = *out.Retention.RetainUntilDate
	}
	return domain.Retention{
		Mode:            domain.RetentionMode(out.Retention.Mode),
		RetainUntilDate: retainDate,
	}, nil
}

func (s *S3ObjectStorage) SetObjectRetention(ctx context.Context, bucket string, key string, versionID string, retention domain.Retention) error {
	input := &s3.PutObjectRetentionInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Retention: &s3types.ObjectLockRetention{
			Mode:            s3types.ObjectLockRetentionMode(retention.Mode),
			RetainUntilDate: aws.Time(retention.RetainUntilDate),
		},
	}
	if versionID != "" {
		input.VersionId = aws.String(versionID)
	}
	_, err := s.client.PutObjectRetention(ctx, input)
	return err
}

func (s *S3ObjectStorage) RestoreObject(ctx context.Context, bucket string, key string, versionID string, req domain.RestoreRequest) error {
	input := &s3.RestoreObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		RestoreRequest: &s3types.RestoreRequest{
			Days: aws.Int32(req.Days),
			GlacierJobParameters: &s3types.GlacierJobParameters{
				Tier: s3types.Tier(req.GlacierJobTier),
			},
		},
	}
	if versionID != "" {
		input.VersionId = aws.String(versionID)
	}

	_, err := s.client.RestoreObject(ctx, input)
	return err
}
