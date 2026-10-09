package keymap

import (
	"github.com/charmbracelet/bubbles/key"
)

type KeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Left         key.Binding
	Right        key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	Top          key.Binding
	Bottom       key.Binding
	Enter        key.Binding
	Back         key.Binding
	Tab          key.Binding
	ShiftTab     key.Binding
	Filter       key.Binding
	Upload       key.Binding
	Download     key.Binding
	Delete       key.Binding
	Presign      key.Binding
	Inspector    key.Binding
	ToggleDetail key.Binding
	Refresh      key.Binding
	Help         key.Binding
	Quit         key.Binding
	Escape         key.Binding
	Select         key.Binding
	SelectAll      key.Binding
	ClearSelect    key.Binding
	OpenMedia      key.Binding
}

var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/↓", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("h", "left"),
		key.WithHelp("h/←", "left"),
	),
	Right: key.NewBinding(
		key.WithKeys("l", "right"),
		key.WithHelp("l/→", "right"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("ctrl+u", "pgup"),
		key.WithHelp("C-u", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("ctrl+d", "pgdown"),
		key.WithHelp("C-d", "page down"),
	),
	Top: key.NewBinding(
		key.WithKeys("g", "home"),
		key.WithHelp("g", "top"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("G", "end"),
		key.WithHelp("G", "bottom"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select/open"),
	),
	Back: key.NewBinding(
		key.WithKeys("backspace", "esc", "h", "left"),
		key.WithHelp("backspace/h", "back"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next pane"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev pane"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Upload: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "upload"),
	),
	Download: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "download"),
	),
	Delete: key.NewBinding(
		key.WithKeys("x", "delete", "D"),
		key.WithHelp("x/del", "delete"),
	),
	Presign: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "presign URL"),
	),
	Inspector: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "toggle inspector"),
	),
	ToggleDetail: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "inspector"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel/close"),
	),
	Select: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle select"),
	),
	SelectAll: key.NewBinding(
		key.WithKeys("a", "ctrl+a"),
		key.WithHelp("a/C-a", "select all"),
	),
	ClearSelect: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clear selection"),
	),
	OpenMedia: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open media"),
	),
}
