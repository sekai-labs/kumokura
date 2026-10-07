package domain_test

import (
	"testing"

	accountdomain "github.com/sekai-labs/kumokura/internal/accounts/domain"
	"github.com/sekai-labs/kumokura/internal/providers/domain"
	"github.com/stretchr/testify/assert"
)

func TestCapabilitiesForType(t *testing.T) {
	awsCaps := domain.CapabilitiesForType(accountdomain.TypeAWS)
	assert.True(t, awsCaps.SupportsVersioning)
	assert.True(t, awsCaps.SupportsObjectLock)
	assert.True(t, awsCaps.SupportsTagging)
	assert.False(t, awsCaps.RequiresCustomEndpoint)
	assert.False(t, awsCaps.RecommendedPathStyle)
	assert.Contains(t, awsCaps.SupportsStorageClasses, "GLACIER")

	minioCaps := domain.CapabilitiesForType(accountdomain.TypeMinIO)
	assert.True(t, minioCaps.SupportsVersioning)
	assert.True(t, minioCaps.SupportsObjectLock)
	assert.True(t, minioCaps.RequiresCustomEndpoint)
	assert.True(t, minioCaps.RecommendedPathStyle)
	assert.Equal(t, "http://localhost:9000", minioCaps.DefaultEndpoint)

	r2Caps := domain.CapabilitiesForType(accountdomain.TypeCloudflareR2)
	assert.False(t, r2Caps.SupportsVersioning)
	assert.False(t, r2Caps.SupportsTagging)
	assert.True(t, r2Caps.RequiresCustomEndpoint)
	assert.False(t, r2Caps.RecommendedPathStyle)

	cephCaps := domain.CapabilitiesForType(accountdomain.TypeCeph)
	assert.True(t, cephCaps.RequiresCustomEndpoint)
	assert.True(t, cephCaps.RecommendedPathStyle)

	spacesCaps := domain.CapabilitiesForType(accountdomain.TypeDigitalOceanSpaces)
	assert.False(t, spacesCaps.RequiresCustomEndpoint)
	assert.True(t, spacesCaps.RecommendedPathStyle)
}
