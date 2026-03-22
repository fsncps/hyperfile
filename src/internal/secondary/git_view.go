package secondary

import (
	"time"

	"github.com/yorukot/superfile/src/internal/backend/git"
)

type GitViewModel struct {
	repoRoot     string
	entries      []*git.FileStatus
	cursor       int
	renderIdx    int
	filter       GitFilter
	loading      bool
	lastRefresh  time.Time
	errorMessage string
	width        int
	height       int
}

type GitFilter int

const (
	GitFilterAll GitFilter = iota
	GitFilterModified
	GitFilterStaged
	GitFilterUntracked
)

func NewGitViewModel() *GitViewModel {
	return &GitViewModel{
		filter: GitFilterAll,
	}
}

func (g *GitViewModel) LoadFromRepo(backend *git.Backend, repoRoot string) error {
	g.repoRoot = repoRoot
	g.loading = true
	g.errorMessage = ""

	if backend == nil {
		backend = git.NewBackend()
	}

	if !backend.IsGitRepo(repoRoot) {
		g.loading = false
		g.errorMessage = "Not a git repository"
		return nil
	}

	entries, err := backend.GetAllStatus(repoRoot)
	if err != nil {
		g.loading = false
		g.errorMessage = err.Error()
		return err
	}

	g.entries = make([]*git.FileStatus, 0, len(entries))
	for _, status := range entries {
		g.entries = append(g.entries, status)
	}

	g.loading = false
	g.lastRefresh = time.Now()
	return nil
}

func (g *GitViewModel) EntryCount() int {
	return len(g.filteredEntries())
}

func (g *GitViewModel) filteredEntries() []*git.FileStatus {
	if g.filter == GitFilterAll {
		return g.entries
	}

	var filtered []*git.FileStatus
	for _, entry := range g.entries {
		switch g.filter {
		case GitFilterModified:
			if entry.IsModified() {
				filtered = append(filtered, entry)
			}
		case GitFilterStaged:
			if entry.IsStaged {
				filtered = append(filtered, entry)
			}
		case GitFilterUntracked:
			if entry.IsUntracked() {
				filtered = append(filtered, entry)
			}
		}
	}
	return filtered
}

func (g *GitViewModel) ListUp(visibleH int) {
	entries := g.filteredEntries()
	if len(entries) == 0 {
		return
	}
	if g.cursor > 0 {
		g.cursor--
	}
	if g.renderIdx > g.cursor {
		g.renderIdx = g.cursor
	}
}

func (g *GitViewModel) ListDown(visibleH int) {
	entries := g.filteredEntries()
	if len(entries) == 0 {
		return
	}
	if g.cursor < len(entries)-1 {
		g.cursor++
	}
	if g.cursor >= g.renderIdx+visibleH {
		g.renderIdx = g.cursor - visibleH + 1
	}
}

func (g *GitViewModel) CycleFilter() {
	g.filter = (g.filter + 1) % 4
	g.cursor = 0
	g.renderIdx = 0
}

func (g *GitViewModel) SetDimensions(width, height int) {
	g.width = width
	g.height = height
}

func (g *GitViewModel) GetSelectedPath() string {
	entries := g.filteredEntries()
	if len(entries) == 0 || g.cursor < 0 || g.cursor >= len(entries) {
		return ""
	}
	return entries[g.cursor].Path
}

func (g *GitViewModel) FilterString() string {
	switch g.filter {
	case GitFilterModified:
		return "Modified"
	case GitFilterStaged:
		return "Staged"
	case GitFilterUntracked:
		return "Untracked"
	default:
		return "All"
	}
}

func (g *GitViewModel) Summary() (staged, modified, untracked int) {
	for _, entry := range g.entries {
		if entry.IsStaged {
			staged++
		}
		if entry.IsModified() {
			modified++
		}
		if entry.IsUntracked() {
			untracked++
		}
	}
	return staged, modified, untracked
}
