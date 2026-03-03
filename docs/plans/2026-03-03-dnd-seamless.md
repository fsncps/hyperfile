# Seamless DnD Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Show a `⠿` drag handle in the tree panel on the cursor/selected rows, and launch dragon with `--on-top` when the handle is clicked with the mouse.

**Architecture:** Three focused changes — (1) render a drag-handle char in the existing 2-char cursor column, (2) detect left-press on that column in `handleMouseMsg` and call `dragItems`, (3) simplify `dragItems` to launch `dragon --on-top --and-exit` with no modal on success. The ctrl+d keyboard shortcut continues to work unchanged.

**Tech Stack:** Go stdlib, Bubble Tea v1 mouse events (`msg.String()` / `msg.X` / `msg.Y`), `os/exec`, existing `notify.Model`.

---

### Task 1: Visual drag handle in tree panel render

**Files:**
- Modify: `src/internal/tree_panel_render.go:99-118`

**Step 1: Add dragChar after isSelected is computed (line 99)**

Around line 99–118 in the item-rendering loop, the block currently reads:

```go
isSelected := tree.selected[node.path]
bgColor := common.FilePanelBGColor
// ... clip color logic ...
rendered := common.PrettierName(...)

line := common.FilePanelCursorStyle.Render(cursorChar+" ") +
    common.TreeBranchStyle.Render(branchStr) +
    expandIndicator + " " + rendered
```

Change to:

```go
isSelected := tree.selected[node.path]
dragChar := " "
if i == tree.cursor || isSelected {
    dragChar = "⠿"
}
bgColor := common.FilePanelBGColor
// ... clip color logic unchanged ...
rendered := common.PrettierName(...)

line := common.FilePanelCursorStyle.Render(cursorChar+dragChar) +
    common.TreeBranchStyle.Render(branchStr) +
    expandIndicator + " " + rendered
```

Note: `overhead` at line 93 already counts 2 chars for `cursorChar+" "` — `cursorChar+dragChar` is still 2 chars, so no width math changes.

**Step 2: Build**

```bash
make build
```
Expected: clean build, no errors.

**Step 3: Smoke-test visually**

Run `./bin/hpf`. The cursor row should show `⠿` in the second char of the cursor indicator column. Shift-selected rows should also show `⠿`. All other rows show a space there.

**Step 4: Commit**

```bash
git add src/internal/tree_panel_render.go
git commit -m "feat: show drag handle icon on cursor/selected rows"
```

---

### Task 2: Mouse left-press triggers dragItems

**Files:**
- Modify: `src/internal/model.go` (handleMouseMsg return type + Update call site + two new helpers)

**Step 1: Change handleMouseMsg to return tea.Cmd**

Find the existing function signature:
```go
func (m *model) handleMouseMsg(msg tea.MouseMsg) {
    msgStr := msg.String()
    if msgStr == "wheel up" || msgStr == "wheel down" {
        wheelMainAction(msgStr, m)
    } else {
        slog.Debug("Mouse event of type that is not handled", "msg", msgStr)
    }
}
```

Replace with:
```go
func (m *model) handleMouseMsg(msg tea.MouseMsg) tea.Cmd {
    msgStr := msg.String()
    switch {
    case msgStr == "wheel up" || msgStr == "wheel down":
        wheelMainAction(msgStr, m)
    case msgStr == "left press":
        return m.handleMouseLeftPress(msg.X, msg.Y)
    default:
        slog.Debug("Mouse event of type that is not handled", "msg", msgStr)
    }
    return nil
}
```

**Step 2: Update the call site in Update()**

Find:
```go
case tea.MouseMsg:
    m.handleMouseMsg(msg)
```

Change to:
```go
case tea.MouseMsg:
    inputCmd = m.handleMouseMsg(msg)
```

**Step 3: Add treePanelStartX helper**

Add this method near `recalcPanelWidths()` (around line 195):

```go
// treePanelStartX returns the terminal column where tree panel idx starts (outer left border).
func (m *model) treePanelStartX(idx int) int {
    sidebarOuter := common.Config.SidebarWidth + 2
    if idx == 0 {
        return sidebarOuter
    }
    tree1OuterW := 0
    if m.treePanels[0].open {
        tree1OuterW = m.treePanels[0].width + 2
    }
    return sidebarOuter + tree1OuterW + m.fileModel.filePreview.width
}
```

**Step 4: Add handleMouseLeftPress helper**

Add this method in `model.go` (near the other mouse handling code around line 117):

```go
// handleMouseLeftPress fires dragItems when the user clicks the drag handle
// column (first 3 chars of a tree panel's content area).
func (m *model) handleMouseLeftPress(x, y int) tea.Cmd {
    for idx := range 2 {
        if !m.treePanels[idx].open {
            continue
        }
        start := m.treePanelStartX(idx)
        end := start + m.treePanels[idx].width + 2
        if x < start || x >= end {
            continue
        }
        // Columns 1–3 relative to panel start = left border + cursor + dragChar
        relX := x - start
        if relX < 1 || relX > 3 {
            return nil
        }
        // Found click in drag handle area of panel idx — switch focus and drag
        if idx == 0 {
            m.setTree1PanelActive()
        } else {
            m.setTree2PanelActive()
        }
        return m.dragItems(&m.treePanels[idx])
    }
    return nil
}
```

**Step 5: Build**

```bash
make build
```
Expected: clean build.

**Step 6: Smoke-test**

Run `./bin/hpf`. Click the `⠿` icon on any cursor row. Dragon should appear (with `--on-top`) and the TUI stays alive. If dragon isn't installed, the "DnD tool not found" notify should appear.

**Step 7: Commit**

```bash
git add src/internal/model.go
git commit -m "feat: trigger drag on mouse click of ⠿ handle column"
```

---

### Task 3: Simplify dragItems — on-top, no success modal

**Files:**
- Modify: `src/internal/handle_file_operations.go` (dragItems function at end of file)

**Step 1: Replace dragItems**

Find the full existing `dragItems` function and replace it with:

```go
// ---- Drag and drop ----

// dragItems launches the configured dnd_tool with the selected or cursor file(s),
// allowing them to be dragged into external X11/Wayland applications.
// The TUI is not suspended; the tool runs alongside it.
func (m *model) dragItems(tree *treePanelModel) tea.Cmd {
    tool := common.Config.DNDTool
    if tool == "" {
        tool = "dragon"
    }

    var paths []string
    if tree.HasSelection() {
        paths = tree.SelectedPaths()
    } else {
        node := tree.GetSelectedNode()
        if node == nil {
            return nil
        }
        paths = []string{node.path}
    }

    if _, err := exec.LookPath(tool); err != nil {
        reqID := m.ioReqCnt
        m.ioReqCnt++
        return func() tea.Msg {
            return NewNotifyModalMsg(
                notify.New(true, "DnD tool not found",
                    "Install '"+tool+"' or set dnd_tool in config.toml", notify.NoAction),
                reqID,
            )
        }
    }

    var args []string
    if tool == "dragon" {
        args = append([]string{"--on-top", "--and-exit"}, paths...)
    } else {
        args = paths
    }

    cmd := exec.Command(tool, args...)
    if err := cmd.Start(); err != nil {
        slog.Error("dragItems: failed to start dnd tool", "tool", tool, "error", err)
        reqID := m.ioReqCnt
        m.ioReqCnt++
        return func() tea.Msg {
            return NewNotifyModalMsg(
                notify.New(true, "Drag failed", err.Error(), notify.NoAction),
                reqID,
            )
        }
    }
    return nil
}
```

Key changes vs previous version:
- `--on-top` added to dragon args (dragon window floats above terminal)
- `--and-exit` stays (dragon closes after drop)
- Success case returns `nil` — the `⠿` icon is the visual feedback; no modal needed
- Error cases still show notify

**Step 2: Build**

```bash
make build
```
Expected: clean build.

**Step 3: Smoke-test**

Run `./bin/hpf`. Press ctrl+d or click `⠿`. Dragon window should appear on top of the terminal window, ready to drag.

**Step 4: Commit**

```bash
git add src/internal/handle_file_operations.go
git commit -m "fix: dragon --on-top, remove success modal, silent on successful drag start"
```
