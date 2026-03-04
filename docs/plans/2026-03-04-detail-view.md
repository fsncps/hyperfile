# Detail View Mode Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a toggleable detail-view mode to any tree panel that displays a flat, info-rich file list (name, permissions, size, mtime) from the directory cursor-selected in the opposite panel.

**Architecture:** A new `treePanelMode` enum flag on `treePanelModel` controls whether the panel renders as a tree or as a columnar detail list. Detail mode maintains its own `detailEntries []detailEntry` slice and `detailRoot string`, populated from the opposite panel's selected directory. Navigation reuses the existing cursor/renderIdx pair via a new `EntryCount()` method that returns the right bound for the active mode. The left panel's Confirm key refreshes the right panel's entries when it's in detail mode.

**Tech Stack:** Go, Bubble Tea (charmbracelet), lipgloss, os/fs standard library, `slices` standard library.

---

### Task 1: Core types and data functions

**Files:**
- Modify: `src/internal/tree_panel.go`
- Test: `src/internal/tree_panel_detail_test.go` (new)

**Background:** `treePanelModel` currently only supports tree mode. We need a mode enum, a detail-entry struct, two new fields on the model, a builder function, and an `EntryCount()` helper.

**Step 1: Write failing tests**

Create `src/internal/tree_panel_detail_test.go`:

```go
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
```

**Step 2: Run tests to verify they fail**

```bash
go test ./src/internal/... -run "TestBuildDetailEntries|TestEntryCount" -v
```
Expected: FAIL — `treePanelMode`, `detailEntry`, `buildDetailEntries`, `EntryCount` undefined.

**Step 3: Add types and functions to `tree_panel.go`**

After the `treeNode` struct definition (around line 26), add:

```go
// treePanelMode controls whether the panel shows the tree or the detail list.
type treePanelMode int

const (
	treePanelModeTree   treePanelMode = iota
	treePanelModeDetail               // flat info-rich file list
)

// detailEntry holds the stat information for one entry in detail-view mode.
type detailEntry struct {
	name    string
	path    string
	isDir   bool
	size    int64
	modTime time.Time
	mode    os.FileMode
}
```

Add `"time"` to the import block (it's already importing `"os"` and `"path/filepath"`).

Add two fields to `treePanelModel` (after `showHidden bool`):

```go
	mode         treePanelMode
	detailRoot   string
	detailEntries []detailEntry
```

Add `buildDetailEntries` function after `addTreeNodes`:

```go
// buildDetailEntries reads dir and returns a flat, stat-populated slice sorted
// directories-first then alphabetically (matching the tree ordering).
func buildDetailEntries(dir string, showHidden bool) []detailEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		slog.Debug("detail: cannot read dir", "dir", dir, "err", err)
		return nil
	}
	result := make([]detailEntry, 0, len(entries))
	for _, e := range entries {
		if len(e.Name()) > 0 && e.Name()[0] == '.' && !showHidden {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, detailEntry{
			name:    e.Name(),
			path:    filepath.Join(dir, e.Name()),
			isDir:   e.IsDir(),
			size:    info.Size(),
			modTime: info.ModTime(),
			mode:    info.Mode(),
		})
	}
	slices.SortStableFunc(result, func(a, b detailEntry) int {
		if a.isDir == b.isDir {
			return 0
		}
		if a.isDir {
			return -1
		}
		return 1
	})
	return result
}
```

Add `EntryCount` method after `HasChildren`:

```go
// EntryCount returns the number of navigable entries in the current mode.
// Tree mode → len(nodes); detail mode → len(detailEntries).
func (t *treePanelModel) EntryCount() int {
	if t.mode == treePanelModeDetail {
		return len(t.detailEntries)
	}
	return len(t.nodes)
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./src/internal/... -run "TestBuildDetailEntries|TestEntryCount" -v
```
Expected: PASS (5 tests).

**Step 5: Commit**

```bash
git add src/internal/tree_panel.go src/internal/tree_panel_detail_test.go
git commit -m "feat: add treePanelMode, detailEntry, buildDetailEntries, EntryCount"
```

---

### Task 2: Hotkey configuration

**Files:**
- Modify: `src/internal/common/config_type.go`
- Modify: `src/hyperfile_config/hotkeys.toml`

**Background:** The toggle key must be user-configurable. We add `ToggleDetailView` to `HotkeysType` and give it a default in the embedded TOML. No new struct embedding or test for config loading is needed — the pattern is identical to `ToggleFooter` which was added the same way.

**Step 1: Add field to `HotkeysType` in `src/internal/common/config_type.go`**

In the `HotkeysType` struct, after `ToggleFilePreviewPanel` (line ~134), add:

```go
	ToggleDetailView       []string `toml:"toggle_detail_view"`
```

**Step 2: Add default to `src/hyperfile_config/hotkeys.toml`**

In the `# App / UI operations  alt+` section (after `toggle_file_preview_panel`), add:

```toml
toggle_detail_view                 = ['alt+d']
```

**Step 3: Build to verify no compile errors**

```bash
CGO_ENABLED=0 go build -o ./bin/hpf ./src/cmd/...
```
Expected: SUCCESS (no errors).

**Step 4: Commit**

```bash
git add src/internal/common/config_type.go src/hyperfile_config/hotkeys.toml
git commit -m "feat: add toggle_detail_view hotkey config (default alt+d)"
```

---

### Task 3: Toggle behavior and mode-aware navigation

**Files:**
- Modify: `src/internal/handle_panel_navigation.go`
- Modify: `src/internal/tree_panel.go` (cursor navigation methods)
- Modify: `src/internal/handle_tree_panel.go` (key handler)
- Test: `src/internal/tree_panel_detail_test.go` (extend existing file)

**Background:** `toggleDetailView` switches a panel between tree and detail mode. Navigation methods `moveDown`, `ListUp`, `ListDown` currently use `len(t.nodes)` for bounds; they need to use `EntryCount()` so they work in detail mode too. Shift-select is read-only/disabled in detail mode for now.

**Step 1: Write failing tests**

Append to `src/internal/tree_panel_detail_test.go`:

```go
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
```

**Step 2: Run tests to verify they fail**

```bash
go test ./src/internal/... -run "TestToggleDetailView|TestDetailMode" -v
```
Expected: FAIL — `toggleDetailView` undefined, `treePanelModeDetail` undefined.

**Step 3: Fix navigation bounds in `tree_panel.go`**

In `moveDown` (around line 246), replace `len(t.nodes)-1` with `t.EntryCount()-1`:

```go
func (t *treePanelModel) moveDown(visibleH int) {
	if t.cursor < t.EntryCount()-1 {
		t.cursor++
		if t.cursor >= t.renderIdx+visibleH {
			t.renderIdx++
		}
	}
}
```

In `ListUp` (around line 256), replace `len(t.nodes)` → `t.EntryCount()`:

```go
func (t *treePanelModel) ListUp(visibleHeight int) {
	if t.EntryCount() == 0 {
		return
	}
	t.ClearSelection()
	if t.cursor > 0 {
		t.moveUp()
	} else {
		t.cursor = t.EntryCount() - 1
		maxRender := t.EntryCount() - visibleHeight
		if maxRender < 0 {
			maxRender = 0
		}
		t.renderIdx = maxRender
	}
}
```

In `ListDown` (around line 274), replace `len(t.nodes)` → `t.EntryCount()`:

```go
func (t *treePanelModel) ListDown(visibleHeight int) {
	if t.EntryCount() == 0 {
		return
	}
	t.ClearSelection()
	if t.cursor < t.EntryCount()-1 {
		t.moveDown(visibleHeight)
	} else {
		t.cursor = 0
		t.renderIdx = 0
	}
}
```

In `ShiftListUp` and `ShiftListDown`, add a detail-mode guard at the top:

```go
func (t *treePanelModel) ShiftListUp(visibleH int) {
	if t.mode == treePanelModeDetail || len(t.nodes) == 0 {
		return
	}
	t.setAnchorIfUnset()
	t.moveUp()
	t.applyRangeSelection()
}

func (t *treePanelModel) ShiftListDown(visibleH int) {
	if t.mode == treePanelModeDetail || len(t.nodes) == 0 {
		return
	}
	t.setAnchorIfUnset()
	t.moveDown(visibleH)
	t.applyRangeSelection()
}
```

Also update `rebuild`'s cursor clamp to use `EntryCount()`:

```go
func (t *treePanelModel) rebuild() {
	t.nodes = buildTreeNodes(t.root, t.maxDepth, t.collapsed, t.expanded, t.showHidden)
	if t.cursor >= t.EntryCount() {
		t.cursor = max(0, t.EntryCount()-1)
	}
	if t.renderIdx > t.cursor {
		t.renderIdx = t.cursor
	}
}
```

**Step 4: Add `toggleDetailView` to `handle_panel_navigation.go`**

After `toggleTree2Panel`, add:

```go
// toggleDetailView switches panel idx between tree mode and detail mode.
// When entering detail mode the entries are loaded from the opposite panel's
// currently selected directory (or its root when the cursor is on a file).
func (m *model) toggleDetailView(idx int) {
	tree := &m.treePanels[idx]
	if tree.mode == treePanelModeDetail {
		tree.mode = treePanelModeTree
		return
	}
	otherIdx := 1 - idx
	source := &m.treePanels[otherIdx]
	root := source.root
	if node := source.GetSelectedNode(); node != nil && node.isDir {
		root = node.path
	}
	tree.detailRoot = root
	tree.detailEntries = buildDetailEntries(root, tree.showHidden)
	tree.cursor = 0
	tree.renderIdx = 0
	tree.mode = treePanelModeDetail
}
```

**Step 5: Wire key handler in `handle_tree_panel.go`**

In `handleTreePanelKey`, after the `ToggleFilePreviewPanel` case, add:

```go
	case slices.Contains(common.Hotkeys.ToggleDetailView, msg):
		m.toggleDetailView(idx)
```

**Step 6: Run tests**

```bash
go test ./src/internal/... -run "TestToggleDetailView|TestDetailMode|TestBuildDetailEntries|TestEntryCount" -v
```
Expected: PASS (9 tests).

**Step 7: Commit**

```bash
git add src/internal/tree_panel.go src/internal/handle_panel_navigation.go \
        src/internal/handle_tree_panel.go src/internal/tree_panel_detail_test.go
git commit -m "feat: toggle detail view mode with mode-aware navigation"
```

---

### Task 4: Detail view rendering

**Files:**
- Modify: `src/internal/tree_panel_render.go`

**Background:** When `tree.mode == treePanelModeDetail`, the panel renders a columnar list: icon+name | permissions (10 chars) | size (8 chars right-aligned) | date (`Jan 02 15:04` = 12 chars). The header row is identical to tree mode. No tests are written for rendering (it's presentation-only and would be brittle snapshot tests); verify visually by running the app.

**Step 1: Add `formatDetailSize` helper to `tree_panel_render.go`**

At the bottom of the file, add:

```go
// formatDetailSize returns a human-readable size string right-padded to 8 chars.
func formatDetailSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	var s string
	switch {
	case bytes >= gb:
		s = fmt.Sprintf("%.1fG", float64(bytes)/gb)
	case bytes >= mb:
		s = fmt.Sprintf("%.1fM", float64(bytes)/mb)
	case bytes >= kb:
		s = fmt.Sprintf("%.1fK", float64(bytes)/kb)
	default:
		s = fmt.Sprintf("%dB", bytes)
	}
	return fmt.Sprintf("%8s", s)
}
```

**Step 2: Add detail-mode branch to `treePanelRender`**

In `treePanelRender` (file `tree_panel_render.go`), immediately after `r.AddSection()` (line ~45) and before `if len(tree.nodes) == 0 {`, insert:

```go
	// ── Detail view mode ───────────────────────────────────────────────────
	if tree.mode == treePanelModeDetail {
		if len(tree.detailEntries) == 0 {
			r.AddLines(common.FilePanelNoneText)
			return r.Render()
		}
		visibleH := m.mainPanelHeight - 2
		end := min(tree.renderIdx+visibleH, len(tree.detailEntries))
		// Fixed column widths: perms(10) + space + size(8) + space + date(12) + space = 32
		const fixedCols = 32
		cursorOverhead := 2 // cursorChar + space
		nameWidth := r.ContentWidth() - cursorOverhead - fixedCols
		if nameWidth < 4 {
			nameWidth = 4
		}
		for i := tree.renderIdx; i < end; i++ {
			e := tree.detailEntries[i]
			cursorChar := " "
			if i == tree.cursor {
				cursorChar = icon.Cursor
			}
			name := common.PrettierName(e.name, nameWidth, e.isDir, false, common.FilePanelBGColor)
			perms := e.mode.String() // "-rwxr-xr-x" = 10 chars
			size := formatDetailSize(e.size)
			date := e.modTime.Format("Jan 02 15:04")
			line := common.FilePanelCursorStyle.Render(cursorChar+" ") +
				name + " " + perms + " " + size + " " + date
			r.AddLines(line)
		}
		return r.Render()
	}
	// ── Tree mode (existing render below) ──────────────────────────────────
```

Also add `"time"` to the import (if not already present) — not needed since we use `e.modTime.Format(...)` which is a `time.Time` method on the entry, no additional import needed in render file. But `fmt` is already imported.

**Step 3: Build and run the app**

```bash
CGO_ENABLED=0 go build -o ./bin/hpf ./src/cmd/... && ./bin/hpf
```

Press `alt+d` with focus on either panel to toggle detail view. Verify:
- Header shows root path and depth indicator
- Each row shows: cursor indicator | icon+name | permissions | size | date
- `up`/`down` move the cursor
- `alt+d` again restores tree view

**Step 4: Run full test suite**

```bash
go test ./src/internal/... -v 2>&1 | tail -20
```
Expected: same pass/fail baseline as before this task (no regressions; pre-existing failures in `ui/metadata` and `ui/prompt` are unchanged).

**Step 5: Commit**

```bash
git add src/internal/tree_panel_render.go
git commit -m "feat: render detail view mode with name/perms/size/date columns"
```

---

### Task 5: Left-panel sync — update detail root on Confirm

**Files:**
- Modify: `src/internal/handle_tree_panel.go`
- Test: `src/internal/tree_panel_detail_test.go` (extend)

**Background:** When tree1 confirms (right arrow) on a directory node, and tree2 is in detail mode, tree2's `detailEntries` should refresh to show the contents of that directory.

**Step 1: Write failing test**

Append to `src/internal/tree_panel_detail_test.go`:

```go
func TestConfirm_UpdatesDetailPanelWhenOtherIsInDetailMode(t *testing.T) {
	dir := populatedTempDir(t) // creates aaa/, bbb/, ccc/, ddd/
	m := defaultTestModel(dir)
	// Put tree2 in detail mode (will initially load from tree1's root)
	m.toggleDetailView(1)
	initialRoot := m.treePanels[1].detailRoot

	// Move tree1 cursor to first dir node (should be "aaa") and confirm
	require.Greater(t, len(m.treePanels[0].nodes), 0)
	node := m.treePanels[0].GetSelectedNode()
	require.NotNil(t, node)
	require.True(t, node.isDir)

	_, _ = TeaUpdate(m, keyMsg("right"))

	// Detail panel root should now be the confirmed dir
	assert.NotEqual(t, initialRoot, m.treePanels[1].detailRoot,
		"detail root should update after confirming into a dir on the opposite panel")
	assert.Equal(t, node.path, m.treePanels[1].detailRoot)
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./src/internal/... -run "TestConfirm_UpdatesDetailPanel" -v
```
Expected: FAIL — detail root not updated.

**Step 3: Add sync logic to `handleTreePanelKey`**

In `handle_tree_panel.go`, find the `Confirm` case (around line 33):

```go
	case slices.Contains(common.Hotkeys.Confirm, msg):
		m.treeEnterNode(idx)
		panel := m.getFocusedFilePanel()
		...
		return nil
```

Immediately after `m.treeEnterNode(idx)`, insert:

```go
		// Sync opposite panel if it is in detail mode.
		otherIdx := 1 - idx
		if m.treePanels[otherIdx].mode == treePanelModeDetail {
			if node := m.treePanels[idx].GetSelectedNode(); node != nil && node.isDir {
				other := &m.treePanels[otherIdx]
				other.detailRoot = node.path
				other.detailEntries = buildDetailEntries(node.path, other.showHidden)
				other.cursor = 0
				other.renderIdx = 0
			}
		}
```

**Step 4: Run test**

```bash
go test ./src/internal/... -run "TestConfirm_UpdatesDetailPanel" -v
```
Expected: PASS.

**Step 5: Run full test suite**

```bash
go test ./src/internal/... 2>&1 | grep -E "^(ok|FAIL|---)"
```
Expected: same baseline (no new failures).

**Step 6: Commit**

```bash
git add src/internal/handle_tree_panel.go src/internal/tree_panel_detail_test.go
git commit -m "feat: sync detail view root when opposite panel confirms into a dir"
```
