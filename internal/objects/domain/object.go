package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrEmptyObjectKey = errors.New("object key cannot be empty")
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

type ChecksumAlgorithm string

const (
	ChecksumAlgorithmCRC32  ChecksumAlgorithm = "CRC32"
	ChecksumAlgorithmCRC32C ChecksumAlgorithm = "CRC32C"
	ChecksumAlgorithmSHA1   ChecksumAlgorithm = "SHA1"
	ChecksumAlgorithmSHA256 ChecksumAlgorithm = "SHA256"
)

type ObjectChecksum struct {
	Algorithm ChecksumAlgorithm
	Value     string
}

type LegalHoldStatus string

const (
	LegalHoldStatusOn  LegalHoldStatus = "ON"
	LegalHoldStatusOff LegalHoldStatus = "OFF"
)

type RetentionMode string

const (
	RetentionModeGovernance RetentionMode = "GOVERNANCE"
	RetentionModeCompliance RetentionMode = "COMPLIANCE"
)

type Retention struct {
	Mode            RetentionMode
	RetainUntilDate time.Time
}

type ObjectMetadata struct {
	ContentType     string
	ContentLength   int64
	ETag            string
	LastModified    time.Time
	StorageClass    StorageClass
	UserMetadata    map[string]string
	VersionID       string
	IsDeleteMarker  bool
	CacheControl    string
	ContentEncoding string
	Checksum        *ObjectChecksum
}

type Object struct {
	Bucket       string
	Key          string
	Size         int64
	ETag         string
	LastModified time.Time
	StorageClass StorageClass
	IsPrefix     bool
	VersionID    string
}

func ValidateObjectKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return ErrEmptyObjectKey
	}
	return nil
}

type ObjectVersion struct {
	Bucket       string
	Key          string
	VersionID    string
	IsLatest     bool
	IsDelete     bool
	LastModified time.Time
	Size         int64
	ETag         string
}

type Prefix struct {
	Bucket string
	Prefix string
}

type ObjectFilter struct {
	Prefix       string
	Delimiter    string
	MaxKeys      int32
	Continuation string
}

type PresignedURL struct {
	URL        string
	Method     string
	Expiration time.Time
}

type ObjectTag struct {
	Key   string
	Value string
}

type RestoreRequest struct {
	Days                 int32
	GlacierJobTier       string
	SelectParametersJSON string
}
