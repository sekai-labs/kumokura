package application

import (
	"context"

	"github.com/sekai-labs/kumokura/internal/buckets/domain"
	"github.com/sekai-labs/kumokura/internal/buckets/ports"
)

type BucketService struct {
	storage ports.BucketStorage
}

func NewBucketService(storage ports.BucketStorage) *BucketService {
	return &BucketService{
		storage: storage,
	}
}

func (s *BucketService) ListBuckets(ctx context.Context) ([]domain.Bucket, error) {
	return s.storage.ListBuckets(ctx)
}

func (s *BucketService) CreateBucket(ctx context.Context, name string, region string) error {
	if err := domain.ValidateBucketName(name); err != nil {
		return err
	}
	return s.storage.CreateBucket(ctx, name, region)
}

func (s *BucketService) DeleteBucket(ctx context.Context, name string) error {
	if err := domain.ValidateBucketName(name); err != nil {
		return err
	}
	return s.storage.DeleteBucket(ctx, name)
}

func (s *BucketService) GetBucketLocation(ctx context.Context, name string) (string, error) {
	if err := domain.ValidateBucketName(name); err != nil {
		return "", err
	}
	return s.storage.GetBucketLocation(ctx, name)
}

func (s *BucketService) GetBucketVersioning(ctx context.Context, name string) (domain.VersioningConfig, error) {
	if err := domain.ValidateBucketName(name); err != nil {
		return domain.VersioningConfig{}, err
	}
	return s.storage.GetBucketVersioning(ctx, name)
}

func (s *BucketService) SetBucketVersioning(ctx context.Context, name string, config domain.VersioningConfig) error {
	if err := domain.ValidateBucketName(name); err != nil {
		return err
	}
	return s.storage.SetBucketVersioning(ctx, name, config)
}

func (s *BucketService) GetBucketLifecycle(ctx context.Context, name string) ([]domain.LifecycleRule, error) {
	if err := domain.ValidateBucketName(name); err != nil {
		return nil, err
	}
	return s.storage.GetBucketLifecycle(ctx, name)
}

func (s *BucketService) SetBucketLifecycle(ctx context.Context, name string, rules []domain.LifecycleRule) error {
	if err := domain.ValidateBucketName(name); err != nil {
		return err
	}
	return s.storage.SetBucketLifecycle(ctx, name, rules)
}

func (s *BucketService) GetBucketPolicy(ctx context.Context, name string) (string, error) {
	if err := domain.ValidateBucketName(name); err != nil {
		return "", err
	}
	return s.storage.GetBucketPolicy(ctx, name)
}

func (s *BucketService) SetBucketPolicy(ctx context.Context, name string, policy string) error {
	if err := domain.ValidateBucketName(name); err != nil {
		return err
	}
	return s.storage.SetBucketPolicy(ctx, name, policy)
}

func (s *BucketService) DeleteBucketPolicy(ctx context.Context, name string) error {
	if err := domain.ValidateBucketName(name); err != nil {
		return err
	}
	return s.storage.DeleteBucketPolicy(ctx, name)
}

func (s *BucketService) GetBucketEncryption(ctx context.Context, name string) (domain.EncryptionConfig, error) {
	if err := domain.ValidateBucketName(name); err != nil {
		return domain.EncryptionConfig{}, err
	}
	return s.storage.GetBucketEncryption(ctx, name)
}

func (s *BucketService) SetBucketEncryption(ctx context.Context, name string, config domain.EncryptionConfig) error {
	if err := domain.ValidateBucketName(name); err != nil {
		return err
	}
	return s.storage.SetBucketEncryption(ctx, name, config)
}

func (s *BucketService) GetPublicAccessBlock(ctx context.Context, name string) (domain.PublicAccessBlock, error) {
	if err := domain.ValidateBucketName(name); err != nil {
		return domain.PublicAccessBlock{}, err
	}
	return s.storage.GetPublicAccessBlock(ctx, name)
}

func (s *BucketService) SetPublicAccessBlock(ctx context.Context, name string, block domain.PublicAccessBlock) error {
	if err := domain.ValidateBucketName(name); err != nil {
		return err
	}
	return s.storage.SetPublicAccessBlock(ctx, name, block)
}

func (s *BucketService) GetBucketTags(ctx context.Context, name string) (domain.TagSet, error) {
	if err := domain.ValidateBucketName(name); err != nil {
		return nil, err
	}
	return s.storage.GetBucketTags(ctx, name)
}

func (s *BucketService) SetBucketTags(ctx context.Context, name string, tags domain.TagSet) error {
	if err := domain.ValidateBucketName(name); err != nil {
		return err
	}
	return s.storage.SetBucketTags(ctx, name, tags)
}

func (s *BucketService) GetBucketCORS(ctx context.Context, name string) ([]domain.CORSRule, error) {
	if err := domain.ValidateBucketName(name); err != nil {
		return nil, err
	}
	return s.storage.GetBucketCORS(ctx, name)
}

func (s *BucketService) SetBucketCORS(ctx context.Context, name string, rules []domain.CORSRule) error {
	if err := domain.ValidateBucketName(name); err != nil {
		return err
	}
	return s.storage.SetBucketCORS(ctx, name, rules)
}
