package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sekai-labs/kumokura/internal/accounts/domain"
	"github.com/sekai-labs/kumokura/internal/accounts/ports"
)

type AccountApplicationService struct {
	repo  ports.AccountRepository
	creds ports.CredentialStore
}

func NewAccountApplicationService(repo ports.AccountRepository, creds ports.CredentialStore) *AccountApplicationService {
	return &AccountApplicationService{
		repo:  repo,
		creds: creds,
	}
}

func generateID() domain.AccountID {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return domain.AccountID(hex.EncodeToString(b))
}

func (s *AccountApplicationService) CreateAccount(ctx context.Context, params ports.CreateAccountParams) (*domain.Account, error) {
	existing, err := s.repo.FindByName(ctx, params.Name)
	if err == nil && existing != nil {
		return nil, ports.ErrAccountConflict
	} else if err != nil && !errors.Is(err, ports.ErrAccountNotFound) {
		return nil, err
	}

	acc, err := domain.NewAccount(
		generateID(),
		params.Name,
		params.Type,
		params.Endpoint,
		params.Region,
		params.UsePathStyle,
	)
	if err != nil {
		return nil, err
	}

	if err := s.creds.Store(ctx, acc.ID, params.Credentials); err != nil {
		return nil, fmt.Errorf("store credentials: %w", err)
	}

	if err := s.repo.Save(ctx, acc); err != nil {
		_ = s.creds.Remove(ctx, acc.ID)
		return nil, fmt.Errorf("save account: %w", err)
	}

	return acc, nil
}

func (s *AccountApplicationService) UpdateAccount(ctx context.Context, params ports.UpdateAccountParams) (*domain.Account, error) {
	acc, err := s.repo.FindByID(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	if params.Name != "" && params.Name != acc.Name {
		existing, err := s.repo.FindByName(ctx, params.Name)
		if err == nil && existing != nil && existing.ID != acc.ID {
			return nil, ports.ErrAccountConflict
		}
		acc.Name = params.Name
	}

	if params.Endpoint != "" || acc.Type == domain.TypeMinIO || acc.Type == domain.TypeCustomS3 {
		acc.Endpoint = params.Endpoint
	}
	if params.Region != "" {
		acc.Region = params.Region
	}
	acc.UsePathStyle = params.UsePathStyle
	acc.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, acc); err != nil {
		return nil, fmt.Errorf("update account: %w", err)
	}

	if params.Credentials != nil {
		if err := s.creds.Store(ctx, acc.ID, *params.Credentials); err != nil {
			return nil, fmt.Errorf("update credentials: %w", err)
		}
	}

	return acc, nil
}

func (s *AccountApplicationService) DeleteAccount(ctx context.Context, id domain.AccountID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.creds.Remove(ctx, id)
	return nil
}

func (s *AccountApplicationService) GetAccount(ctx context.Context, id domain.AccountID) (*domain.Account, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *AccountApplicationService) ListAccounts(ctx context.Context) ([]*domain.Account, error) {
	return s.repo.FindAll(ctx)
}

func (s *AccountApplicationService) GetCredentials(ctx context.Context, id domain.AccountID) (domain.Credentials, error) {
	return s.creds.Retrieve(ctx, id)
}

func (s *AccountApplicationService) TestConnection(ctx context.Context, id domain.AccountID) error {
	acc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	creds, err := s.creds.Retrieve(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to retrieve account credentials: %w", err)
	}

	optFns := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(acc.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			creds.AccessKeyID,
			creds.SecretAccessKey,
			creds.SessionToken,
		)),
		awsconfig.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return fmt.Errorf("load aws config: %w", err)
	}

	s3OptFns := []func(*s3.Options){
		func(o *s3.Options) {
			o.UsePathStyle = acc.UsePathStyle
		},
	}
	if acc.Endpoint != "" {
		s3OptFns = append(s3OptFns, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(acc.Endpoint)
		})
	}

	client := s3.NewFromConfig(cfg, s3OptFns...)
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err = client.ListBuckets(timeoutCtx, &s3.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("s3 list buckets failed: %w", err)
	}

	return nil
}
