package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleTreePanelKey_ShiftDown_SelectsRange(t *testing.T) {
	m := defaultTestModel(populatedTempDir(t))
	tree := &m.treePanels[0]
	require.Greater(t, len(tree.nodes), 2)

	_, _ = TeaUpdate(m, keyMsg("shift+down"))
	assert.True(t, tree.HasSelection(), "shift+down should create a selection")
	assert.Equal(t, 1, tree.cursor)
}

func TestHandleTreePanelKey_PlainDown_ClearsSelection(t *testing.T) {
	m := defaultTestModel(populatedTempDir(t))
	tree := &m.treePanels[0]
	require.Greater(t, len(tree.nodes), 2)

	_, _ = TeaUpdate(m, keyMsg("shift+down"))
	require.True(t, tree.HasSelection())

	_, _ = TeaUpdate(m, keyMsg("down"))
	assert.False(t, tree.HasSelection())
}

// populatedTempDir creates a tempdir with enough entries for selection tests.
func populatedTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"aaa", "bbb", "ccc", "ddd"} {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, name), 0755))
	}
	return dir
}

func TestTreePanelSelection_ShiftDown_SetsAnchorAndRange(t *testing.T) {
	m := defaultTestModel(populatedTempDir(t))
	tree := &m.treePanels[0]
	require.Greater(t, len(tree.nodes), 2, "need at least 3 nodes")

	tree.ShiftListDown(10)
	assert.Equal(t, 0, tree.anchor, "anchor should be set to initial cursor (0)")
	assert.Equal(t, 1, tree.cursor)
	assert.Len(t, tree.selected, 2, "nodes 0 and 1 should be selected")
	assert.True(t, tree.selected[tree.nodes[0].path])
	assert.True(t, tree.selected[tree.nodes[1].path])
}

func TestTreePanelSelection_PlainDown_ClearsSelection(t *testing.T) {
	m := defaultTestModel(populatedTempDir(t))
	tree := &m.treePanels[0]
	require.Greater(t, len(tree.nodes), 2)

	tree.ShiftListDown(10) // select something
	tree.ListDown(10)      // plain nav should clear
	assert.Empty(t, tree.selected)
	assert.Equal(t, -1, tree.anchor)
}

func TestTreePanelSelection_ShiftUp_ShrinksRange(t *testing.T) {
	m := defaultTestModel(populatedTempDir(t))
	tree := &m.treePanels[0]
	require.Greater(t, len(tree.nodes), 2)

	tree.ShiftListDown(10) // cursor=1, anchor=0, selected=[0,1]
	tree.ShiftListDown(10) // cursor=2, anchor=0, selected=[0,1,2]
	tree.ShiftListUp(10)   // cursor=1, anchor=0, selected=[0,1]
	assert.Len(t, tree.selected, 2)
	assert.False(t, tree.selected[tree.nodes[2].path])
}

func TestTreePanelSelection_ToggleSelected(t *testing.T) {
	m := defaultTestModel(populatedTempDir(t))
	tree := &m.treePanels[0]
	require.Greater(t, len(tree.nodes), 0)

	path := tree.nodes[0].path
	tree.ToggleSelected(path)
	assert.True(t, tree.selected[path])
	tree.ToggleSelected(path)
	assert.False(t, tree.selected[path])
}

func TestTreePanelSelection_ClearedOnRootChange(t *testing.T) {
	dir := populatedTempDir(t)
	m := defaultTestModel(dir)
	tree := &m.treePanels[0]
	tree.ShiftListDown(10)
	require.True(t, tree.HasSelection())

	tree.SetRoot(t.TempDir())
	assert.Empty(t, tree.selected)
	assert.Equal(t, -1, tree.anchor)
}
