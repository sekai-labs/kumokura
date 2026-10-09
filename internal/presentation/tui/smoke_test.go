package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestTUI_SmokeRenderView(t *testing.T) {
	model := NewModel(Services{})
	rendered := model.View()

	assert.NotEmpty(t, rendered)

	assert.True(t, strings.Contains(rendered, "[1 Buckets]"))
	assert.True(t, strings.Contains(rendered, "[2 Objects]"))
	assert.True(t, strings.Contains(rendered, "[3 Transfers]"))
	assert.True(t, strings.Contains(rendered, "[4 Sync]"))

	assert.True(t, strings.Contains(rendered, "BUCKETS"))
	assert.True(t, strings.Contains(rendered, "OBJECT EXPLORER"))

	assert.True(t, strings.Contains(rendered, "[Tab] Switch Pane"))
	assert.True(t, strings.Contains(rendered, "[j/k] Navigate"))
	assert.True(t, strings.Contains(rendered, "[/] Filter"))
	assert.True(t, strings.Contains(rendered, "[u] Upload"))
	assert.True(t, strings.Contains(rendered, "[d] Download"))
	assert.True(t, strings.Contains(rendered, "[?] Help"))

	m1, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	v1 := m1.(Model).View()
	assert.NotEmpty(t, v1)
	assert.Contains(t, v1, "BUCKETS")

	m2, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	v2 := m2.(Model).View()
	assert.NotEmpty(t, v2)
	assert.Contains(t, v2, "OBJECT EXPLORER")
}
