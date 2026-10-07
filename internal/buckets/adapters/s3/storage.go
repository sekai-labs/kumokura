package s3

import (
	"context"
	"strings"
	"github.com/aws/aws-sdk-go-v2/aws"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sekai-labs/kumokura/internal/buckets/domain"
	"github.com/sekai-labs/kumokura/internal/buckets/ports"
)

type S3ClientAPI interface {
	ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
	CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
	DeleteBucket(ctx context.Context, params *s3.DeleteBucketInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error)
	GetBucketLocation(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error)
	GetBucketVersioning(ctx context.Context, params *s3.GetBucketVersioningInput, optFns ...func(*s3.Options)) (*s3.GetBucketVersioningOutput, error)
	PutBucketVersioning(ctx context.Context, params *s3.PutBucketVersioningInput, optFns ...func(*s3.Options)) (*s3.PutBucketVersioningOutput, error)
	GetBucketLifecycleConfiguration(ctx context.Context, params *s3.GetBucketLifecycleConfigurationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLifecycleConfigurationOutput, error)
	PutBucketLifecycleConfiguration(ctx context.Context, params *s3.PutBucketLifecycleConfigurationInput, optFns ...func(*s3.Options)) (*s3.PutBucketLifecycleConfigurationOutput, error)
	GetBucketPolicy(ctx context.Context, params *s3.GetBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.GetBucketPolicyOutput, error)
	PutBucketPolicy(ctx context.Context, params *s3.PutBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.PutBucketPolicyOutput, error)
	DeleteBucketPolicy(ctx context.Context, params *s3.DeleteBucketPolicyInput, optFns ...func(*s3.Options)) (*s3.DeleteBucketPolicyOutput, error)
	GetBucketEncryption(ctx context.Context, params *s3.GetBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error)
	PutBucketEncryption(ctx context.Context, params *s3.PutBucketEncryptionInput, optFns ...func(*s3.Options)) (*s3.PutBucketEncryptionOutput, error)
	GetPublicAccessBlock(ctx context.Context, params *s3.GetPublicAccessBlockInput, optFns ...func(*s3.Options)) (*s3.GetPublicAccessBlockOutput, error)
	PutPublicAccessBlock(ctx context.Context, params *s3.PutPublicAccessBlockInput, optFns ...func(*s3.Options)) (*s3.PutPublicAccessBlockOutput, error)
	GetBucketTagging(ctx context.Context, params *s3.GetBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error)
	PutBucketTagging(ctx context.Context, params *s3.PutBucketTaggingInput, optFns ...func(*s3.Options)) (*s3.PutBucketTaggingOutput, error)
	GetBucketCors(ctx context.Context, params *s3.GetBucketCorsInput, optFns ...func(*s3.Options)) (*s3.GetBucketCorsOutput, error)
	PutBucketCors(ctx context.Context, params *s3.PutBucketCorsInput, optFns ...func(*s3.Options)) (*s3.PutBucketCorsOutput, error)
}

type S3BucketStorage struct {
	client S3ClientAPI
}

func NewS3BucketStorage(client S3ClientAPI) ports.BucketStorage {
	return &S3BucketStorage{
		client: client,
	}
}

func (s *S3BucketStorage) ListBuckets(ctx context.Context) ([]domain.Bucket, error) {
	out, err := s.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	result := make([]domain.Bucket, 0, len(out.Buckets))
	for _, b := range out.Buckets {
		var name string
		if b.Name != nil {
			name = *b.Name
		}
		var creationTime domain.Bucket
		if b.CreationDate != nil {
			creationTime.CreationDate = *b.CreationDate
		}
		result = append(result, domain.Bucket{
			Name:         name,
			CreationDate: creationTime.CreationDate,
		})
	}
	return result, nil
}

func (s *S3BucketStorage) CreateBucket(ctx context.Context, bucket string, region string) error {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	}
	if region != "" && region != "us-east-1" {
		input.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(region),
		}
	}
	_, err := s.client.CreateBucket(ctx, input)
	return err
}

func (s *S3BucketStorage) DeleteBucket(ctx context.Context, bucket string) error {
	_, err := s.client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucket),
	})
	return err
}

func (s *S3BucketStorage) GetBucketLocation(ctx context.Context, bucket string) (string, error) {
	out, err := s.client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return "", err
	}
	loc := string(out.LocationConstraint)
	if loc == "" {
		loc = "us-east-1"
	}
	return loc, nil
}

func (s *S3BucketStorage) GetBucketVersioning(ctx context.Context, bucket string) (domain.VersioningConfig, error) {
	out, err := s.client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return domain.VersioningConfig{}, err
	}

	config := domain.VersioningConfig{
		Status: domain.VersioningStatusDisabled,
	}
	if out.Status != "" {
		config.Status = domain.VersioningStatus(out.Status)
	}
	if out.MFADelete != "" {
		config.MFADelete = out.MFADelete == s3types.MFADeleteStatusEnabled
	}
	return config, nil
}

func (s *S3BucketStorage) SetBucketVersioning(ctx context.Context, bucket string, config domain.VersioningConfig) error {
	input := &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucket),
		VersioningConfiguration: &s3types.VersioningConfiguration{
			Status: s3types.BucketVersioningStatus(config.Status),
		},
	}
	if config.MFADelete {
		input.VersioningConfiguration.MFADelete = s3types.MFADelete(s3types.MFADeleteStatusEnabled)
	}
	_, err := s.client.PutBucketVersioning(ctx, input)
	return err
}

func (s *S3BucketStorage) GetBucketLifecycle(ctx context.Context, bucket string) ([]domain.LifecycleRule, error) {
	out, err := s.client.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchLifecycleConfiguration") {
			return nil, nil
		}
		return nil, err
	}

	rules := make([]domain.LifecycleRule, 0, len(out.Rules))
	for _, r := range out.Rules {
		rule := domain.LifecycleRule{
			ID:     aws.ToString(r.ID),
			Status: string(r.Status),
			Prefix: aws.ToString(r.Prefix),
		}
		if r.Expiration != nil {
			rule.Expiration = &domain.LifecycleExpiration{
				Days:                      aws.ToInt32(r.Expiration.Days),
				Date:                      r.Expiration.Date,
				ExpiredObjectDeleteMarker: aws.ToBool(r.Expiration.ExpiredObjectDeleteMarker),
			}
		}
		for _, t := range r.Transitions {
			rule.Transitions = append(rule.Transitions, domain.LifecycleTransition{
				Days:         aws.ToInt32(t.Days),
				Date:         t.Date,
				StorageClass: domain.StorageClass(t.StorageClass),
			})
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (s *S3BucketStorage) SetBucketLifecycle(ctx context.Context, bucket string, rules []domain.LifecycleRule) error {
	s3Rules := make([]s3types.LifecycleRule, 0, len(rules))
	for _, r := range rules {
		rule := s3types.LifecycleRule{
			ID:     aws.String(r.ID),
			Status: s3types.ExpirationStatus(r.Status),
			Prefix: aws.String(r.Prefix),
		}
		if r.Expiration != nil {
			rule.Expiration = &s3types.LifecycleExpiration{
				Days:                      aws.Int32(r.Expiration.Days),
				Date:                      r.Expiration.Date,
				ExpiredObjectDeleteMarker: aws.Bool(r.Expiration.ExpiredObjectDeleteMarker),
			}
		}
		for _, t := range r.Transitions {
			rule.Transitions = append(rule.Transitions, s3types.Transition{
				Days:         aws.Int32(t.Days),
				Date:         t.Date,
				StorageClass: s3types.TransitionStorageClass(t.StorageClass),
			})
		}
		s3Rules = append(s3Rules, rule)
	}

	_, err := s.client.PutBucketLifecycleConfiguration(ctx, &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucket),
		LifecycleConfiguration: &s3types.BucketLifecycleConfiguration{
			Rules: s3Rules,
		},
	})
	return err
}

func (s *S3BucketStorage) GetBucketPolicy(ctx context.Context, bucket string) (string, error) {
	out, err := s.client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.Policy), nil
}

func (s *S3BucketStorage) SetBucketPolicy(ctx context.Context, bucket string, policy string) error {
	_, err := s.client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(bucket),
		Policy: aws.String(policy),
	})
	return err
}

func (s *S3BucketStorage) DeleteBucketPolicy(ctx context.Context, bucket string) error {
	_, err := s.client.DeleteBucketPolicy(ctx, &s3.DeleteBucketPolicyInput{
		Bucket: aws.String(bucket),
	})
	return err
}

func (s *S3BucketStorage) GetBucketEncryption(ctx context.Context, bucket string) (domain.EncryptionConfig, error) {
	out, err := s.client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return domain.EncryptionConfig{}, err
	}

	config := domain.EncryptionConfig{}
	if out.ServerSideEncryptionConfiguration != nil && len(out.ServerSideEncryptionConfiguration.Rules) > 0 {
		rule := out.ServerSideEncryptionConfiguration.Rules[0]
		config.BucketKey = aws.ToBool(rule.BucketKeyEnabled)
		if rule.ApplyServerSideEncryptionByDefault != nil {
			config.SSEAlgorithm = string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm)
			config.KMSMasterKeyID = aws.ToString(rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID)
		}
	}
	return config, nil
}

func (s *S3BucketStorage) SetBucketEncryption(ctx context.Context, bucket string, config domain.EncryptionConfig) error {
	_, err := s.client.PutBucketEncryption(ctx, &s3.PutBucketEncryptionInput{
		Bucket: aws.String(bucket),
		ServerSideEncryptionConfiguration: &s3types.ServerSideEncryptionConfiguration{
			Rules: []s3types.ServerSideEncryptionRule{
				{
					BucketKeyEnabled: aws.Bool(config.BucketKey),
					ApplyServerSideEncryptionByDefault: &s3types.ServerSideEncryptionByDefault{
						SSEAlgorithm:   s3types.ServerSideEncryption(config.SSEAlgorithm),
						KMSMasterKeyID: aws.String(config.KMSMasterKeyID),
					},
				},
			},
		},
	})
	return err
}

func (s *S3BucketStorage) GetPublicAccessBlock(ctx context.Context, bucket string) (domain.PublicAccessBlock, error) {
	out, err := s.client.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return domain.PublicAccessBlock{}, err
	}
	if out.PublicAccessBlockConfiguration == nil {
		return domain.PublicAccessBlock{}, nil
	}

	p := out.PublicAccessBlockConfiguration
	return domain.PublicAccessBlock{
		BlockPublicAcls:       aws.ToBool(p.BlockPublicAcls),
		IgnorePublicAcls:      aws.ToBool(p.IgnorePublicAcls),
		BlockPublicPolicy:     aws.ToBool(p.BlockPublicPolicy),
		RestrictPublicBuckets: aws.ToBool(p.RestrictPublicBuckets),
	}, nil
}

func (s *S3BucketStorage) SetPublicAccessBlock(ctx context.Context, bucket string, block domain.PublicAccessBlock) error {
	_, err := s.client.PutPublicAccessBlock(ctx, &s3.PutPublicAccessBlockInput{
		Bucket: aws.String(bucket),
		PublicAccessBlockConfiguration: &s3types.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(block.BlockPublicAcls),
			IgnorePublicAcls:      aws.Bool(block.IgnorePublicAcls),
			BlockPublicPolicy:     aws.Bool(block.BlockPublicPolicy),
			RestrictPublicBuckets: aws.Bool(block.RestrictPublicBuckets),
		},
	})
	return err
}

func (s *S3BucketStorage) GetBucketTags(ctx context.Context, bucket string) (domain.TagSet, error) {
	out, err := s.client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return nil, err
	}

	tags := make(domain.TagSet, 0, len(out.TagSet))
	for _, t := range out.TagSet {
		tags = append(tags, domain.Tag{
			Key:   aws.ToString(t.Key),
			Value: aws.ToString(t.Value),
		})
	}
	return tags, nil
}

func (s *S3BucketStorage) SetBucketTags(ctx context.Context, bucket string, tags domain.TagSet) error {
	s3Tags := make([]s3types.Tag, 0, len(tags))
	for _, t := range tags {
		s3Tags = append(s3Tags, s3types.Tag{
			Key:   aws.String(t.Key),
			Value: aws.String(t.Value),
		})
	}
	_, err := s.client.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
		Bucket: aws.String(bucket),
		Tagging: &s3types.Tagging{
			TagSet: s3Tags,
		},
	})
	return err
}

func (s *S3BucketStorage) GetBucketCORS(ctx context.Context, bucket string) ([]domain.CORSRule, error) {
	out, err := s.client.GetBucketCors(ctx, &s3.GetBucketCorsInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return nil, err
	}

	rules := make([]domain.CORSRule, 0, len(out.CORSRules))
	for _, r := range out.CORSRules {
		rules = append(rules, domain.CORSRule{
			AllowedHeaders: r.AllowedHeaders,
			AllowedMethods: r.AllowedMethods,
			AllowedOrigins: r.AllowedOrigins,
			ExposeHeaders:  r.ExposeHeaders,
			MaxAgeSeconds:  aws.ToInt32(r.MaxAgeSeconds),
		})
	}
	return rules, nil
}

func (s *S3BucketStorage) SetBucketCORS(ctx context.Context, bucket string, rules []domain.CORSRule) error {
	s3Rules := make([]s3types.CORSRule, 0, len(rules))
	for _, r := range rules {
		s3Rules = append(s3Rules, s3types.CORSRule{
			AllowedHeaders: r.AllowedHeaders,
			AllowedMethods: r.AllowedMethods,
			AllowedOrigins: r.AllowedOrigins,
			ExposeHeaders:  r.ExposeHeaders,
			MaxAgeSeconds:  aws.Int32(r.MaxAgeSeconds),
		})
	}
	_, err := s.client.PutBucketCors(ctx, &s3.PutBucketCorsInput{
		Bucket: aws.String(bucket),
		CORSConfiguration: &s3types.CORSConfiguration{
			CORSRules: s3Rules,
		},
	})
	return err
}
