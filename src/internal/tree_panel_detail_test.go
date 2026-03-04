package internal

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDetailEntries_ReturnsSortedDirsFirst(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b_file.txt"), []byte("x"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "a_dir"), 0755))

	entries := buildDetailEntries(dir, false)
	require.Len(t, entries, 2)
	assert.True(t, entries[0].isDir, "directories should come first")
	assert.Equal(t, "a_dir", entries[0].name)
	assert.Equal(t, "b_file.txt", entries[1].name)
}

func TestBuildDetailEntries_RespectsShowHidden(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".hidden"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "visible"), []byte("x"), 0644))

	hidden := buildDetailEntries(dir, false)
	require.Len(t, hidden, 1)
	assert.Equal(t, "visible", hidden[0].name)

	all := buildDetailEntries(dir, true)
	assert.Len(t, all, 2)
}

func TestBuildDetailEntries_PopulatesStats(t *testing.T) {
	dir := t.TempDir()
	content := []byte("hello world")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), content, 0644))

	entries := buildDetailEntries(dir, false)
	require.Len(t, entries, 1)
	e := entries[0]
	assert.Equal(t, "file.txt", e.name)
	assert.Equal(t, filepath.Join(dir, "file.txt"), e.path)
	assert.False(t, e.isDir)
	assert.EqualValues(t, len(content), e.size)
	assert.WithinDuration(t, time.Now(), e.modTime, 5*time.Second)
}

func TestEntryCount_TreeMode(t *testing.T) {
	dir := populatedTempDir(t)
	tp := defaultTreePanel(dir)
	assert.Greater(t, tp.EntryCount(), 0)
	assert.Equal(t, len(tp.nodes), tp.EntryCount())
}

func TestEntryCount_DetailMode(t *testing.T) {
	dir := populatedTempDir(t)
	tp := defaultTreePanel(dir)
	tp.mode = treePanelModeDetail
	tp.detailEntries = buildDetailEntries(dir, false)
	assert.Greater(t, tp.EntryCount(), 0)
	assert.Equal(t, len(tp.detailEntries), tp.EntryCount())
}

func TestToggleDetailView_SwitchesToDetailMode(t *testing.T) {
	dir := populatedTempDir(t)
	m := defaultTestModel(dir)
	// tree1 cursor is on a dir; toggle detail on tree2
	m.toggleDetailView(1)
	assert.Equal(t, treePanelModeDetail, m.treePanels[1].mode)
	assert.NotEmpty(t, m.treePanels[1].detailEntries)
}

func TestToggleDetailView_TogglesBackToTree(t *testing.T) {
	dir := populatedTempDir(t)
	m := defaultTestModel(dir)
	m.toggleDetailView(1)
	m.toggleDetailView(1)
	assert.Equal(t, treePanelModeTree, m.treePanels[1].mode)
}

func TestDetailMode_NavigationUsesEntryCount(t *testing.T) {
	dir := populatedTempDir(t)
	m := defaultTestModel(dir)
	tree := &m.treePanels[1]
	m.toggleDetailView(1)
	require.Greater(t, tree.EntryCount(), 1)
	initialCursor := tree.cursor
	tree.ListDown(20)
	assert.Greater(t, tree.cursor, initialCursor, "cursor should move down in detail mode")
}

func TestDetailMode_ShiftSelectIsNoop(t *testing.T) {
	dir := populatedTempDir(t)
	m := defaultTestModel(dir)
	m.toggleDetailView(1)
	tree := &m.treePanels[1]
	tree.ShiftListDown(20)
	assert.False(t, tree.HasSelection(), "shift-select should be disabled in detail mode")
}
