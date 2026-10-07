package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidAccountID   = errors.New("invalid account id")
	ErrInvalidAccountName = errors.New("invalid account name: must not be empty or exceed 64 characters")
	ErrUnsupportedType    = errors.New("unsupported account type")
	ErrInvalidCredentials = errors.New("invalid credentials: access key and secret key are required")
	ErrInvalidEndpoint    = errors.New("invalid endpoint: required for custom or self-hosted s3 providers")
)

type AccountID string

func (id AccountID) String() string {
	return string(id)
}

type AccountType string

const (
	TypeAWS                 AccountType = "AWS"
	TypeMinIO               AccountType = "MinIO"
	TypeCloudflareR2        AccountType = "CloudflareR2"
	TypeWasabi              AccountType = "Wasabi"
	TypeBackblazeB2         AccountType = "BackblazeB2"
	TypeDigitalOceanSpaces  AccountType = "DigitalOceanSpaces"
	TypeCeph                AccountType = "Ceph"
	TypeCustomS3            AccountType = "CustomS3"
)

func (t AccountType) IsValid() bool {
	switch t {
	case TypeAWS, TypeMinIO, TypeCloudflareR2, TypeWasabi,
		TypeBackblazeB2, TypeDigitalOceanSpaces, TypeCeph, TypeCustomS3:
		return true
	default:
		return false
	}
}

type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

func NewCredentials(accessKeyID, secretAccessKey, sessionToken string) (Credentials, error) {
	ak := strings.TrimSpace(accessKeyID)
	sk := strings.TrimSpace(secretAccessKey)
	if ak == "" || sk == "" {
		return Credentials{}, ErrInvalidCredentials
	}
	return Credentials{
		AccessKeyID:     ak,
		SecretAccessKey: sk,
		SessionToken:    strings.TrimSpace(sessionToken),
	}, nil
}

type Account struct {
	ID           AccountID
	Name         string
	Type         AccountType
	Endpoint     string
	Region       string
	UsePathStyle bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewAccount(id AccountID, name string, accType AccountType, endpoint, region string, usePathStyle bool) (*Account, error) {
	if strings.TrimSpace(string(id)) == "" {
		return nil, ErrInvalidAccountID
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" || len(trimmedName) > 64 {
		return nil, ErrInvalidAccountName
	}
	if !accType.IsValid() {
		return nil, ErrUnsupportedType
	}

	trimmedEndpoint := strings.TrimSpace(endpoint)
	if (accType == TypeMinIO || accType == TypeCeph || accType == TypeCustomS3) && trimmedEndpoint == "" {
		return nil, ErrInvalidEndpoint
	}

	trimmedRegion := strings.TrimSpace(region)
	if trimmedRegion == "" {
		trimmedRegion = "us-east-1"
	}

	now := time.Now().UTC()
	return &Account{
		ID:           id,
		Name:         trimmedName,
		Type:         accType,
		Endpoint:     trimmedEndpoint,
		Region:       trimmedRegion,
		UsePathStyle: usePathStyle,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}
