package domain

import (
	"github.com/sekai-labs/kumokura/internal/accounts/domain"
)

type Feature string

const (
	FeatureVersioning        Feature = "Versioning"
	FeatureObjectLock        Feature = "ObjectLock"
	FeatureTagging           Feature = "Tagging"
	FeatureMultipart         Feature = "Multipart"
	FeatureLifecycle         Feature = "Lifecycle"
	FeatureServerEncryption  Feature = "ServerEncryption"
	FeatureCORS              Feature = "CORS"
	FeatureBucketPolicy      Feature = "BucketPolicy"
	FeaturePublicAccessBlock Feature = "PublicAccessBlock"
)

type ProviderCapability struct {
	SupportsVersioning        bool
	SupportsObjectLock        bool
	SupportsTagging           bool
	SupportsMultipart         bool
	SupportsLifecycle         bool
	SupportsServerEncrypt     bool
	SupportsCORS              bool
	SupportsBucketPolicy      bool
	SupportsPublicAccessBlock bool
	SupportsAccelerate        bool
	SupportsStorageClasses    []string
	DefaultEndpoint           string
	DefaultRegion             string
	RequiresCustomEndpoint    bool
	RecommendedPathStyle      bool
}

func (c ProviderCapability) SupportsFeature(f Feature) bool {
	switch f {
	case FeatureVersioning:
		return c.SupportsVersioning
	case FeatureObjectLock:
		return c.SupportsObjectLock
	case FeatureTagging:
		return c.SupportsTagging
	case FeatureMultipart:
		return c.SupportsMultipart
	case FeatureLifecycle:
		return c.SupportsLifecycle
	case FeatureServerEncryption:
		return c.SupportsServerEncrypt
	case FeatureCORS:
		return c.SupportsCORS
	case FeatureBucketPolicy:
		return c.SupportsBucketPolicy
	case FeaturePublicAccessBlock:
		return c.SupportsPublicAccessBlock
	default:
		return false
	}
}

func CapabilitiesForType(accType domain.AccountType) ProviderCapability {
	switch accType {
	case domain.TypeAWS:
		return ProviderCapability{
			SupportsVersioning:        true,
			SupportsObjectLock:        true,
			SupportsTagging:           true,
			SupportsMultipart:         true,
			SupportsLifecycle:         true,
			SupportsServerEncrypt:     true,
			SupportsCORS:              true,
			SupportsBucketPolicy:      true,
			SupportsPublicAccessBlock: true,
			SupportsAccelerate:        true,
			SupportsStorageClasses:    []string{"STANDARD", "STANDARD_IA", "ONEZONE_IA", "GLACIER", "DEEP_ARCHIVE", "INTELLIGENT_TIERING"},
			DefaultEndpoint:           "",
			DefaultRegion:             "us-east-1",
			RequiresCustomEndpoint:    false,
			RecommendedPathStyle:      false,
		}
	case domain.TypeMinIO:
		return ProviderCapability{
			SupportsVersioning:        true,
			SupportsObjectLock:        true,
			SupportsTagging:           true,
			SupportsMultipart:         true,
			SupportsLifecycle:         true,
			SupportsServerEncrypt:     true,
			SupportsCORS:              true,
			SupportsBucketPolicy:      true,
			SupportsPublicAccessBlock: false,
			SupportsAccelerate:        false,
			SupportsStorageClasses:    []string{"STANDARD"},
			DefaultEndpoint:           "http://localhost:9000",
			DefaultRegion:             "us-east-1",
			RequiresCustomEndpoint:    true,
			RecommendedPathStyle:      true,
		}
	case domain.TypeCloudflareR2:
		return ProviderCapability{
			SupportsVersioning:        false,
			SupportsObjectLock:        false,
			SupportsTagging:           false,
			SupportsMultipart:         true,
			SupportsLifecycle:         true,
			SupportsServerEncrypt:     false,
			SupportsCORS:              true,
			SupportsBucketPolicy:      false,
			SupportsPublicAccessBlock: false,
			SupportsAccelerate:        false,
			SupportsStorageClasses:    []string{"STANDARD"},
			DefaultEndpoint:           "",
			DefaultRegion:             "auto",
			RequiresCustomEndpoint:    true,
			RecommendedPathStyle:      false,
		}
	case domain.TypeWasabi:
		return ProviderCapability{
			SupportsVersioning:        true,
			SupportsObjectLock:        true,
			SupportsTagging:           true,
			SupportsMultipart:         true,
			SupportsLifecycle:         true,
			SupportsServerEncrypt:     true,
			SupportsCORS:              true,
			SupportsBucketPolicy:      true,
			SupportsPublicAccessBlock: false,
			SupportsAccelerate:        false,
			SupportsStorageClasses:    []string{"STANDARD"},
			DefaultEndpoint:           "https://s3.wasabisys.com",
			DefaultRegion:             "us-east-1",
			RequiresCustomEndpoint:    false,
			RecommendedPathStyle:      false,
		}
	case domain.TypeBackblazeB2:
		return ProviderCapability{
			SupportsVersioning:        true,
			SupportsObjectLock:        true,
			SupportsTagging:           false,
			SupportsMultipart:         true,
			SupportsLifecycle:         true,
			SupportsServerEncrypt:     true,
			SupportsCORS:              true,
			SupportsBucketPolicy:      false,
			SupportsPublicAccessBlock: false,
			SupportsAccelerate:        false,
			SupportsStorageClasses:    []string{"STANDARD"},
			DefaultEndpoint:           "",
			DefaultRegion:             "us-east-005",
			RequiresCustomEndpoint:    true,
			RecommendedPathStyle:      false,
		}
	case domain.TypeDigitalOceanSpaces:
		return ProviderCapability{
			SupportsVersioning:        false,
			SupportsObjectLock:        false,
			SupportsTagging:           false,
			SupportsMultipart:         true,
			SupportsLifecycle:         true,
			SupportsServerEncrypt:     false,
			SupportsCORS:              true,
			SupportsBucketPolicy:      false,
			SupportsPublicAccessBlock: false,
			SupportsAccelerate:        false,
			SupportsStorageClasses:    []string{"STANDARD"},
			DefaultEndpoint:           "https://nyc3.digitaloceanspaces.com",
			DefaultRegion:             "nyc3",
			RequiresCustomEndpoint:    false,
			RecommendedPathStyle:      true,
		}
	case domain.TypeCeph:
		return ProviderCapability{
			SupportsVersioning:        true,
			SupportsObjectLock:        false,
			SupportsTagging:           true,
			SupportsMultipart:         true,
			SupportsLifecycle:         true,
			SupportsServerEncrypt:     true,
			SupportsCORS:              true,
			SupportsBucketPolicy:      true,
			SupportsPublicAccessBlock: false,
			SupportsAccelerate:        false,
			SupportsStorageClasses:    []string{"STANDARD"},
			DefaultEndpoint:           "",
			DefaultRegion:             "default",
			RequiresCustomEndpoint:    true,
			RecommendedPathStyle:      true,
		}
	case domain.TypeCustomS3:
		fallthrough
	default:
		return ProviderCapability{
			SupportsVersioning:        true,
			SupportsObjectLock:        false,
			SupportsTagging:           true,
			SupportsMultipart:         true,
			SupportsLifecycle:         true,
			SupportsServerEncrypt:     false,
			SupportsCORS:              true,
			SupportsBucketPolicy:      false,
			SupportsPublicAccessBlock: false,
			SupportsAccelerate:        false,
			SupportsStorageClasses:    []string{"STANDARD"},
			DefaultEndpoint:           "",
			DefaultRegion:             "us-east-1",
			RequiresCustomEndpoint:    true,
			RecommendedPathStyle:      true,
		}
	}
}
