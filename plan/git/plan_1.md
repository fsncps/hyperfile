# Git Integration Plan - Phase 1: Status & Visibility

**Date:** 2026-03-18
**Status:** Draft
**Scope:** Read-only git status, git column, footer info, gitignore toggle

---

## Executive Summary

Integrate git functionality into hyperfile to provide real-time repository awareness. Phase 1 focuses on **display and visibility** - showing git status without modifying repository state. This establishes the foundation for future write operations.

---

## Architecture Overview

### Current Panel Layout

```
┌─────────────────────────────────────────────────────────────────┐
│                         Header (optional)                        │
├────────┬───────────────────┬───────────────────┬───────────────┤
│        │                   │                   │               │
│ Sidebar │   Primary Panel   │  Secondary Panel  │   Preview     │
│        │   (file tree)      │  (file tree or    │   Panel       │
│ pinned │                   │   detail view)    │               │
│ dirs   │                   │                   │               │
│        │                   │                   │               │
├────────┴───────────────────┴───────────────────┴───────────────┤
│                         Footer (process/metadata)              │
└─────────────────────────────────────────────────────────────────┘
```

### Proposed Addition: Git Column

```
┌─────────────────────────────────────────────────────────────────┐
│  d:2 ~/projects/myrepo                          [main ✓] │ 3 files │ ← git status in header
├────┬───┬───────────────────────┬───────────────┬───────────┤
│    │ G │  Primary Panel        │  Secondary    │  Preview  │
│ S  │ i │  (file tree)          │   Panel       │  Panel    │
│ i  │ t │                       │               │           │
│ d  │ C │  name  size  date     │               │           │
│ e  │ o │  ─────────────────    │               │           │
│ b  │ l │  src/                 │               │           │
│ a  │ u │   ├─ main.go    ✓     │               │           │
│ r  │ m │   ├─ mod.go     M     │               │           │
│    │ n │   └─ test.go    ??    │               │           │
├────┴───┴───────────────────────┴───────────────┴───────────┤
│  Git: main ±3 ↓2 ↑1 | .gitignore: 47 files hidden          │ ← git footer context
└─────────────────────────────────────────────────────────────────┘
```

**Git Column (3-4 chars):**
- `✓` - clean tracked file
- `M` - modified tracked  
- `A` - added (staged)
- `D` - deleted
- `??` - untracked
- `!!` - ignored

---

## Components

### 1. Git Backend Package (`src/internal/backend/git/`)

```
src/internal/backend/git/
├── git.go           # Core git operations using go-git
├── status.go        # Repository status parsing
├── ignore.go        # .gitignore handling
├── branch.go        # Branch information
├── remote.go        # Remote tracking info
└── cache.go         # Status caching for performance
```

**Key Types:**

```go
// FileStatus represents git status for a single file
type FileStatus struct {
    Path     string
    Staging  StatusCode // from git status --porcelain
    Worktree StatusCode
    IsTracked bool
    IsIgnored bool
}

type StatusCode rune // ' ', 'M', 'A', 'D', 'R', 'C', '?', '!', etc.

// RepoInfo aggregates repository-level information
type RepoInfo struct {
    Root         string    // .git directory parent
    Head         string    // current branch or commit
    HeadRef      string    // symbolic ref (branch name) or "HEAD"
    Upstream     string    // remote tracking branch, if any
    Ahead        int       // commits ahead of upstream
    Behind       int       // commits behind upstream
    Dirty        bool      // any uncommitted changes
    Staged       int       // files in staging area
    Untracked    int       // untracked files
    Ignored      int       // ignored files
    LastFetched  time.Time // approximate last fetch time
}

// GitBackend provides all git operations
type GitBackend struct {
    cache *StatusCache
}
```

### 2. Git Column in Tree Panel

**Integration Point:** `src/internal/tree_panel.go` and `src/internal/tree_panel_render.go`

The git column sits between the sidebar and the file tree, showing status at a glance:

```go
// TreeModel extension
type treePanelModel struct {
    // ... existing fields
    gitStatus map[string]git.FileStatus  // path -> status, nil if not in repo
    gitRoot   string                      // empty if not in repo
}
```

**Rendering Layout:**
- Column width: 3 characters (2 for status + 1 spacing)
- Status indicator aligned left
- Color coding:
  - Clean tracked: default
  - Modified: yellow
  - Staged: green
  - Untracked: dim
  - Ignored: dimmed with special indicator

### 3. Footer Git Context

**Integration Point:** `src/internal/model_render.go`

When inside a git repository, the footer shows context:

```
┌─────────────────────────────────────────────────────────────────┐
│ Git: main ✓ | ↑0 ↓0 | tracked: 47 | modified: 3 | untracked: 5 │
│ .gitignore: hiding 12 files | .hidden files visible (.)          │
└─────────────────────────────────────────────────────────────────┘
```

Options to toggle:
- `.` - toggle hidden files (existing)
- `I` - toggle .gitignore filtering (new)
- `T` - toggle showing only tracked files (new)

### 4. Secondary Panel Git Mode

**New View Mode:** `viewModeGit` (alongside tree/detail modes)

```
┌─────────────────────────────────────────────────────────────────┐
│ Git Log (main ↔ feature/auth)                                  │
├───────────────┬───────────────────┬─────────────────────────────┤
│ Branches      │ Commits          │ Commit Detail              │
│ ───────────   │ ───────────────   │ ─────────────────────────  │
│ * main        │ * a1b2c3 Fix bug │ Author: John Doe          │
│   feature/x   │   y4z5d6 Add feat│ Date: 2024-03-18 10:30    │
│   hotfix/y    │   ...            │ Message: Fix bug in auth    │
│               │                  │                            │
│               │                  │ Files changed:            │
│               │                  │  M src/auth.go            │
│               │                  │  A src/auth_test.go       │
│               │                  │  D old/auth.go            │
└───────────────┴───────────────────┴─────────────────────────────┘
```

### 5. Configuration Updates

**New hotkey entries in config:**

```toml
# hotkeys.toml - new section
[git]
toggle_gitignore_filter = ["I"]      # toggle showing ignored files
toggle_tracked_only = ["T"]          # toggle showing only tracked files
git_status_refresh = ["g", "F5"]     # force refresh git status
open_git_log = ["g", "l"]            # open git log in secondary panel
open_git_blame = ["g", "b"]          # open git blame for current file
```

**New config options:**

```toml
[git]
enabled = true
column_visible = true
auto_refresh_interval = 30  # seconds
refresh_on_focus = true     # refresh when panel gains focus
show_ignored_by_default = false
show_untracked_by_default = true
```

---

## Data Flow

### Status Refresh Pipeline

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  File System │────▶│  GitBackend  │────▶│   StatusCache │
│   Change     │     │  (go-git)     │     │   (30s TTL)   │
└──────────────┘     └──────────────┘     └──────────────┘
                            │
                            ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   TreePanel  │◀────│    Model     │◀────│  RepoInfo    │
│   Render     │     │   Update     │     │  Aggregate   │
└──────────────┘     └──────────────┘     └──────────────┘
```

**Refresh Triggers:**
1. Timer-based (configurable interval)
2. Focus gain (when switching to tree panel)
3. Manual refresh (hotkey)
4. After write operations (future: commit, stash)

### Performance Strategy

**Problem:** `git status` on large repos (linux kernel: 70k+ files) is slow.

**Solution: Multi-tier caching:**

```go
type StatusCache struct {
    mu            sync.RWMutex
    repoStatus    map[string]*RepoInfo      // repo_root -> status
    fileStatus    map[string]git.FileStatus // file_path -> status
    lastRefresh   map[string]time.Time       // repo_root -> last scan
    gitignore     map[string][]string        // repo_root -> patterns
    refreshTicker *time.Ticker
}
```

**Optimization Techniques:**
1. **Incremental updates** - only rescan changed directories
2. **Background refresh** - don't block UI
3. **Lazy loading** - load status only for visible files
4. **Watch-based** - future: use fsnotify for .git changes

---

## Implementation Phases

### Phase 1a: Core Git Backend (Week 1)

1. Add `go-git` dependency
2. Implement `GitBackend` with status parsing
3. Implement status caching
4. Unit tests for status parsing

**Deliverable:** Working `git status` parsing with tests

### Phase 1b: Git Column (Week 2)

1. Modify `treePanelModel` to store git status
2. Update `treePanelRender` to show git column
3. Add status indicators and colors
4. Wire refresh triggers

**Deliverable:** Git column showing file status in primary panel

### Phase 1c: Git Footer Context (Week 2)

1. Add git info to footer rendering
2. Implement toggle for .gitignore files
3. Implement toggle for tracked-only view
4. Add hotkeys for git operations

**Deliverable:** Contextual git information in footer

### Phase 1d: Secondary Panel Git Mode (Week 3)

1. Add `viewModeGit` to view mode enum
2. Implement git log view with branches/commits
3. Implement commit detail view
4. Navigation between git columns

**Deliverable:** Git log browsing in secondary panel

### Phase 1e: Polish & Integration (Week 3)

1. Configuration options
2. Performance optimization
3. Error handling for non-git directories
4. Documentation

**Deliverable:** Complete Phase 1 implementation

---

## Technical Decisions

### Library Choice: go-git v6

**Rationale:**
- Pure Go (no CGo) - matches project architecture
- Full status, log, diff, blame support
- Active maintenance (7.3k stars)
- Easy cross-compilation for all platforms
- Sufficient for read-only operations

**Alternative considered:** `git2go` - rejected due to CGo complexity

### Status Caching Strategy

**Decision:** In-memory cache with 30-second TTL, background refresh

**Rationale:**
- Fresh enough for interactive use
- Doesn't block UI on large repos
- Configurable for power users

**Alternative considered:** Real-time via fsnotify - too complex for Phase 1

### Git Column Width

**Decision:** 3 characters fixed width

**Rationale:**
- 2 chars for porcelain status code (e.g., `MM`, ` M`, `??`)
- 1 char for spacing
- Compact but informative

---

## Testing Strategy

### Unit Tests

```go
func TestParseGitStatus(t *testing.T) {
    tests := []struct {
        input    string
        expected []FileStatus
    }{
        {" M file.txt\n?? new.txt\n", []FileStatus{
            {Path: "file.txt", Staging: ' ', Worktree: 'M'},
            {Path: "new.txt", Staging: '?', Worktree: '?'},
        }},
        // ... more cases
    }
    // ...
}

func TestStatusCache_Refresh(t *testing.T) {
    // Test cache expiration and refresh
}

func TestGitignorePattern_Matching(t *testing.T) {
    // Test .gitignore pattern parsing and matching
}
```

### Integration Tests

```go
func TestGitBackend_RealRepo(t *testing.T) {
    // Create temp repo with git init
    // Add, modify, delete files
    // Verify status parsing
}
```

### Performance Benchmarks

```go
func BenchmarkGitStatus_LargeRepo(b *testing.B) {
    // Benchmark status on repos with 10k, 50k, 100k files
}
```

---

## Edge Cases & Error Handling

### Not in a Git Repository

- Git column hidden
- Footer shows "Not in a git repository" or nothing
- No git-related hotkeys active

### Detached HEAD State

- Show commit hash in footer instead of branch name
- Indicate "(detached)" visually

### Submodules

- Initial: treat submodule directories as tracked files showing commit
- Future: recurse into submodules (Phase 2)

### Corrupt .git Directory

- Graceful degradation: show error in footer
- Continue operating as non-git directory

### Large Repository Performance

- Show partial status with "calculating..." indicator
- Background thread for full status
- Configurable timeout for status commands

### .gitignore Parsing

- Use go-git's builtin ignore patterns
- Cache parsed patterns per repository
- Handle nested .gitignore files

---

## Future Phases (Out of Scope)

### Phase 2: Write Operations
- Stage/unstage files
- Commit with message
- Basic stash

### Phase 3: Advanced Git
- Branch management
- Merge/rebase
- Worktree support
- Git blame in preview

### Phase 4: Remote Integration
- Fetch/pull
- Push (with credential handling)
- Remote management

### Phase 5: Platform Integration
- GitHub/GitLab API for issues
- PR viewer in preview panel
- Actions/Workflows overview
- `gh` CLI integration

---

## Questions for Clarification

1. **Column vs. Inline indicators:** Should git status be a separate column (as designed) or integrated into the filename column with icons?

2. **Preview panel integration:** Should git blame be shown in the preview panel for selected files? (Phase 1d or Phase 2?)

3. **Hotkey conflicts:** The proposed `g` prefix may conflict with existing navigation. Should git hotkeys use a different prefix like `G` (shift-g) or `C-g`?

4. **Secondary panel mode switching:** How should users switch between file tree and git modes? (dedicated hotkey vs. mode cycle)

5. **Submodule handling priority:** How important is submodule support in Phase 1?

---

## Success Metrics

- [ ] Git status displays correctly for tracked/modified/untracked files
- [ ] Status refreshes within 500ms for repos under 10k files
- [ ] Status refreshes within 2s for repos under 50k files
- [ ] Toggle hotkeys work correctly
- [ ] Non-git directories work normally with no errors
- [ ] Memory overhead under 50MB for 50k file repo
- [ ] All existing tests pass
- [ ] New code has >80% test coverage