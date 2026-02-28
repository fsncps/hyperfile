# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Superfile** (binary: `spf`) is a modern terminal file manager written in Go, using the [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework.

## Common Commands

```bash
# Full development workflow: fmt → lint → test → build
make dev           # or: ./dev.sh

# Individual steps
make build         # Build only (skips tests); output: ./bin/hpf
make test          # Run unit tests: go test ./...
make lint          # Run golangci-lint
make testsuite     # Integration tests (requires Python + tmux)
make clean         # Remove ./bin/

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
  - **`model.go`** — Central Bubble Tea model and main `Update` loop
  - **`model_render.go`** — `View()` rendering logic
  - **`key_function.go`** — Hotkey dispatch
  - **`file_operations.go`** — High-level file I/O (copy, paste, delete, compress, extract)
- **`src/pkg/`** — Independent packages: `file_preview/` (image/text rendering), `string_function/`
- **`src/superfile_config/`** — Embedded default configs (TOML) for config, hotkeys, and themes
- **`testsuite/`** — Python integration tests using `libtmux` to drive the TUI

### Key Architecture Patterns

**Bubble Tea MVC:** The app follows strict Bubble Tea conventions. `model.go` holds all state. Messages flow through `Update()`, rendering is in `View()`. Sub-panels (sidebar, processbar, prompt, metadata) are nested structs with their own `Update`/`View` methods.

**Backend Interface:** The `backend` package exposes interfaces so unit tests can test file-operation logic without real filesystem calls. UI code imports `backend`; `backend` never imports UI.

**Embedded Defaults:** Default configs/themes are embedded at compile time via Go's `embed` directive in `src/superfile_config/`. User configs at `~/.config/superfile/` override them.

**Process Bar:** Background operations (copy, compress, extract) run in goroutines and post progress updates as Bubble Tea `Cmd` messages, displayed in the process bar panel.

### Configuration Paths (runtime)

| Purpose | Path |
|---|---|
| Config file | `~/.config/superfile/config.toml` |
| Hotkeys | `~/.config/superfile/hotkeys.toml` |
| Log file | `~/.local/state/superfile/superfile.log` |
| Data directory | `~/.local/share/superfile/` |

## Linting

golangci-lint is configured in `.golangci.yaml`. Active complexity linters include `cyclop`, `funlen`, `gocognit`, `gocyclo`, and `lll` (line length). New functions should be kept short and focused to stay within these limits.

## PR Standards

- PR titles must follow [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `refactor:`, `chore:`, `docs:`, `test:`, `style:`, `perf:`, `ci:`, `build:`
- Run `go fmt ./...` and `golangci-lint run` before submitting
- No debug logs or TODO comments in commits
