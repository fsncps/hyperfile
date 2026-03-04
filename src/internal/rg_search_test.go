package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCtrlGOpenRgSearchBar(t *testing.T) {
	m := defaultTestModel(t.TempDir())
	m.setTree1PanelActive()

	m.handleTreePanelKey("ctrl+g", 0)
	assert.True(t, m.treePanels[0].rgSearchBar.Focused())
}

func TestCtrlGCloseRgSearchBar(t *testing.T) {
	m := defaultTestModel(t.TempDir())
	m.setTree1PanelActive()
	m.treePanels[0].rgSearchBar.Focus()

	m.handleTreePanelKey("ctrl+g", 0)
	assert.False(t, m.treePanels[0].rgSearchBar.Focused())
	assert.Equal(t, "", m.treePanels[0].rgSearchBar.Value())
	assert.Nil(t, m.treePanels[0].rgMatches)
	assert.Equal(t, 0, m.treePanels[0].cursor)
	assert.Equal(t, 0, m.treePanels[0].renderIdx)
}

func TestEscClosesRgSearchBar(t *testing.T) {
	m := defaultTestModel(t.TempDir())
	m.setTree1PanelActive()
	m.treePanels[0].rgSearchBar.Focus()
	m.treePanels[0].rgSearchBar.SetValue("hello")

	cmd := m.handleTreePanelKey("esc", 0)
	assert.Nil(t, cmd)
	assert.False(t, m.treePanels[0].rgSearchBar.Focused())
	assert.Equal(t, "", m.treePanels[0].rgSearchBar.Value())
	assert.Nil(t, m.treePanels[0].rgMatches)
}
