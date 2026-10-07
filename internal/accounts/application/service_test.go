package application_test

import (
	"context"
	"testing"

	"github.com/sekai-labs/kumokura/internal/accounts/application"
	"github.com/sekai-labs/kumokura/internal/accounts/domain"
	"github.com/sekai-labs/kumokura/internal/accounts/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memoryAccountRepo struct {
	accounts map[domain.AccountID]*domain.Account
}

func newMemoryAccountRepo() *memoryAccountRepo {
	return &memoryAccountRepo{
		accounts: make(map[domain.AccountID]*domain.Account),
	}
}

func (m *memoryAccountRepo) Save(ctx context.Context, account *domain.Account) error {
	m.accounts[account.ID] = account
	return nil
}

func (m *memoryAccountRepo) FindByID(ctx context.Context, id domain.AccountID) (*domain.Account, error) {
	acc, ok := m.accounts[id]
	if !ok {
		return nil, ports.ErrAccountNotFound
	}
	return acc, nil
}

func (m *memoryAccountRepo) FindByName(ctx context.Context, name string) (*domain.Account, error) {
	for _, acc := range m.accounts {
		if acc.Name == name {
			return acc, nil
		}
	}
	return nil, ports.ErrAccountNotFound
}

func (m *memoryAccountRepo) FindAll(ctx context.Context) ([]*domain.Account, error) {
	var list []*domain.Account
	for _, acc := range m.accounts {
		list = append(list, acc)
	}
	return list, nil
}

func (m *memoryAccountRepo) Update(ctx context.Context, account *domain.Account) error {
	if _, ok := m.accounts[account.ID]; !ok {
		return ports.ErrAccountNotFound
	}
	m.accounts[account.ID] = account
	return nil
}

func (m *memoryAccountRepo) Delete(ctx context.Context, id domain.AccountID) error {
	if _, ok := m.accounts[id]; !ok {
		return ports.ErrAccountNotFound
	}
	delete(m.accounts, id)
	return nil
}

type memoryCredStore struct {
	creds map[domain.AccountID]domain.Credentials
}

func newMemoryCredStore() *memoryCredStore {
	return &memoryCredStore{
		creds: make(map[domain.AccountID]domain.Credentials),
	}
}

func (m *memoryCredStore) Store(ctx context.Context, accountID domain.AccountID, creds domain.Credentials) error {
	m.creds[accountID] = creds
	return nil
}

func (m *memoryCredStore) Retrieve(ctx context.Context, accountID domain.AccountID) (domain.Credentials, error) {
	c, ok := m.creds[accountID]
	if !ok {
		return domain.Credentials{}, ports.ErrAccountNotFound
	}
	return c, nil
}

func (m *memoryCredStore) Remove(ctx context.Context, accountID domain.AccountID) error {
	delete(m.creds, accountID)
	return nil
}

func TestAccountApplicationService(t *testing.T) {
	repo := newMemoryAccountRepo()
	credStore := newMemoryCredStore()
	svc := application.NewAccountApplicationService(repo, credStore)
	ctx := context.Background()

	creds, err := domain.NewCredentials("access", "secret", "")
	require.NoError(t, err)

	acc, err := svc.CreateAccount(ctx, ports.CreateAccountParams{
		Name:         "Prod AWS",
		Type:         domain.TypeAWS,
		Region:       "eu-west-1",
		UsePathStyle: false,
		Credentials:  creds,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, acc.ID)

	_, err = svc.CreateAccount(ctx, ports.CreateAccountParams{
		Name:        "Prod AWS",
		Type:        domain.TypeAWS,
		Credentials: creds,
	})
	assert.ErrorIs(t, err, ports.ErrAccountConflict)

	found, err := svc.GetAccount(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, acc.Name, found.Name)

	retrievedCreds, err := svc.GetCredentials(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, "access", retrievedCreds.AccessKeyID)

	updated, err := svc.UpdateAccount(ctx, ports.UpdateAccountParams{
		ID:     acc.ID,
		Name:   "Renamed Prod AWS",
		Region: "us-west-2",
	})
	require.NoError(t, err)
	assert.Equal(t, "Renamed Prod AWS", updated.Name)
	assert.Equal(t, "us-west-2", updated.Region)

	list, err := svc.ListAccounts(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	err = svc.DeleteAccount(ctx, acc.ID)
	require.NoError(t, err)

	_, err = svc.GetAccount(ctx, acc.ID)
	assert.ErrorIs(t, err, ports.ErrAccountNotFound)

	_, err = svc.GetCredentials(ctx, acc.ID)
	assert.ErrorIs(t, err, ports.ErrAccountNotFound)
}
