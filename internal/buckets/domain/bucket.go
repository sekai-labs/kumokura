package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

var (
	ErrInvalidBucketName = errors.New("invalid bucket name")
	bucketNameRegex      = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)
)

type VersioningStatus string

const (
	VersioningStatusDisabled  VersioningStatus = "Disabled"
	VersioningStatusEnabled   VersioningStatus = "Enabled"
	VersioningStatusSuspended VersioningStatus = "Suspended"
)

type StorageClass string

const (
	StorageClassStandard           StorageClass = "STANDARD"
	StorageClassReducedRedundancy  StorageClass = "REDUCED_REDUNDANCY"
	StorageClassStandardIA         StorageClass = "STANDARD_IA"
	StorageClassOneZoneIA          StorageClass = "ONEZONE_IA"
	StorageClassIntelligentTiering StorageClass = "INTELLIGENT_TIERING"
	StorageClassGlacier            StorageClass = "GLACIER"
	StorageClassDeepArchive        StorageClass = "DEEP_ARCHIVE"
	StorageClassOutposts           StorageClass = "OUTPOSTS"
	StorageClassGlacierIR          StorageClass = "GLACIER_IR"
	StorageClassSnow               StorageClass = "SNOW"
	StorageClassExpressOneZone     StorageClass = "EXPRESS_ONEZONE"
)

type Bucket struct {
	Name         string
	Region       string
	CreationDate time.Time
}

func ValidateBucketName(name string) error {
	if len(name) < 3 || len(name) > 63 {
		return ErrInvalidBucketName
	}
	if !bucketNameRegex.MatchString(name) {
		return ErrInvalidBucketName
	}
	if strings.Contains(name, "..") {
		return ErrInvalidBucketName
	}
	if strings.Contains(name, ".-") || strings.Contains(name, "-.") {
		return ErrInvalidBucketName
	}
	return nil
}

func NewBucket(name string, region string, creationDate time.Time) (Bucket, error) {
	if err := ValidateBucketName(name); err != nil {
		return Bucket{}, err
	}
	return Bucket{
		Name:         name,
		Region:       region,
		CreationDate: creationDate,
	}, nil
}

type VersioningConfig struct {
	Status    VersioningStatus
	MFADelete bool
}

type LifecycleTransition struct {
	Days         int32
	Date         *time.Time
	StorageClass StorageClass
}

type LifecycleExpiration struct {
	Days                      int32
	Date                      *time.Time
	ExpiredObjectDeleteMarker bool
}

type NoncurrentVersionExpiration struct {
	NoncurrentDays int32
	NewerVersions  int32
}

type NoncurrentVersionTransition struct {
	NoncurrentDays int32
	StorageClass   StorageClass
	NewerVersions  int32
}

type LifecycleRule struct {
	ID                           string
	Status                       string
	Prefix                       string
	Transitions                  []LifecycleTransition
	Expiration                   *LifecycleExpiration
	NoncurrentVersionExpiration  *NoncurrentVersionExpiration
	NoncurrentVersionTransitions []NoncurrentVersionTransition
}

type EncryptionConfig struct {
	SSEAlgorithm   string
	KMSMasterKeyID string
	BucketKey      bool
}

type PublicAccessBlock struct {
	BlockPublicAcls       bool
	IgnorePublicAcls      bool
	BlockPublicPolicy     bool
	RestrictPublicBuckets bool
}

type CORSRule struct {
	AllowedHeaders []string
	AllowedMethods []string
	AllowedOrigins []string
	ExposeHeaders  []string
	MaxAgeSeconds  int32
}

type Tag struct {
	Key   string
	Value string
}

type TagSet []Tag
