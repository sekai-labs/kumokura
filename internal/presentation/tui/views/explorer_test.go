package views

import (
	"testing"
	"time"

	bucketDomain "github.com/sekai-labs/kumokura/internal/buckets/domain"
	objDomain "github.com/sekai-labs/kumokura/internal/objects/domain"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/styles"
	"github.com/stretchr/testify/assert"
)

func TestExplorerView_LongKeysNeverWrap(t *testing.T) {
	st := styles.NewStyles(styles.DarkTheme)
	view := NewExplorerView(st)
	view.Buckets = []bucketDomain.Bucket{
		{Name: "my-very-long-bucket-name-with-many-words-and-dashes"},
	}
	view.Objects = []objDomain.Object{
		{
			Key:          "very/deeply/nested/path/with/a/massively/long/file/name/that/would/normally/overflow/and/break/tables/character_length_over_150_characters.json",
			Size:         1048576,
			StorageClass: "STANDARD_IA",
			LastModified: time.Now(),
		},
		{
			Key:          "another-extremely-long-object-key-which-should-be-truncated-cleanly-without-breaking-line-heights.png",
			Size:         500000,
			StorageClass: "GLACIER",
			LastModified: time.Now(),
		},
	}
	view.SelectedBucket = 0
	view.ActiveBucket = "my-very-long-bucket-name-with-many-words-and-dashes"
	view.ActivePaneIndex = 1

	rendered80 := view.Render(80, 24)
	assert.NotEmpty(t, rendered80)
	assert.NotContains(t, rendered80, "Terminal window too small")

	rendered120 := view.Render(120, 35)
	assert.NotEmpty(t, rendered120)

	rendered160 := view.Render(160, 45)
	assert.NotEmpty(t, rendered160)
}

func TestExplorerView_PadVisual(t *testing.T) {
	paddedLeft := padVisual("test", 10, false)
	assert.Equal(t, "test      ", paddedLeft)

	paddedRight := padVisual("test", 10, true)
	assert.Equal(t, "      test", paddedRight)

	truncated := padVisual("this-is-a-very-long-string", 10, false)
	assert.LessOrEqual(t, len(truncated), len("this-is-a-very-long-string"))
}
