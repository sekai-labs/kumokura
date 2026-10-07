package s3client_test

import (
	"context"
	"testing"

	accountdomain "github.com/sekai-labs/kumokura/internal/accounts/domain"
	"github.com/sekai-labs/kumokura/internal/providers/adapters/s3client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientFactoryBuild(t *testing.T) {
	factory, err := s3client.NewClientFactory(nil)
	require.NoError(t, err)
	require.NotNil(t, factory)

	acc, err := accountdomain.NewAccount(
		"acc-1",
		"MinIO Local",
		accountdomain.TypeMinIO,
		"http://localhost:9000",
		"us-east-1",
		true,
	)
	require.NoError(t, err)

	creds, err := accountdomain.NewCredentials("minioadmin", "minioadmin", "")
	require.NoError(t, err)

	ctx := context.Background()
	client, err := factory.Build(ctx, acc, creds)
	require.NoError(t, err)
	assert.NotNil(t, client)

	_, err = factory.Build(ctx, nil, creds)
	assert.Error(t, err)
}
