package application_test

import (
	"context"
	"testing"

	accountdomain "github.com/sekai-labs/kumokura/internal/accounts/domain"
	"github.com/sekai-labs/kumokura/internal/providers/application"
	providerdomain "github.com/sekai-labs/kumokura/internal/providers/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderApplicationService(t *testing.T) {
	svc := application.NewProviderApplicationService()
	ctx := context.Background()

	caps, err := svc.GetCapabilities(ctx, accountdomain.TypeAWS)
	require.NoError(t, err)
	assert.True(t, caps.SupportsVersioning)

	_, err = svc.GetCapabilities(ctx, accountdomain.AccountType("invalid"))
	assert.ErrorIs(t, err, accountdomain.ErrUnsupportedType)

	types := svc.ListSupportedTypes(ctx)
	assert.Contains(t, types, accountdomain.TypeAWS)
	assert.Contains(t, types, accountdomain.TypeMinIO)
	assert.Contains(t, types, accountdomain.TypeCloudflareR2)

	canLock, err := svc.CheckCapability(ctx, accountdomain.TypeAWS, providerdomain.FeatureObjectLock)
	require.NoError(t, err)
	assert.True(t, canLock)

	r2Lock, err := svc.CheckCapability(ctx, accountdomain.TypeCloudflareR2, providerdomain.FeatureObjectLock)
	require.NoError(t, err)
	assert.False(t, r2Lock)
}
