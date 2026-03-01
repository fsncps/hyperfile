# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Hyperfile** (binary: `hpf`) is a terminal file manager written in Go, using the [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework. It uses a fixed 3-panel layout: a directory-only folder panel, a tree panel, and a file preview panel.

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
- **`src/config/`** — Global config paths (`fixed_variable.go`), icon definitions, and app version
- **`src/internal/`** — Core application logic:
  - **`backend/`** — OS-level file operations (no UI imports); uses interfaces for testability
  - **`common/`** — Shared types: config structs, theme/style helpers
  - **`ui/`** — Bubble Tea sub-models: `notify/`, `prompt/`, `processbar/`, `metadata/`, `sidebar/`, `rendering/`
  - **`model.go`** — Central Bubble Tea model and main `Update` loop; `recalcPanelWidths()` controls all panel widths
  - **`model_render.go`** — `View()` rendering; `getPreviewItemPath()` determines what the preview shows
  - **`key_function.go`** — Hotkey dispatch; `mainKey()` routes to tree panel or folder panel based on `activeFileArea`
  - **`tree_panel.go`** — Tree panel state and logic (`treePanelModel`, `buildTreeNodes`, expand/collapse)
  - **`tree_panel_render.go`** — Tree panel rendering
  - **`handle_tree_panel.go`** — Tree panel key handlers; `setTreePanelActive()` / `setFolderPanelActive()`
  - **`file_operations.go`** — High-level file I/O (copy, paste, delete, compress, extract)
- **`src/pkg/`** — Independent packages: `file_preview/` (image/text rendering), `string_function/`
- **`src/hyperfile_config/`** — Embedded default configs (TOML) for config, hotkeys, and themes

### Key Architecture Patterns

**Bubble Tea MVC:** `model.go` holds all state. Messages flow through `Update()`, rendering is in `View()`. Sub-panels (sidebar, processbar, prompt, metadata) are nested structs.

**3-Panel Layout:** `fileModel.filePanels[0]` is always the folder panel (left, `dirOnly: true` — directories only). `treePanelModel` is the middle panel, rooted at whatever directory the folder panel cursor sits on (synced via `syncTreeRoot()` in `updateModelStateAfterMsg`). Width split: folder ≈ 20%, preview ≈ 35%, tree gets the rest, computed in `recalcPanelWidths()`. Focus switches between panels with `activeFileArea` (`folderPanelActive` | `treePanelActive`).

**Backend Interface:** The `backend` package exposes interfaces so unit tests can test file-operation logic without real filesystem calls. UI code imports `backend`; `backend` never imports UI.

**Embedded Defaults:** Default configs/themes are embedded at compile time via Go's `embed` directive in `src/hyperfile_config/`. User configs at `~/.config/hyperfile/` override them.

**Process Bar:** Background operations (copy, compress, extract) run in goroutines and post progress updates as Bubble Tea `Cmd` messages.

**Hotkeys:** `common.Hotkeys.Confirm` = `['right']` (navigation confirm). `common.Hotkeys.ConfirmTyping` = `['enter']` (modal/typing confirm). These are intentionally separate — don't conflate them.

### Configuration Paths (runtime)

| Purpose | Path |
|---|---|
| Config file | `~/.config/hyperfile/config.toml` |
| Hotkeys | `~/.config/hyperfile/hotkeys.toml` |
| Log file | `~/.local/state/hyperfile/hyperfile.log` |
| Data directory | `~/.local/share/hyperfile/` |

### Testing Notes

- `defaultTestModel()` in `test_utils.go` overrides `dirOnly=false` for all panels so existing tests that navigate to files still work.
- `createNewFilePanel()` and `closeFilePanel()` are kept for test compatibility (no keyboard shortcuts expose them in the real app).
- `NewTestTeaProgWithEventLoop` runs a real Bubble Tea event loop in a goroutine; use `assert.Eventually` for async effects. `SendKeyDirectly` bypasses the event loop and calls `Update()` directly — useful for setup, but the model inside `prog` and `p.m` can diverge if used after the loop has started.
- Pre-existing failures in CI: `ui/metadata` (exiftool not installed), `ui/prompt/TestModel_HandleUpdate` (unrelated regression).

## Linting

golangci-lint is configured in `.golangci.yaml`. Active complexity linters: `cyclop`, `funlen`, `gocognit`, `gocyclo`, `lll`. Keep new functions short. Add `//nolint: gocyclo,cyclop,funlen // <reason>` only on functions that are inherently large dispatch switches.

`gochecknoglobals` and `gochecknoinits` are enforced. Avoid new global variables and `init()` functions. If a global is truly necessary, add it to one of the designated excluded files: `src/config/fixed_variable.go`, `src/internal/common/style.go`, `src/internal/common/predefined_variable.go`, or `src/internal/common/default_config.go`.

`nolintlint` is strict: directives must name the specific linter(s) and include an explanation (`// reason`), except `funlen`, `gocognit`, and `lll` which allow no explanation. Example: `//nolint:gocyclo // large dispatch switch`.

## PR Standards

- PR titles must follow [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `refactor:`, `chore:`, `docs:`, `test:`, `style:`, `perf:`, `ci:`, `build:`
- Run `go fmt ./...` and `golangci-lint run` before submitting
- No debug logs or TODO comments in commits
