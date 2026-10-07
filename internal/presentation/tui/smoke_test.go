package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTUI_SmokeRenderView(t *testing.T) {
	model := NewModel(Services{})
	rendered := model.View()

	assert.NotEmpty(t, rendered)

	assert.True(t, strings.Contains(rendered, "1: Explorer"))
	assert.True(t, strings.Contains(rendered, "2: Transfers"))
	assert.True(t, strings.Contains(rendered, "3: Accounts"))
	assert.True(t, strings.Contains(rendered, "4: Help"))

	assert.True(t, strings.Contains(rendered, "BUCKETS"))
	assert.True(t, strings.Contains(rendered, "OBJECTS"))

	assert.True(t, strings.Contains(rendered, "[j/k] Navigate"))
	assert.True(t, strings.Contains(rendered, "[Enter] Open"))
	assert.True(t, strings.Contains(rendered, "[Tab] Switch Pane"))
	assert.True(t, strings.Contains(rendered, "[?] Help"))
	assert.True(t, strings.Contains(rendered, "[q] Quit"))
}
