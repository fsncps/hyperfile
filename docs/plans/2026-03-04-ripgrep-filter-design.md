# Design: Ripgrep Content Filter (`ctrl+g`)

## Overview

Add an inline content-search bar to the active tree panel, triggered by `ctrl+g`. As the user types, `rg` filters the tree to show only files whose contents match the pattern, plus their ancestor directories. Esc or `ctrl+g` again clears the filter and restores the full tree.

## UX

1. `ctrl+g` — opens rg search bar on the active tree panel (inline, below the path line)
2. User types a pattern — rg runs debounced (~300ms) on each keystroke
3. Tree updates: only files with matching content + their ancestor directories are shown
4. Esc or `ctrl+g` — clears filter, restores full tree, blurs the search bar

The rg command used:
```
rg --files-with-matches --no-messages --smart-case -- <pattern> <root>
```

## Data Model

Two fields added to `treePanelModel`:

```go
rgSearchBar textinput.Model  // the input widget; Focused() = bar is active
rgMatches   map[string]bool  // nil = no filter; non-nil = set of matching absolute paths
```

## Filter Logic

`buildTreeNodes` receives the `rgMatches` set (nil = no filter). When non-nil:

- **File node** shown only if `rgMatches[path]`
- **Dir node** shown only if it is an ancestor of any matched path

Ancestor check: pre-compute `ancestorDirs map[string]bool` from `rgMatches` before traversal — for each match, walk `filepath.Dir` up to root, adding each dir to the set.

## Async Execution

Follows the existing `ModelUpdateMessage` pattern in `model_msg.go`:

1. On each keystroke, post `rgSearchTickMsg{panelIdx int, query string}` sleeping 300ms
2. On tick arrival: if `tree.rgSearchBar.Value() != msg.query`, discard (stale)
3. Otherwise: launch goroutine running `rg`, collect stdout lines into `map[string]bool`, post `RgResultMsg`
4. `RgResultMsg.ApplyToModel`: if query still current, set `rgMatches`, call `rebuild()`

```go
type rgSearchTickMsg struct{ panelIdx int; query string }

type RgResultMsg struct {
    panelIdx int
    query    string
    matches  map[string]bool
}
```

## Hotkey

- `content_search = ['ctrl+g']` in `hotkeys.toml` under file operations section
- `ContentSearch []string` field added to `HotkeysType` in `common/config_type.go`

## Key Handling (when rg bar is focused)

- Printable chars: update bar, post debounce tick
- Backspace: trim bar, post debounce tick (empty query → clear filter)
- Esc / `ctrl+g`: clear `rgMatches`, clear bar value, blur bar, rebuild
- Up/Down: pass through to normal tree navigation (navigate filtered results)
- Enter: no-op (search is live)

## Rendering

- Rg search bar renders in `tree_panel_render.go`, below the panel path line, when `rgSearchBar.Focused()` or `rgSearchBar.Value() != ""`
- Visual: prefix `  rg: ` + `rgSearchBar.View()`
- When filter active but bar not focused: show dim indicator `  ~rg: <value>`

## Files Changed

| File | Change |
|---|---|
| `src/hyperfile_config/hotkeys.toml` | Add `content_search = ['ctrl+g']` |
| `src/internal/common/config_type.go` | Add `ContentSearch []string` to `HotkeysType` |
| `src/internal/tree_panel.go` | Add `rgSearchBar`, `rgMatches` fields; extend `buildTreeNodes` with filter param |
| `src/internal/tree_panel_render.go` | Render rg bar in panel header |
| `src/internal/handle_tree_panel.go` | Handle `ctrl+g`; route keys to rg bar when focused |
| `src/internal/model_msg.go` | Add `RgResultMsg` |
| `src/internal/model.go` | Handle `rgSearchTickMsg` and `RgResultMsg` in `Update()` |
