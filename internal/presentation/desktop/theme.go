package desktop

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// BlackTheme provides a sleek, modern, pure dark/black theme for Kumokura.
// Palette:
//   - Deep black backgrounds: #0a0a0c
//   - Modern dark elevated cards/menus/inputs: #121214, #18181b
//   - Crisp high-contrast white text: #f8fafc
//   - Subtle borders / separators: #27272a
//   - High-contrast primary accents: #38bdf8 (sky blue) or #6366f1 (indigo)
type BlackTheme struct{}

var _ fyne.Theme = (*BlackTheme)(nil)

func NewBlackTheme() *BlackTheme {
	return &BlackTheme{}
}

func (t *BlackTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		// Pure sleek deep black surface (#0a0a0c)
		return color.NRGBA{R: 0x0a, G: 0x0a, B: 0x0c, A: 0xff}

	case theme.ColorNameHeaderBackground, theme.ColorNameMenuBackground, theme.ColorNameOverlayBackground:
		// Slightly elevated dark surface (#121214)
		return color.NRGBA{R: 0x12, G: 0x12, B: 0x14, A: 0xff}

	case theme.ColorNameInputBackground:
		// Input surface (#18181b)
		return color.NRGBA{R: 0x18, G: 0x18, B: 0x1b, A: 0xff}

	case theme.ColorNameButton:
		// Crisp dark button surface (#202024)
		return color.NRGBA{R: 0x20, G: 0x20, B: 0x24, A: 0xff}

	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 0x16, G: 0x16, B: 0x1a, A: 0xff}

	case theme.ColorNameDisabled:
		// Muted gray for disabled text/icons (#71717a)
		return color.NRGBA{R: 0x71, G: 0x71, B: 0x7a, A: 0xff}

	case theme.ColorNameForeground:
		// Crisp high-contrast bright white text (#f8fafc)
		return color.NRGBA{R: 0xf8, G: 0xfa, B: 0xfc, A: 0xff}

	case theme.ColorNamePlaceHolder:
		// Legible placeholder gray (#94a3b8)
		return color.NRGBA{R: 0x94, G: 0xa3, B: 0xb8, A: 0xff}

	case theme.ColorNameInputBorder:
		// Subtle modern border (#3f3f46)
		return color.NRGBA{R: 0x3f, G: 0x3f, B: 0x46, A: 0xff}

	case theme.ColorNameSeparator:
		// Subtle clean separator line (#27272a)
		return color.NRGBA{R: 0x27, G: 0x27, B: 0x2a, A: 0xff}

	case theme.ColorNameHover:
		// Subtle translucent white hover state
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x18}

	case theme.ColorNamePressed:
		// Translucent pressed state
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0x28}

	case theme.ColorNamePrimary:
		// Modern vibrant sky blue accent (#38bdf8)
		return color.NRGBA{R: 0x38, G: 0xbd, B: 0xf8, A: 0xff}

	case theme.ColorNameForegroundOnPrimary:
		// Dark text on bright primary
		return color.NRGBA{R: 0x0a, G: 0x0a, B: 0x0c, A: 0xff}

	case theme.ColorNameFocus:
		// Focus highlight (#38bdf8 with alpha)
		return color.NRGBA{R: 0x38, G: 0xbd, B: 0xf8, A: 0x60}

	case theme.ColorNameSelection:
		// Table/list row selection highlight (#0284c7 with alpha)
		return color.NRGBA{R: 0x02, G: 0x84, B: 0xc7, A: 0x50}

	case theme.ColorNameSuccess:
		// Vibrant emerald green (#10b981)
		return color.NRGBA{R: 0x10, G: 0xb9, B: 0x81, A: 0xff}

	case theme.ColorNameForegroundOnSuccess:
		return color.NRGBA{R: 0x0a, G: 0x0a, B: 0x0c, A: 0xff}

	case theme.ColorNameWarning:
		// Vibrant amber (#f59e0b)
		return color.NRGBA{R: 0xf5, G: 0x9e, B: 0x0b, A: 0xff}

	case theme.ColorNameForegroundOnWarning:
		return color.NRGBA{R: 0x0a, G: 0x0a, B: 0x0c, A: 0xff}

	case theme.ColorNameError:
		// Vibrant rose red (#f43f5e)
		return color.NRGBA{R: 0xf4, G: 0x3f, B: 0x5e, A: 0xff}

	case theme.ColorNameForegroundOnError:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}

	case theme.ColorNameHyperlink:
		// Modern cyan-blue hyperlink (#60a5fa)
		return color.NRGBA{R: 0x60, G: 0xa5, B: 0xfa, A: 0xff}

	case theme.ColorNameScrollBar:
		// Semi-transparent scrollbar (#71717a with alpha)
		return color.NRGBA{R: 0x71, G: 0x71, B: 0x7a, A: 0x80}

	case theme.ColorNameScrollBarBackground:
		return color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x00}

	case theme.ColorNameShadow:
		return color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x99}

	case theme.ColorNameInnerWindowBorder:
		return color.NRGBA{R: 0x27, G: 0x27, B: 0x2a, A: 0xff}

	case theme.ColorNameInnerWindowBorderInactive:
		return color.NRGBA{R: 0x18, G: 0x18, B: 0x1b, A: 0xff}

	default:
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

func (t *BlackTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *BlackTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *BlackTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
