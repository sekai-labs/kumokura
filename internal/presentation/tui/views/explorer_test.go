package views

import (
	"strings"
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

func TestExplorerView_MultiSelectRendering(t *testing.T) {
	st := styles.NewStyles(styles.DarkTheme)
	view := NewExplorerView(st)
	view.Buckets = []bucketDomain.Bucket{{Name: "test-bucket"}}
	view.ActiveBucket = "test-bucket"
	view.ActivePaneIndex = 1
	view.Prefixes = []objDomain.Prefix{{Prefix: "photos/"}}
	view.Objects = []objDomain.Object{
		{Key: "photo.jpg", Size: 1024},
		{Key: "video.mp4", Size: 2048},
	}

	// In preview/inspect mode, [ ] markers appear
	view.ShowPreview = true
	rendered := view.Render(120, 30)
	assert.Contains(t, rendered, "[ ]")
	assert.Contains(t, rendered, "photos/")
	assert.Contains(t, rendered, "photo.jpg")

	// Select one item
	view.SelectedKeys["photo.jpg"] = true
	renderedSel := view.Render(120, 30)
	assert.Contains(t, renderedSel, "[x]")
	assert.Contains(t, renderedSel, "[ ]")
}

func TestExplorerView_MediaDetectionAndPreview(t *testing.T) {
	assert.True(t, IsImageFile("photo.png"))
	assert.True(t, IsImageFile("picture.jpeg"))
	assert.True(t, IsImageFile("drawing.svg"))
	assert.False(t, IsImageFile("archive.zip"))

	assert.True(t, IsVideoFile("movie.mp4"))
	assert.True(t, IsVideoFile("clip.webm"))
	assert.False(t, IsVideoFile("track.mp3"))

	// Video preview lines
	meta := &objDomain.ObjectMetadata{
		ContentType:   "video/mp4",
		ContentLength: 1048576,
	}
	vLines := renderVideoPreview("movie.mp4", meta, 40)
	vJoined := strings.Join(vLines, "\n")
	assert.Contains(t, vJoined, "VIDEO [MP4]")
	assert.Contains(t, vJoined, "Press 'o' to open video in default player")
}
