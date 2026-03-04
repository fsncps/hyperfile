# Ripgrep Content Filter Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `ctrl+g` to the active tree panel to open an inline rg-powered content filter that hides non-matching files from the tree in real time.

**Architecture:** A `rgSearchBar textinput.Model` and `rgMatches map[string]bool` are added to `treePanelModel`. On each keystroke, a 300ms debounce fires `rg --files-with-matches` as a goroutine; the result arrives as `RgResultMsg` (implementing `ModelUpdateMessage`) and calls `rebuild()` with the match set. `buildTreeNodes` skips files not in `rgMatches` and dirs with no matching descendants. Esc or `ctrl+g` clears the filter and restores the full tree.

**Tech Stack:** Go, Bubble Tea (`charmbracelet/bubbles/textinput`), `os/exec` for rg, existing `ModelUpdateMessage` pattern in `model_msg.go`.

---

### Task 1: Hotkey config

**Files:**
- Modify: `src/hyperfile_config/hotkeys.toml`
- Modify: `src/internal/common/config_type.go:138-170`

**Step 1: Add field to HotkeysType**

In `config_type.go`, inside `HotkeysType`, add after `DragItems` (line 142):

```go
ContentSearch []string `toml:"content_search"`
```

**Step 2: Add default in hotkeys.toml**

In `hotkeys.toml`, add after `drag_items = ['ctrl+d']` under the file operations section:

```toml
content_search         = ['ctrl+g']
```

**Step 3: Build and verify no compile errors**

```bash
make build
```
Expected: clean build, `./bin/hpf` produced.

**Step 4: Commit**

```bash
git add src/hyperfile_config/hotkeys.toml src/internal/common/config_type.go
git commit -m "feat: add content_search hotkey (ctrl+g) to config"
```

---

### Task 2: Tree panel filter data model + buildTreeNodes

**Files:**
- Modify: `src/internal/tree_panel.go`
- Test: `src/internal/function_test.go` (or new `src/internal/tree_panel_test.go`)

**Step 1: Write failing test**

Create `src/internal/tree_panel_filter_test.go`:

```go
package internal

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestBuildTreeNodesWithRgFilter(t *testing.T) {
    // Setup: tmp dir with a/match.go, a/skip.txt, b/match.go
    root := t.TempDir()
    _ = os.MkdirAll(filepath.Join(root, "a"), 0o755)
    _ = os.MkdirAll(filepath.Join(root, "b"), 0o755)
    _ = os.WriteFile(filepath.Join(root, "a", "match.go"), []byte("x"), 0o644)
    _ = os.WriteFile(filepath.Join(root, "a", "skip.txt"), []byte("x"), 0o644)
    _ = os.WriteFile(filepath.Join(root, "b", "match.go"), []byte("x"), 0o644)

    rgMatches := map[string]bool{
        filepath.Join(root, "a", "match.go"): true,
        filepath.Join(root, "b", "match.go"): true,
    }

    nodes := buildTreeNodes(root, 5, nil, nil, false, rgMatches)
    paths := make([]string, len(nodes))
    for i, n := range nodes {
        paths[i] = n.path
    }

    assert.Contains(t, paths, filepath.Join(root, "a"))
    assert.Contains(t, paths, filepath.Join(root, "a", "match.go"))
    assert.NotContains(t, paths, filepath.Join(root, "a", "skip.txt"))
    assert.Contains(t, paths, filepath.Join(root, "b"))
    assert.Contains(t, paths, filepath.Join(root, "b", "match.go"))
}

func TestBuildTreeNodesNoFilter(t *testing.T) {
    root := t.TempDir()
    _ = os.WriteFile(filepath.Join(root, "a.go"), []byte("x"), 0o644)
    _ = os.WriteFile(filepath.Join(root, "b.txt"), []byte("x"), 0o644)

    nodes := buildTreeNodes(root, 5, nil, nil, false, nil)
    assert.Len(t, nodes, 2) // no filter: both files shown
}
```

**Step 2: Run test — must fail**

```bash
go test ./src/internal/... -run TestBuildTreeNodesWithRgFilter -v
```
Expected: compile error — `buildTreeNodes` takes 5 args, not 6.

**Step 3: Add fields to treePanelModel**

In `tree_panel.go`, update the `treePanelModel` struct (starting line 47). Add after `width int`:

```go
rgSearchBar  textinput.Model
rgMatches    map[string]bool  // nil = no filter; non-nil = set of matching absolute file paths
lastRgInput  time.Time        // timestamp of last rg search bar keystroke (debounce sentinel)
```

Also add the `textinput` import if not present (it's already imported via `common`; import `"github.com/charmbracelet/bubbles/textinput"` directly at the top of `tree_panel.go`).

**Step 4: Add ancestor dir helper**

Add after `buildDetailEntries` (around line 167):

```go
// buildRgAncestorDirs returns the set of all ancestor directories (up to but not
// including root) of the paths in matches.
func buildRgAncestorDirs(matches map[string]bool, root string) map[string]bool {
    dirs := make(map[string]bool)
    for p := range matches {
        for dir := filepath.Dir(p); dir != root && strings.HasPrefix(dir, root+string(filepath.Separator)); dir = filepath.Dir(dir) {
            dirs[dir] = true
        }
    }
    return dirs
}
```

**Step 5: Update buildTreeNodes signature and filter logic**

Change `buildTreeNodes` (line 84) to:

```go
func buildTreeNodes(root string, maxDepth int, collapsed, expanded map[string]bool, showHidden bool, rgMatches map[string]bool) []treeNode {
    nodes := make([]treeNode, 0, 64)
    var rgAncestors map[string]bool
    if rgMatches != nil {
        rgAncestors = buildRgAncestorDirs(rgMatches, root)
    }
    addTreeNodes(&nodes, root, 0, maxDepth, collapsed, expanded, showHidden, rgMatches, rgAncestors)
    return nodes
}
```

Change `addTreeNodes` (line 90) signature to add the two filter params:

```go
func addTreeNodes(nodes *[]treeNode, dir string, depth, maxDepth int, collapsed, expanded map[string]bool, showHidden bool, rgMatches, rgAncestors map[string]bool) {
```

Inside the loop, after computing `path` and before appending `node`, add rg filter:

```go
    // rg content filter: skip entries not in the match/ancestor sets.
    if rgMatches != nil {
        if e.IsDir() && !rgAncestors[path] {
            continue
        }
        if !e.IsDir() && !rgMatches[path] {
            continue
        }
    }
```

In the recursive call at the end, pass the filter sets and change expand condition:

```go
    if e.IsDir() {
        var shouldExpand bool
        if rgMatches != nil {
            shouldExpand = rgAncestors[path] // always expand ancestors when filtering
        } else {
            shouldExpand = (depth < maxDepth || expanded[path]) && !collapsed[path]
        }
        if shouldExpand {
            addTreeNodes(nodes, path, depth+1, maxDepth, collapsed, expanded, showHidden, rgMatches, rgAncestors)
        }
    }
```

**Step 6: Update all callers of buildTreeNodes to pass rgMatches**

All callers pass `nil` for the new param, except `rebuild()` which passes `t.rgMatches`:

- `defaultTreePanel` (line 76): `buildTreeNodes(root, t.maxDepth, t.collapsed, t.expanded, t.showHidden, nil)`
- `NavigateTo` (line 179): `buildTreeNodes(root, t.maxDepth, t.collapsed, t.expanded, t.showHidden, nil)`
- `SetRoot` (line 193): `buildTreeNodes(root, t.maxDepth, t.collapsed, t.expanded, t.showHidden, nil)`
- `rebuild()` (line 198): `buildTreeNodes(t.root, t.maxDepth, t.collapsed, t.expanded, t.showHidden, t.rgMatches)`

Also in `NavigateTo` and `SetRoot`, clear the rg filter on root change. In `NavigateTo` after `t.expanded = make(...)`:

```go
t.rgMatches = nil
t.rgSearchBar.SetValue("")
t.rgSearchBar.Blur()
```

**Step 7: Init rgSearchBar in defaultTreePanel**

Add in `defaultTreePanel` after `anchor: -1`:

```go
rgSearchBar: common.GenerateRgSearchBar(),
```

Add `GenerateRgSearchBar()` to `src/internal/common/style_function.go` after `GenerateSearchBar()`:

```go
func GenerateRgSearchBar() textinput.Model {
    ti := textinput.New()
    ti.Cursor.Style = FooterCursorStyle
    ti.Cursor.TextStyle = FooterStyle
    ti.TextStyle = FilePanelStyle
    ti.Prompt = FilePanelTopDirectoryIconStyle.Render(" rg: ")
    ti.Cursor.Blink = true
    ti.PlaceholderStyle = FilePanelStyle
    ti.Placeholder = "search file contents..."
    ti.Blur()
    ti.CharLimit = 256
    return ti
}
```

**Step 8: Run tests**

```bash
go test ./src/internal/... -run TestBuildTreeNodes -v
```
Expected: both tests PASS.

**Step 9: Build**

```bash
make build
```

**Step 10: Commit**

```bash
git add src/internal/tree_panel.go src/internal/tree_panel_filter_test.go src/internal/common/style_function.go
git commit -m "feat: add rgMatches filter to treePanelModel and buildTreeNodes"
```

---

### Task 3: Async rg execution — RgResultMsg + debounce tick

**Files:**
- Modify: `src/internal/model_msg.go`
- Modify: `src/internal/model.go`

**Step 1: Add RgResultMsg to model_msg.go**

At the end of `model_msg.go`, append:

```go
// RgResultMsg carries the result of an async rg --files-with-matches run.
// It implements ModelUpdateMessage; stale results are discarded via query comparison.
type RgResultMsg struct {
    BaseMessage
    panelIdx int
    query    string
    matches  map[string]bool
}

func (msg RgResultMsg) ApplyToModel(m *model) tea.Cmd {
    tree := &m.treePanels[msg.panelIdx]
    if tree.rgSearchBar.Value() != msg.query {
        slog.Debug("RgResultMsg: stale, discarding", "want", tree.rgSearchBar.Value(), "got", msg.query)
        return nil
    }
    tree.rgMatches = msg.matches
    tree.cursor = 0
    tree.renderIdx = 0
    tree.rebuild()
    return nil
}
```

**Step 2: Add rgSearchTickMsg type and rg launcher to model.go**

Add `rgSearchTickMsg` near the `previewTickMsg` definition (around line 31 of `model.go`):

```go
// rgSearchTickMsg is posted after the rg debounce delay. If inputTime matches
// tree.lastRgInput the query is current and rg is launched.
type rgSearchTickMsg struct {
    panelIdx  int
    inputTime time.Time
}
```

Add the rg launcher as a method on `*model` (add to `model.go` or a new `handle_rg_search.go`). Put it in `model.go` near `startPreviewDebounce`:

```go
// startRgDebounce posts a tick that fires after 300ms. If the user types
// another character before it fires, lastRgInput will have changed and the
// tick is discarded.
func (m *model) startRgDebounce(idx int) tea.Cmd {
    now := time.Now()
    m.treePanels[idx].lastRgInput = now
    return func() tea.Msg {
        time.Sleep(300 * time.Millisecond)
        return rgSearchTickMsg{panelIdx: idx, inputTime: now}
    }
}

// launchRgSearch runs rg in a goroutine and posts back a RgResultMsg.
func (m *model) launchRgSearch(idx int, query string) tea.Cmd {
    root := m.treePanels[idx].root
    return func() tea.Msg {
        cmd := exec.Command("rg", "--files-with-matches", "--no-messages", "--smart-case", "--", query, root)
        out, _ := cmd.Output() // non-zero exit = no matches, not an error we surface
        matches := make(map[string]bool)
        scanner := bufio.NewScanner(strings.NewReader(string(out)))
        for scanner.Scan() {
            if p := strings.TrimSpace(scanner.Text()); p != "" {
                matches[p] = true
            }
        }
        return RgResultMsg{query: query, panelIdx: idx, matches: matches}
    }
}
```

Make sure `bufio`, `strings`, `os/exec` are imported in `model.go` (check existing imports first).

**Step 3: Handle rgSearchTickMsg in Update()**

In `model.go`, in the `Update()` switch, add after the `previewTickMsg` case:

```go
case rgSearchTickMsg:
    tree := &m.treePanels[msg.panelIdx]
    if !msg.inputTime.Equal(tree.lastRgInput) {
        break // superseded by a later keystroke
    }
    query := tree.rgSearchBar.Value()
    if query == "" {
        tree.rgMatches = nil
        tree.cursor = 0
        tree.renderIdx = 0
        tree.rebuild()
        break
    }
    updateCmd = m.launchRgSearch(msg.panelIdx, query)
```

**Step 4: Build**

```bash
make build
```
Expected: clean.

**Step 5: Commit**

```bash
git add src/internal/model_msg.go src/internal/model.go
git commit -m "feat: add RgResultMsg and async rg launcher with debounce"
```

---

### Task 4: Key handling — ctrl+g toggle and rg bar routing

**Files:**
- Modify: `src/internal/handle_tree_panel.go`

**Step 1: Write failing test**

Add to an existing test file (e.g. `model_navigation_test.go`) or create `src/internal/rg_search_test.go`:

```go
package internal

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestCtrlGOpenRgSearchBar(t *testing.T) {
    m := defaultTestModel(t)
    m.setTree1PanelActive()

    // ctrl+g should focus the rg search bar
    m.handleTreePanelKey("ctrl+g", 0)
    assert.True(t, m.treePanels[0].rgSearchBar.Focused())
}

func TestCtrlGCloseRgSearchBar(t *testing.T) {
    m := defaultTestModel(t)
    m.setTree1PanelActive()
    m.treePanels[0].rgSearchBar.Focus()

    // ctrl+g again should blur and clear
    m.handleTreePanelKey("ctrl+g", 0)
    assert.False(t, m.treePanels[0].rgSearchBar.Focused())
    assert.Equal(t, "", m.treePanels[0].rgSearchBar.Value())
    assert.Nil(t, m.treePanels[0].rgMatches)
}

func TestEscClosesRgSearchBar(t *testing.T) {
    m := defaultTestModel(t)
    m.setTree1PanelActive()
    m.treePanels[0].rgSearchBar.Focus()
    m.treePanels[0].rgSearchBar.SetValue("hello")

    cmd := m.handleTreePanelKey("esc", 0)
    assert.Nil(t, cmd)
    assert.False(t, m.treePanels[0].rgSearchBar.Focused())
    assert.Equal(t, "", m.treePanels[0].rgSearchBar.Value())
    assert.Nil(t, m.treePanels[0].rgMatches)
}
```

**Step 2: Run tests — must fail**

```bash
go test ./src/internal/... -run TestCtrlG -v
go test ./src/internal/... -run TestEscClosesRg -v
```
Expected: FAIL — `ctrl+g` not handled.

**Step 3: Add rg bar key handler function**

Add to `handle_tree_panel.go`:

```go
// handleRgSearchBarKey handles input while the rg search bar is focused.
// Returns a cmd (debounce tick for typing, nil for navigation/cancel).
func (m *model) handleRgSearchBarKey(msg string, idx int) tea.Cmd {
    tree := &m.treePanels[idx]
    switch {
    case slices.Contains(common.Hotkeys.ContentSearch, msg), msg == "esc":
        tree.rgSearchBar.Blur()
        tree.rgSearchBar.SetValue("")
        tree.rgMatches = nil
        tree.cursor = 0
        tree.renderIdx = 0
        tree.rebuild()
        return nil
    case slices.Contains(common.Hotkeys.ListUp, msg):
        tree.ListUp(m.mainPanelHeight - 2)
        return m.startPreviewDebounce()
    case slices.Contains(common.Hotkeys.ListDown, msg):
        tree.ListDown(m.mainPanelHeight - 2)
        return m.startPreviewDebounce()
    }
    // All other input (printable chars, backspace, etc.) is forwarded to the
    // text input via updateFilePanelsState; we just schedule a debounce tick.
    return m.startRgDebounce(idx)
}
```

**Step 4: Wire ctrl+g and rg bar routing into handleTreePanelKey**

At the very top of `handleTreePanelKey`, before the main `switch`, add:

```go
// If the rg search bar is focused, route all keys through it.
if tree.rgSearchBar.Focused() {
    return m.handleRgSearchBarKey(msg, idx)
}
```

Then in the main switch, add the `ctrl+g` case (e.g. after `ToggleDetailView`):

```go
case slices.Contains(common.Hotkeys.ContentSearch, msg):
    tree.rgSearchBar.Focus()
    tree.rgSearchBar.Width = tree.width - 8
    return nil
```

**Step 5: Run tests**

```bash
go test ./src/internal/... -run TestCtrlG -v
go test ./src/internal/... -run TestEscClosesRg -v
```
Expected: all PASS.

**Step 6: Build + full test suite**

```bash
make build && go test ./src/internal/...
```
Expected: existing failures only (exiftool, prompt regression, paste/copy).

**Step 7: Commit**

```bash
git add src/internal/handle_tree_panel.go src/internal/rg_search_test.go
git commit -m "feat: ctrl+g toggles rg search bar on active tree panel"
```

---

### Task 5: Route text input updates to rg search bar

**Files:**
- Modify: `src/internal/model.go:460-488` (`updateFilePanelsState`)

**Step 1: Extend updateFilePanelsState**

In `updateFilePanelsState`, add a case for the active tree panel's rg search bar. Add after the `case focusPanel.searchBar.Focused():` block (line ~468):

```go
case m.treePanels[int(m.activeFileArea)].rgSearchBar.Focused():
    idx := int(m.activeFileArea)
    m.treePanels[idx].rgSearchBar, cmd = m.treePanels[idx].rgSearchBar.Update(msg)
```

**Step 2: Build**

```bash
make build
```

**Step 3: Manual smoke test**

```bash
./bin/hpf
```
Press `ctrl+g`, type a pattern — the cursor should blink in the rg bar. Esc should close.

**Step 4: Commit**

```bash
git add src/internal/model.go
git commit -m "feat: route tea.Msg updates to rg search bar text input"
```

---

### Task 6: Render rg search bar in tree panel header

**Files:**
- Modify: `src/internal/tree_panel_render.go:25-49` (header section)

**Step 1: Add rg bar rendering**

In `treePanelRender`, after `r.AddSection()` (line 49), add:

```go
// Rg search bar: shown when focused or has a value.
if tree.rgSearchBar.Focused() || tree.rgSearchBar.Value() != "" {
    tree.rgSearchBar.Width = r.ContentWidth() - 6
    r.AddLines(" " + tree.rgSearchBar.View())
    r.AddSection()
}
```

**Step 2: Build**

```bash
make build
```

**Step 3: Manual end-to-end test**

```bash
./bin/hpf
```
1. Press `ctrl+g` — rg bar appears below path line with "rg: " prompt and blinking cursor
2. Type a word that appears in a file (e.g. `package`) — after ~300ms, tree filters to matching files
3. Press up/down — navigate filtered results normally
4. Press `ctrl+g` or Esc — tree restores, bar disappears
5. If `rg` is not installed: tree shows empty (no crash)

**Step 4: Run full test suite**

```bash
go test ./src/internal/...
```
Expected: same pre-existing failures, no new failures.

**Step 5: Commit**

```bash
git add src/internal/tree_panel_render.go
git commit -m "feat: render rg content search bar in tree panel header"
```
