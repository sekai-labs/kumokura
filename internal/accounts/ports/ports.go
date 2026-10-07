package ports

import (
	"context"
	"errors"

	"github.com/sekai-labs/kumokura/internal/accounts/domain"
)

var (
	ErrAccountNotFound = errors.New("account not found")
	ErrAccountConflict = errors.New("account with this name already exists")
)

type AccountRepository interface {
	Save(ctx context.Context, account *domain.Account) error
	FindByID(ctx context.Context, id domain.AccountID) (*domain.Account, error)
	FindByName(ctx context.Context, name string) (*domain.Account, error)
	FindAll(ctx context.Context) ([]*domain.Account, error)
	Update(ctx context.Context, account *domain.Account) error
	Delete(ctx context.Context, id domain.AccountID) error
}

type CredentialStore interface {
	Store(ctx context.Context, accountID domain.AccountID, creds domain.Credentials) error
	Retrieve(ctx context.Context, accountID domain.AccountID) (domain.Credentials, error)
	Remove(ctx context.Context, accountID domain.AccountID) error
}

type AccountService interface {
	CreateAccount(ctx context.Context, params CreateAccountParams) (*domain.Account, error)
	UpdateAccount(ctx context.Context, params UpdateAccountParams) (*domain.Account, error)
	DeleteAccount(ctx context.Context, id domain.AccountID) error
	GetAccount(ctx context.Context, id domain.AccountID) (*domain.Account, error)
	ListAccounts(ctx context.Context) ([]*domain.Account, error)
	GetCredentials(ctx context.Context, id domain.AccountID) (domain.Credentials, error)
	TestConnection(ctx context.Context, id domain.AccountID) error
}

type CreateAccountParams struct {
	Name         string
	Type         domain.AccountType
	Endpoint     string
	Region       string
	UsePathStyle bool
	Credentials  domain.Credentials
}

type UpdateAccountParams struct {
	ID           domain.AccountID
	Name         string
	Endpoint     string
	Region       string
	UsePathStyle bool
	Credentials  *domain.Credentials
}
