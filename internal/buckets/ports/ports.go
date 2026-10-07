package ports

import (
	"context"

	"github.com/sekai-labs/kumokura/internal/buckets/domain"
)

type BucketStorage interface {
	ListBuckets(ctx context.Context) ([]domain.Bucket, error)
	CreateBucket(ctx context.Context, bucket string, region string) error
	DeleteBucket(ctx context.Context, bucket string) error
	GetBucketLocation(ctx context.Context, bucket string) (string, error)
	GetBucketVersioning(ctx context.Context, bucket string) (domain.VersioningConfig, error)
	SetBucketVersioning(ctx context.Context, bucket string, config domain.VersioningConfig) error
	GetBucketLifecycle(ctx context.Context, bucket string) ([]domain.LifecycleRule, error)
	SetBucketLifecycle(ctx context.Context, bucket string, rules []domain.LifecycleRule) error
	GetBucketPolicy(ctx context.Context, bucket string) (string, error)
	SetBucketPolicy(ctx context.Context, bucket string, policy string) error
	DeleteBucketPolicy(ctx context.Context, bucket string) error
	GetBucketEncryption(ctx context.Context, bucket string) (domain.EncryptionConfig, error)
	SetBucketEncryption(ctx context.Context, bucket string, config domain.EncryptionConfig) error
	GetPublicAccessBlock(ctx context.Context, bucket string) (domain.PublicAccessBlock, error)
	SetPublicAccessBlock(ctx context.Context, bucket string, block domain.PublicAccessBlock) error
	GetBucketTags(ctx context.Context, bucket string) (domain.TagSet, error)
	SetBucketTags(ctx context.Context, bucket string, tags domain.TagSet) error
	GetBucketCORS(ctx context.Context, bucket string) ([]domain.CORSRule, error)
	SetBucketCORS(ctx context.Context, bucket string, rules []domain.CORSRule) error
}
