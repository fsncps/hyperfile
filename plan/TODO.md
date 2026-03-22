# Project TODO

Tracking ordered goals, priorities, and progress for hyperfile.

## Legend

- `►` — In Progress
- `○` — Planned (Next Up)
- `✓` — Completed
- `⚠` — Blocked/Needs Discussion

---

## Active

### Secondary Panel Modes — Panel Hierarchy Refactor
`plan/secondary-panel-modes/plan.md` | Status: In Progress

Refactor primary/secondary panels into distinct types with clear roles. Primary is always a file tree; secondary is context-sensitive.

**Phase 1: Structural Refactor**
- [x] Rename `treePanels [2]treePanelModel` to `primaryPanel`/`secondaryPanel`
- [x] Update all index-based references to named fields
- [x] Add `SecondaryMode` enum and `SecondaryPanel` container
- [x] Create mode-switching infrastructure (hotkeys `alt+t/d/g/u`)
- [x] Add `secondaryPanelRender()` dispatch function
- [ ] Integrate tree panels into `View()` (separate effort)

**Phase 2: Mode Implementations**
- [x] **Detail Mode** — `detailViewModel`, `renderDetailPanel()`, `buildDetailEntries()`
- [x] **Git Mode** — `GitViewModel` with `LoadFromRepo()`, status filtering, render
- [x] **Disk Usage Mode** — `DiskUsageModel` with background calculation, render
- [ ] **Search Results Mode** — Show `content_search` results as flat list
- [ ] **Archive Mode** — Auto-list `.zip`/`.tar` contents

**Phase 3: Polish**
- [ ] Mode bar visual indicator
- [ ] State persistence when switching modes
- [ ] Auto-switch for archives
- [ ] Integration tests for mode switching

---

### Git Integration — Phase 1: Status & Visibility
`plan/git/plan_1.md` | Started: 2026-03-18

- [ ] **Phase 1a: Core Git Backend**
  - [ ] Add `go-git` dependency
  - [ ] Implement `GitBackend` with status parsing (`src/internal/backend/git/`)
  - [ ] Implement status caching
  - [ ] Unit tests for status parsing

- [ ] **Phase 1b: Git Column**
  - [ ] Modify `treePanelModel` to store git status
  - [ ] Update `treePanelRender` to show git column
  - [ ] Add status indicators and colors
  - [ ] Wire refresh triggers

- [ ] **Phase 1c: Git Footer Context**
  - [ ] Add git info to footer rendering
  - [ ] Implement toggle for .gitignore files
  - [ ] Implement toggle for tracked-only view
  - [ ] Add hotkeys for git operations

- [ ] **Phase 1d: Secondary Panel Git Mode**
  - [ ] Add `viewModeGit` to view mode enum
  - [ ] Implement git log view with branches/commits
  - [ ] Implement commit detail view
  - [ ] Navigation between git columns

- [ ] **Phase 1e: Polish & Integration**
  - [ ] Configuration options
  - [ ] Performance optimization
  - [ ] Error handling for non-git directories
  - [ ] Documentation

---

## Planned

### Performance: Parallelize Disk Usage Calculation

*Depends on Secondary Panel Modes completion*

Research and implement faster directory size calculation:

- [ ] **Research:** Evaluate existing Go libraries for recursive directory sizing
  - `github.com/dundee/gdu` — parallel disk usage analyzer
  - `github.com/rclone/rclone` — has efficient size calc
  - Custom implementation with `sync.WaitGroup` worker pool
- [ ] **Benchmark:** Compare serial vs parallel on typical project directories
- [ ] **Implement:** Add parallel calculation with configurable worker count
- [ ] **Handle:** Edge cases (permission errors, NFS mounts, symlinks)

### Git Integration — Phase 2: Write Operations
*Depends on Phase 1 completion*

- [ ] Stage/unstage files
- [ ] Commit with message
- [ ] Basic stash

### Git Integration — Phase 3: Advanced Git

- [ ] Branch management
- [ ] Merge/rebase
- [ ] Worktree support
- [ ] Git blame in preview

### Git Integration — Phase 4: Remote Integration

- [ ] Fetch/pull
- [ ] Push (with credential handling)
- [ ] Remote management

### Git Integration — Phase 5: Platform Integration

- [ ] GitHub/GitLab API for issues
- [ ] PR viewer in preview panel
- [ ] Actions/Workflows overview
- [ ] `gh` CLI integration

---

## Needs Discussion

> Items requiring design decisions before planning

- [ ] **Git Integration dependency:** Phase 1d (Secondary Panel Git Mode) overlaps with Secondary Panel Modes Phase 2. Consider coordinating or making Git Mode depend on PanelModes refactor.
- [ ] **Hotkey conflicts:** Git hotkey `g` prefix may conflict with navigation. Use `G` or `C-g`?
- [ ] **Column vs inline:** Separate git column vs integrated icons in filename?
- [ ] **Submodule handling:** Priority in Phase 1 vs later phases?

---

## Completed

| Feature | Plan | Date |
|---------|------|------|
| Log Panel | `docs/plans/2026-03-08-log-panel.md` | 2026-03-08 |
| Cursor Highlight Style | `docs/plans/2026-03-07-cursor-highlight-style.md` | 2026-03-07 |
| Detail View Mode | `docs/plans/2026-03-04-detail-view.md` | 2026-03-04 |
| Seamless DnD | `docs/plans/2026-03-03-dnd-seamless.md` | 2026-03-03 |
| External DnD | `docs/plans/2026-03-03-dnd-drag.md` | 2026-03-03 |
| DnD Design | `docs/plans/2026-03-03-dnd-drag-design.md` | 2026-03-03 |

---

## How to Add a New Feature

1. Create plan file: `plan/<feature>/<feature-name>.md`
2. Add to **Active** or **Planned** section above
3. Use plan template (see `plan/TEMPLATE.md`)
4. Upon completion, move plan to `docs/plans/YYYY-MM-DD-<name>.md`
5. Update this TODO.md

---

## Quick Links

- [AGENTS.md](../AGENTS.md) — Coding agent guidelines
- [CONTRIBUTING.md](../CONTRIBUTING.md) — Contribution guidelines
- [Active Plan: Git Integration](git/plan_1.md)