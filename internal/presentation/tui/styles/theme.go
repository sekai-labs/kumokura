package styles

import (
	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	BaseBackground  lipgloss.Color
	CardBackground  lipgloss.Color
	HighlightRow    lipgloss.Color
	BorderActive    lipgloss.Color
	BorderInactive  lipgloss.Color
	TextPrimary     lipgloss.Color
	TextMuted       lipgloss.Color
	TextSubtle      lipgloss.Color
	AccentSky       lipgloss.Color
	AccentCyan      lipgloss.Color
	BadgeSuccess    lipgloss.Color
	BadgeWarning    lipgloss.Color
	BadgeDanger     lipgloss.Color
	ModalBackground lipgloss.Color
	ModalBorder     lipgloss.Color
}

var DarkTheme = Theme{
	BaseBackground:  lipgloss.Color("#0F172A"),
	CardBackground:  lipgloss.Color("#1E293B"),
	HighlightRow:    lipgloss.Color("#334155"),
	BorderActive:    lipgloss.Color("#06B6D4"),
	BorderInactive:  lipgloss.Color("#374151"),
	TextPrimary:     lipgloss.Color("#F8FAFC"),
	TextMuted:       lipgloss.Color("#94A3B8"),
	TextSubtle:      lipgloss.Color("#64748B"),
	AccentSky:       lipgloss.Color("#38BDF8"),
	AccentCyan:      lipgloss.Color("#06B6D4"),
	BadgeSuccess:    lipgloss.Color("#4ADE80"),
	BadgeWarning:    lipgloss.Color("#FBBF24"),
	BadgeDanger:     lipgloss.Color("#F87171"),
	ModalBackground: lipgloss.Color("#1E293B"),
	ModalBorder:     lipgloss.Color("#818CF8"),
}

var LightTheme = Theme{
	BaseBackground:  lipgloss.Color("#F8FAFC"),
	CardBackground:  lipgloss.Color("#FFFFFF"),
	HighlightRow:    lipgloss.Color("#E2E8F0"),
	BorderActive:    lipgloss.Color("#0284C7"),
	BorderInactive:  lipgloss.Color("#CBD5E1"),
	TextPrimary:     lipgloss.Color("#0F172A"),
	TextMuted:       lipgloss.Color("#475569"),
	TextSubtle:      lipgloss.Color("#64748B"),
	AccentSky:       lipgloss.Color("#0284C7"),
	AccentCyan:      lipgloss.Color("#0891B2"),
	BadgeSuccess:    lipgloss.Color("#16A34A"),
	BadgeWarning:    lipgloss.Color("#D97706"),
	BadgeDanger:     lipgloss.Color("#DC2626"),
	ModalBackground: lipgloss.Color("#FFFFFF"),
	ModalBorder:     lipgloss.Color("#6366F1"),
}

type Styles struct {
	Theme Theme

	TopBar        lipgloss.Style
	ActiveTab     lipgloss.Style
	InactiveTab   lipgloss.Style
	BadgeAccount  lipgloss.Style
	ActivePanel   lipgloss.Style
	InactivePanel lipgloss.Style
	PanelTitle    lipgloss.Style
	SelectedRow   lipgloss.Style
	NormalRow     lipgloss.Style
	StatusBar     lipgloss.Style
	StatusKey     lipgloss.Style
	StatusDesc    lipgloss.Style
	ModalBox      lipgloss.Style
	ModalHeader   lipgloss.Style
	ModalBody     lipgloss.Style
	SuccessPill   lipgloss.Style
	WarningPill   lipgloss.Style
	DangerPill    lipgloss.Style
}

func NewStyles(t Theme) Styles {
	return Styles{
		Theme: t,

		TopBar: lipgloss.NewStyle().
			Background(t.CardBackground).
			Foreground(t.TextPrimary).
			Padding(0, 1),

		ActiveTab: lipgloss.NewStyle().
			Bold(true).
			Background(t.BorderActive).
			Foreground(t.TextPrimary).
			Padding(0, 1),

		InactiveTab: lipgloss.NewStyle().
			Background(t.CardBackground).
			Foreground(t.TextMuted).
			Padding(0, 1),

		BadgeAccount: lipgloss.NewStyle().
			Bold(true).
			Background(t.HighlightRow).
			Foreground(t.AccentSky).
			Padding(0, 1),

		ActivePanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderActive),

		InactivePanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderInactive),

		PanelTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.AccentSky).
			Padding(0, 1),

		SelectedRow: lipgloss.NewStyle().
			Bold(true).
			Background(t.HighlightRow).
			Foreground(t.TextPrimary),

		NormalRow: lipgloss.NewStyle().
			Foreground(t.TextPrimary),

		StatusBar: lipgloss.NewStyle().
			Background(t.CardBackground).
			Foreground(t.TextMuted).
			Padding(0, 1),

		StatusKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.AccentSky),

		StatusDesc: lipgloss.NewStyle().
			Foreground(t.TextSubtle),

		ModalBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.ModalBorder).
			Background(t.ModalBackground).
			Padding(1, 2),

		ModalHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.TextPrimary).
			MarginBottom(1),

		ModalBody: lipgloss.NewStyle().
			Foreground(t.TextMuted),

		SuccessPill: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.BadgeSuccess),

		WarningPill: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.BadgeWarning),

		DangerPill: lipgloss.NewStyle().
			Bold(true).
			Foreground(t.BadgeDanger),
	}
}
