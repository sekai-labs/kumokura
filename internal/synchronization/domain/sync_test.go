package domain_test

import (
	"testing"
	"time"

	"github.com/sekai-labs/kumokura/internal/synchronization/domain"
	"github.com/stretchr/testify/assert"
)

func TestFilterIncludesAndExcludes(t *testing.T) {
	filter := domain.Filter{
		Excludes: []string{
			"*.log",
			".git/**",
			"node_modules/**",
		},
	}

	assert.False(t, filter.Matches("app.log"))
	assert.False(t, filter.Matches("logs/app.log"))
	assert.False(t, filter.Matches(".git/config"))
	assert.False(t, filter.Matches(".git/objects/1a/2b"))
	assert.False(t, filter.Matches("node_modules/react/index.js"))
	assert.True(t, filter.Matches("src/index.ts"))
	assert.True(t, filter.Matches("README.md"))

	filterWithInclude := domain.Filter{
		Includes: []string{
			"*.jpg",
			"*.png",
			"photos/**",
		},
	}

	assert.True(t, filterWithInclude.Matches("photo.jpg"))
	assert.True(t, filterWithInclude.Matches("image.png"))
	assert.True(t, filterWithInclude.Matches("photos/2026/trip.mov"))
	assert.False(t, filterWithInclude.Matches("document.pdf"))
}

func TestETagAndMultipartHelpers(t *testing.T) {
	etag1 := "\"1b2cf535f27731c974343645a3985328\""
	assert.Equal(t, "1b2cf535f27731c974343645a3985328", domain.NormalizeETag(etag1))
	assert.False(t, domain.IsMultipartETag(etag1))

	etagMP := "\"1b2cf535f27731c974343645a3985328-14\""
	assert.Equal(t, "1b2cf535f27731c974343645a3985328-14", domain.NormalizeETag(etagMP))
	assert.True(t, domain.IsMultipartETag(etagMP))
}

func TestTimestampTolerance(t *testing.T) {
	now := time.Now()
	t1 := now
	t2 := now.Add(1500 * time.Millisecond)

	assert.True(t, domain.TimestampsMatchWithTolerance(t1, t2, 2*time.Second))
	assert.False(t, domain.TimestampsMatchWithTolerance(t1, t2, 1*time.Second))
}

func TestValidateSyncPath(t *testing.T) {
	assert.NoError(t, domain.ValidateSyncPath("folder/file.txt"))
	assert.NoError(t, domain.ValidateSyncPath("file.txt"))
	assert.Error(t, domain.ValidateSyncPath("../file.txt"))
	assert.Error(t, domain.ValidateSyncPath("folder/../../file.txt"))
	assert.Error(t, domain.ValidateSyncPath("folder/\x00/file.txt"))
}
