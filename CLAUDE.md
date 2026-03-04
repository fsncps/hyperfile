# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Hyperfile** (binary: `hpf`) is a terminal file manager written in Go, using the [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework. It uses a fixed 3-panel layout: two independent filesystem tree panels and a file preview panel.

## Common Commands

```bash
# Full development workflow: fmt → lint → test → build
make dev           # or: ./dev.sh

# Individual steps
make build         # Build only; output: ./bin/hpf
make test          # Run unit tests: go test ./...
make install       # Build and install to ~/.local/bin/hpf
make lint          # Run golangci-lint
make testsuite     # Integration tests (requires Python + tmux)
make clean         # Remove ./bin/

# Build directly (CGO_ENABLED=0 is required — make targets set this automatically)
CGO_ENABLED=0 go build -o ./bin/hpf

# Run a single Go test
go test ./src/internal/... -run TestFunctionName

# Integration test options
python testsuite/main.py -t RenameTest    # Run specific test
python testsuite/main.py -d               # Debug mode

# dev.sh flags
./dev.sh --skip-tests      # Build without running unit tests
./dev.sh --testsuite       # Include Python integration tests
./dev.sh -v                # Verbose output
```

## Architecture

### Package Layout

- **`src/cmd/`** — CLI entry point using `urfave/cli`; parses flags like `path-list`, `--fix-hotkeys`, `--fix-config-file`
- **`src/config/`** — Global config paths (`fixed_variable.go`), icon definitions (`icon/`), and app version
- **`src/internal/`** — Core application logic:
  - **`backend/`** — OS-level file operations (no UI imports); uses interfaces for testability
  - **`common/`** — Shared types: config structs (`config_type.go`), theme/style helpers, icon utilities
  - **`ui/`** — Bubble Tea sub-models: `notify/`, `prompt/`, `processbar/`, `metadata/`, `sidebar/`, `rendering/`
  - **`xdnd/`** — Pure-Go X11 XDND drag source protocol (native window DnD)
  - **`model.go`** — Central Bubble Tea model; `Update()` loop, `recalcPanelWidths()`, `treePanelStartX()`, `handleMouseMsg()` → `tea.Cmd`
  - **`model_render.go`** — `View()` rendering; `getPreviewItemPath()` determines what the preview shows
  - **`key_function.go`** — Hotkey dispatch; `mainKey()` routes all file-area input to the active tree panel via `handleTreePanelKey`
  - **`tree_panel.go`** — Tree panel state (`treePanelModel`), expand/collapse logic, multi-select (`selected map[string]bool`, `anchor int`), `HasSelection()`, `SelectedPaths()`
  - **`tree_panel_render.go`** — Tree panel rendering; cursor column shows `⠿` drag handle on cursor/selected rows
  - **`handle_tree_panel.go`** — Tree panel key handlers; `setTree1PanelActive()` / `setTree2PanelActive()`
  - **`handle_file_operations.go`** — High-level file I/O (copy, paste, delete, compress, extract, `dragItems`)
  - **`handle_panel_movement.go`**, **`handle_panel_navigation.go`**, **`handle_panel_up_down.go`** — Split-out key handler groups
  - **`config_function.go`** — Production startup: loads config/theme files, calls `icon.InitIcon` then `icon.ApplyIconTheme`
- **`src/pkg/`** — Independent packages: `file_preview/` (image/text rendering), `string_function/`
- **`src/hyperfile_config/`** — Embedded default configs (TOML) for config, hotkeys, and themes

### Key Architecture Patterns

**Bubble Tea MVC:** `model.go` holds all state. Messages flow through `Update()`, rendering is in `View()`. Sub-panels (sidebar, processbar, prompt, metadata) are nested structs.

**3-Panel Layout:** `treePanels[0]` (left, tree1) and `treePanels[1]` (right, tree2) are two independent `treePanelModel` instances — both navigate files and directories freely. Layout order left-to-right: sidebar | tree1 | tree2 | preview. Width split computed in `recalcPanelWidths()`; `treePanelStartX(idx)` returns the terminal column where each tree panel begins (used for mouse hit-testing). Focus switches with `activeFileArea` (`tree1PanelActive` | `tree2PanelActive`). `fileModel.filePanels` is preserved for file-operation infrastructure but not rendered directly.

**Tree Selection:** `treePanelModel.selected` is a `map[string]bool` of selected paths; `anchor` is the cursor index where shift-select began (-1 = unset). `ClearSelection()` is called on navigation. `HasSelection()` / `SelectedPaths()` are used by file operations and DnD.

**Drag-and-Drop:** `ctrl+d` and mouse left-click on the `⠿` column both call `dragItems()` in `handle_file_operations.go`. It launches the configured `dnd_tool` (default: `dragon --on-top --and-exit`) with the selected or cursor paths. `handleMouseLeftPress(x, y)` in `model.go` hit-tests against `treePanelStartX(idx)` to detect clicks in cols 1–3 of a tree panel.

**Backend Interface:** The `backend` package exposes interfaces so unit tests can test file-operation logic without real filesystem calls. UI code imports `backend`; `backend` never imports UI.

**Embedded Defaults:** Default configs/themes are embedded at compile time via Go's `embed` directive in `src/hyperfile_config/`. User configs at `~/.config/hyperfile/` override them.

**Icon Theming:** `src/config/icon/function.go` has `InitIcon(nerdfont, dirColor)` and `ApplyIconTheme(map[string]string)`. The latter overrides `Color` (never the glyph) in the global `Icons`/`Folders` maps. Both are called in `config_function.go` at startup. Users add an optional `[icon_colors]` section to any theme TOML (e.g. `folder = "#7aa2f7"`, `go = "#00ADD8"`); keys match entries in `icon.Icons` / `icon.Folders`.

**Process Bar:** Background operations (copy, compress, extract) run in goroutines and post progress updates as Bubble Tea `Cmd` messages.

**Hotkeys:** `common.Hotkeys.Confirm` = `['right']` (navigation confirm). `common.Hotkeys.ConfirmTyping` = `['enter']` (modal/typing confirm). These are intentionally separate — don't conflate them.

### Configuration Paths (runtime)

| Purpose | Path |
|---|---|
| Config file | `~/.config/hyperfile/config.toml` |
| Hotkeys | `~/.config/hyperfile/hotkeys.toml` |
| Theme files | `~/.config/hyperfile/theme/<name>.toml` |
| Log file | `~/.local/state/hyperfile/hyperfile.log` |
| Data directory | `~/.local/share/hyperfile/` |

### Testing Notes

- `defaultTestModel()` in `test_utils.go` sets up both tree panels and disables metadata fetching.
- `createNewFilePanel()` and `closeFilePanel()` are kept for test compatibility; `fileModel.filePanels` is used by file-op tests via `getFocusedFilePanel()`.
- `NewTestTeaProgWithEventLoop` runs a real Bubble Tea event loop in a goroutine; use `assert.Eventually` for async effects. `SendKeyDirectly` bypasses the event loop and calls `Update()` directly — the model inside `prog` and `p.m` can diverge if used after the loop has started.
- Pre-existing failures: `ui/metadata` (exiftool not installed), `ui/prompt/TestModel_HandleUpdate` (unrelated regression), `TestPasteItem`/`TestCopy` in `src/internal`.

## Linting

golangci-lint is configured in `.golangci.yaml`. Active complexity linters: `cyclop`, `funlen`, `gocognit`, `gocyclo`, `lll`. Keep new functions short. Add `//nolint: gocyclo,cyclop,funlen // <reason>` only on functions that are inherently large dispatch switches.

`gochecknoglobals` and `gochecknoinits` are enforced. Avoid new global variables and `init()` functions. If a global is truly necessary, add it to one of the designated excluded files: `src/config/fixed_variable.go`, `src/internal/common/style.go`, `src/internal/common/predefined_variable.go`, or `src/internal/common/default_config.go`.

`nolintlint` is strict: directives must name the specific linter(s) and include an explanation (`// reason`), except `funlen`, `gocognit`, and `lll` which allow no explanation. Example: `//nolint:gocyclo // large dispatch switch`.

## PR Standards

- PR titles must follow [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `refactor:`, `chore:`, `docs:`, `test:`, `style:`, `perf:`, `ci:`, `build:`
- Run `go fmt ./...` and `golangci-lint run` before submitting
- No debug logs or TODO comments in commits
