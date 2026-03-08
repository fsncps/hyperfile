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
	// tree0 cursor starts at 0; ensure the first entry (a dir) has children
	// so toggleDetailView uses that dir and gets non-empty detailEntries.
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "file.txt"), []byte("x"), 0644))
	m := defaultTestModel(dir)
	m.toggleDetailView(1)
	assert.Equal(t, treePanelModeDetail, m.secondaryPanel.mode)
	assert.NotEmpty(t, m.secondaryPanel.detailEntries)
}

func TestToggleDetailView_TogglesBackToTree(t *testing.T) {
	dir := populatedTempDir(t)
	m := defaultTestModel(dir)
	m.toggleDetailView(1)
	m.toggleDetailView(1)
	assert.Equal(t, treePanelModeTree, m.secondaryPanel.mode)
}

func TestDetailMode_NavigationUsesEntryCount(t *testing.T) {
	// tree0 cursor at 0; first entry must be a dir with multiple children
	// so detail mode has >1 entry and navigation can move the cursor.
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(subdir, name), []byte("x"), 0644))
	}
	m := defaultTestModel(dir)
	// Set primary panel to show subdir so detail view uses it
	m.primaryPanel.maxDepth = 1
	m.primaryPanel.nodes = buildTreeNodesWithRoot(dir, 1, m.primaryPanel.collapsed, m.primaryPanel.expanded, m.primaryPanel.showHidden)
	// Find and select subdir in primary panel
	for i, n := range m.primaryPanel.nodes {
		if n.path == subdir {
			m.primaryPanel.cursor = i
			break
		}
	}
	tree := &m.secondaryPanel
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
	tree := &m.secondaryPanel
	tree.ShiftListDown(20)
	assert.False(t, tree.HasSelection(), "shift-select should be disabled in detail mode")
}

func TestConfirm_UpdatesDetailPanelWhenOtherIsInDetailMode(t *testing.T) {
	// Create a dir with two subdirs, each containing at least one file.
	dir := t.TempDir()
	sub1 := filepath.Join(dir, "sub1")
	sub2 := filepath.Join(dir, "sub2")
	require.NoError(t, os.MkdirAll(sub1, 0755))
	require.NoError(t, os.MkdirAll(sub2, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sub1, "a.txt"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sub2, "b.txt"), []byte("x"), 0644))

	m := defaultTestModel(dir)
	// Set primary panel to show subdirs so detail view can use them
	m.primaryPanel.maxDepth = 1
	m.primaryPanel.nodes = buildTreeNodesWithRoot(dir, 1, m.primaryPanel.collapsed, m.primaryPanel.expanded, m.primaryPanel.showHidden)
	// Find and select sub1 in primary panel
	for i, n := range m.primaryPanel.nodes {
		if n.path == sub1 {
			m.primaryPanel.cursor = i
			break
		}
	}
	// tree1 cursor starts at sub1. Toggle detail on tree2.
	m.toggleDetailView(1)
	require.Equal(t, sub1, m.secondaryPanel.detailRoot, "detail root should be sub1 initially")

	// Move tree1 cursor to sub2 and send Confirm (right arrow).
	for i, n := range m.primaryPanel.nodes {
		if n.path == sub2 {
			m.primaryPanel.cursor = i
			break
		}
	}
	_, _ = TeaUpdate(m, keyMsg("right"))

	assert.Equal(t, sub2, m.secondaryPanel.detailRoot,
		"detail root should update to sub2 after confirming on tree1")
}
