package desktop

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlackTheme(t *testing.T) {
	th := NewBlackTheme()
	require.NotNil(t, th)

	bg := th.Color(theme.ColorNameBackground, theme.VariantDark)
	r, g, b, a := bg.RGBA()
	assert.Equal(t, uint32(0xffff), a)
	assert.True(t, r < 0x2000, "Background red should be very dark")
	assert.True(t, g < 0x2000, "Background green should be very dark")
	assert.True(t, b < 0x2000, "Background blue should be very dark")

	fg := th.Color(theme.ColorNameForeground, theme.VariantDark)
	fr, fg_, fb, fa := fg.RGBA()
	assert.Equal(t, uint32(0xffff), fa)
	assert.True(t, fr > 0xf000, "Foreground text should be bright white")
	assert.True(t, fg_ > 0xf000, "Foreground text should be bright white")
	assert.True(t, fb > 0xf000, "Foreground text should be bright white")

	pri := th.Color(theme.ColorNamePrimary, theme.VariantDark)
	assert.NotNil(t, pri)
	inp := th.Color(theme.ColorNameInputBackground, theme.VariantDark)
	assert.NotNil(t, inp)

	assert.NotNil(t, th.Font(fyne.TextStyle{}))
	assert.NotNil(t, th.Icon(theme.IconNameCancel))
	assert.True(t, th.Size(theme.SizeNameText) > 0)
}
