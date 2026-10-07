package domain_test

import (
	"strings"
	"testing"

	"github.com/sekai-labs/kumokura/internal/accounts/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountValidation(t *testing.T) {
	_, err := domain.NewAccount("", "My AWS", domain.TypeAWS, "", "us-east-1", false)
	assert.ErrorIs(t, err, domain.ErrInvalidAccountID)

	_, err = domain.NewAccount("acc1", "", domain.TypeAWS, "", "us-east-1", false)
	assert.ErrorIs(t, err, domain.ErrInvalidAccountName)

	longName := strings.Repeat("a", 65)
	_, err = domain.NewAccount("acc1", longName, domain.TypeAWS, "", "us-east-1", false)
	assert.ErrorIs(t, err, domain.ErrInvalidAccountName)

	_, err = domain.NewAccount("acc1", "Valid Name", domain.AccountType("Unknown"), "", "us-east-1", false)
	assert.ErrorIs(t, err, domain.ErrUnsupportedType)

	_, err = domain.NewAccount("acc1", "MinIO Local", domain.TypeMinIO, "", "us-east-1", false)
	assert.ErrorIs(t, err, domain.ErrInvalidEndpoint)

	_, err = domain.NewAccount("acc1", "Ceph Cluster", domain.TypeCeph, "", "us-east-1", false)
	assert.ErrorIs(t, err, domain.ErrInvalidEndpoint)

	acc, err := domain.NewAccount("acc1", "AWS Prod", domain.TypeAWS, "", "", false)
	require.NoError(t, err)
	assert.Equal(t, "us-east-1", acc.Region)
	assert.Equal(t, domain.AccountID("acc1"), acc.ID)
	assert.Equal(t, "acc1", acc.ID.String())
	assert.False(t, acc.CreatedAt.IsZero())
	assert.False(t, acc.UpdatedAt.IsZero())

	minioAcc, err := domain.NewAccount("acc2", "Local MinIO", domain.TypeMinIO, "http://localhost:9000", "us-east-1", true)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:9000", minioAcc.Endpoint)
	assert.True(t, minioAcc.UsePathStyle)
}

func TestCredentialsValidation(t *testing.T) {
	_, err := domain.NewCredentials("", "secret", "")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)

	_, err = domain.NewCredentials("access", "", "")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)

	creds, err := domain.NewCredentials("  access-key  ", "  secret-key  ", "  session-token  ")
	require.NoError(t, err)
	assert.Equal(t, "access-key", creds.AccessKeyID)
	assert.Equal(t, "secret-key", creds.SecretAccessKey)
	assert.Equal(t, "session-token", creds.SessionToken)
}
