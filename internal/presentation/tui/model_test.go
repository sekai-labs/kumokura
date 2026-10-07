package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	bucketDomain "github.com/sekai-labs/kumokura/internal/buckets/domain"
	objDomain "github.com/sekai-labs/kumokura/internal/objects/domain"
	objPorts "github.com/sekai-labs/kumokura/internal/objects/ports"
	"github.com/sekai-labs/kumokura/internal/presentation/tui/messages"
	"github.com/stretchr/testify/assert"
)

func TestTUI_ModelInitAndRender(t *testing.T) {
	model := NewModel(Services{})
	assert.Equal(t, 0, model.activeTab)

	viewStr := model.View()
	assert.NotEmpty(t, viewStr)
	assert.Contains(t, viewStr, "1: Explorer")
	assert.Contains(t, viewStr, "BUCKETS")
}

func TestTUI_ResponsiveResize(t *testing.T) {
	model := NewModel(Services{})

	m, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	updated := m.(Model)
	assert.Equal(t, 80, updated.width)
	assert.Equal(t, 24, updated.height)

	render80x24 := updated.View()
	assert.NotEmpty(t, render80x24)

	m160, _ := model.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	updated160 := m160.(Model)
	render160 := updated160.View()
	assert.NotEmpty(t, render160)
}

func TestTUI_TabSwitching(t *testing.T) {
	model := NewModel(Services{})

	m2, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	updated := m2.(Model)
	assert.Equal(t, 1, updated.activeTab)
	viewTransfers := updated.View()
	assert.Contains(t, viewTransfers, "TRANSFERS")

	m1, _ := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	updated1 := m1.(Model)
	assert.Equal(t, 0, updated1.activeTab)
}

func TestTUI_DataLoadedHandling(t *testing.T) {
	model := NewModel(Services{})

	m, _ := model.Update(messages.BucketsLoadedMsg{
		Buckets: []bucketDomain.Bucket{
			{Name: "prod-storage", Region: "us-west-2"},
		},
	})
	updated := m.(Model)
	assert.Len(t, updated.explorerView.Buckets, 1)
	assert.Equal(t, "prod-storage", updated.explorerView.Buckets[0].Name)

	mObj, _ := updated.Update(messages.ObjectsLoadedMsg{
		Result: objPorts.ListObjectsResult{
			Objects: []objDomain.Object{
				{Key: "data/db.sqlite", Size: 4096},
			},
			CommonPrefixes: []objDomain.Prefix{
				{Prefix: "data/logs/"},
			},
		},
	})
	updatedObj := mObj.(Model)
	assert.Len(t, updatedObj.explorerView.Objects, 1)
	assert.Len(t, updatedObj.explorerView.Prefixes, 1)
}

func TestTUI_HelpModal(t *testing.T) {
	model := NewModel(Services{})

	m, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	updated := m.(Model)
	assert.True(t, updated.showHelpModal)
	view := updated.View()
	assert.Contains(t, view, "HELP OVERLAY")

	mEsc, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	closed := mEsc.(Model)
	assert.False(t, closed.showHelpModal)
}
