package internal

import (
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/fsncps/hyperfile/src/internal/common"
)

// fileAreaFocus indicates which file-area panel currently has keyboard focus.
type fileAreaFocus int

const (
	tree1PanelActive fileAreaFocus = iota // left tree (index 0)
	tree2PanelActive                      // right tree (index 1)
)

// treeNode is a single entry in the flattened visible tree list.
type treeNode struct {
	name   string
	path   string
	isDir  bool
	depth  int  // 0 = direct child of root
	isLast bool // last sibling in its parent directory (used for branch-line rendering)
}

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

// treePanelModel holds all state for the middle tree panel.
type treePanelModel struct {
	root       string
	nodes      []treeNode      // flattened visible list rebuilt on change
	cursor     int
	renderIdx  int
	maxDepth   int             // auto-expand depth on rebuild; does not limit manual expansion
	collapsed  map[string]bool // paths manually collapsed (takes priority over depth)
	expanded   map[string]bool // paths manually expanded beyond maxDepth
	selected   map[string]bool // paths currently selected; nil = no selection
	anchor     int             // cursor idx when shift-select began; -1 = unset
	showHidden    bool            // mirrors model.toggleDotFile
	mode          treePanelMode
	detailRoot    string
	detailEntries []detailEntry
	focusType     filePanelFocusType
	open       bool
	width      int
	rgSearchBar textinput.Model
	rgMatches   map[string]bool // nil = no filter; non-nil = set of matching absolute file paths
	lastRgInput time.Time       // timestamp of last rg search bar keystroke (debounce sentinel)
}

func defaultTreePanel(root string) treePanelModel {
	t := treePanelModel{
		root:        root,
		maxDepth:    2,
		collapsed:   make(map[string]bool),
		expanded:    make(map[string]bool),
		anchor:      -1,
		open:        true,
		focusType:   noneFocus,
		rgSearchBar: common.GenerateRgSearchBar(),
	}
	t.nodes = buildTreeNodes(root, t.maxDepth, t.collapsed, t.expanded, t.showHidden, nil)
	return t
}

// buildTreeNodes recursively walks root, honouring collapsed/expanded sets and maxDepth.
// maxDepth controls auto-expansion on rebuild; manually expanded dirs (expanded map)
// are shown regardless of depth. collapsed always takes priority.
// When rgMatches is non-nil only matching files and their ancestor dirs are shown.
// Returns a flat list in display order.
func buildTreeNodes(root string, maxDepth int, collapsed, expanded map[string]bool, showHidden bool, rgMatches map[string]bool) []treeNode {
	nodes := make([]treeNode, 0, 64)
	var rgAncestors map[string]bool
	if rgMatches != nil {
		rgAncestors = buildRgAncestorDirs(rgMatches, root)
	}
	addTreeNodes(&nodes, root, 0, maxDepth, collapsed, expanded, showHidden, rgMatches, rgAncestors)
	return nodes
}

func addTreeNodes(nodes *[]treeNode, dir string, depth, maxDepth int, collapsed, expanded map[string]bool, showHidden bool, rgMatches, rgAncestors map[string]bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		slog.Debug("tree: cannot read dir", "dir", dir, "err", err)
		return
	}
	// Filter hidden entries first so we can correctly identify the last visible sibling.
	visible := make([]os.DirEntry, 0, len(entries))
	for _, e := range entries {
		if len(e.Name()) > 0 && e.Name()[0] == '.' && !showHidden {
			continue
		}
		visible = append(visible, e)
	}
	// Dirs before files; within each group preserve ReadDir's alphabetical order.
	slices.SortStableFunc(visible, func(a, b os.DirEntry) int {
		if a.IsDir() == b.IsDir() {
			return 0
		}
		if a.IsDir() {
			return -1
		}
		return 1
	})
	for i, e := range visible {
		path := filepath.Join(dir, e.Name())
		// rg content filter: skip entries not in the match/ancestor sets.
		if rgMatches != nil {
			if e.IsDir() && !rgAncestors[path] {
				continue
			}
			if !e.IsDir() && !rgMatches[path] {
				continue
			}
		}
		node := treeNode{
			name:   e.Name(),
			path:   path,
			isDir:  e.IsDir(),
			depth:  depth,
			isLast: i == len(visible)-1,
		}
		*nodes = append(*nodes, node)
		if e.IsDir() {
			var shouldExpand bool
			if rgMatches != nil {
				shouldExpand = rgAncestors[path]
			} else {
				shouldExpand = (depth < maxDepth || expanded[path]) && !collapsed[path]
			}
			if shouldExpand {
				addTreeNodes(nodes, path, depth+1, maxDepth, collapsed, expanded, showHidden, rgMatches, rgAncestors)
			}
		}
	}
}

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

// buildRgAncestorDirs returns the set of all ancestor directories (between root
// and the match, exclusive of root) for paths in matches.
func buildRgAncestorDirs(matches map[string]bool, root string) map[string]bool {
	dirs := make(map[string]bool)
	for p := range matches {
		for dir := filepath.Dir(p); dir != root && strings.HasPrefix(dir, root+string(filepath.Separator)); dir = filepath.Dir(dir) {
			dirs[dir] = true
		}
	}
	return dirs
}

// NavigateTo sets the tree root and resets to depth=0, clearing all expansion state.
// Unlike SetRoot it always rebuilds even when root is unchanged, and forces maxDepth=0.
func (t *treePanelModel) NavigateTo(root string) {
	t.ClearSelection()
	t.root = root
	t.maxDepth = 0
	t.cursor = 0
	t.renderIdx = 0
	t.collapsed = make(map[string]bool)
	t.expanded = make(map[string]bool)
	t.rgMatches = nil
	t.rgSearchBar.SetValue("")
	t.rgSearchBar.Blur()
	t.nodes = buildTreeNodes(root, t.maxDepth, t.collapsed, t.expanded, t.showHidden, nil)
}

// SetRoot resets the tree root and rebuilds nodes.
func (t *treePanelModel) SetRoot(root string) {
	if root == t.root {
		return
	}
	t.ClearSelection()
	t.root = root
	t.cursor = 0
	t.renderIdx = 0
	t.collapsed = make(map[string]bool)
	t.expanded = make(map[string]bool)
	t.rgMatches = nil
	t.rgSearchBar.SetValue("")
	t.rgSearchBar.Blur()
	t.nodes = buildTreeNodes(root, t.maxDepth, t.collapsed, t.expanded, t.showHidden, nil)
}

// rebuild regenerates the node list without changing root or depth settings.
func (t *treePanelModel) rebuild() {
	t.nodes = buildTreeNodes(t.root, t.maxDepth, t.collapsed, t.expanded, t.showHidden, t.rgMatches)
	if t.cursor >= len(t.nodes) {
		t.cursor = max(0, len(t.nodes)-1)
	}
	if t.renderIdx > t.cursor {
		t.renderIdx = t.cursor
	}
}

// ToggleNode expands a collapsed dir or collapses an expanded dir at cursor.
func (t *treePanelModel) ToggleNode() {
	if len(t.nodes) == 0 || t.cursor >= len(t.nodes) {
		return
	}
	node := t.nodes[t.cursor]
	if !node.isDir {
		return
	}
	if t.collapsed[node.path] {
		delete(t.collapsed, node.path)
		if t.expanded == nil {
			t.expanded = make(map[string]bool)
		}
		t.expanded[node.path] = true
	} else {
		t.collapsed[node.path] = true
		delete(t.expanded, node.path)
	}
	t.rebuild()
}

// ExpandNode ensures the dir at cursor is expanded regardless of maxDepth.
func (t *treePanelModel) ExpandNode() {
	if len(t.nodes) == 0 || t.cursor >= len(t.nodes) {
		return
	}
	node := t.nodes[t.cursor]
	if !node.isDir {
		return
	}
	delete(t.collapsed, node.path)
	if t.expanded == nil {
		t.expanded = make(map[string]bool)
	}
	t.expanded[node.path] = true
	t.rebuild()
}

// CollapseNode collapses the dir at cursor, or the parent dir if already collapsed / file.
func (t *treePanelModel) CollapseNode() {
	if len(t.nodes) == 0 || t.cursor >= len(t.nodes) {
		return
	}
	node := t.nodes[t.cursor]
	if node.isDir && !t.collapsed[node.path] {
		t.collapsed[node.path] = true
		delete(t.expanded, node.path)
		t.rebuild()
		return
	}
	// Move to and collapse parent dir
	if node.depth > 0 {
		parentPath := filepath.Dir(node.path)
		t.collapsed[parentPath] = true
		delete(t.expanded, parentPath)
		// Find parent position in visible node list
		for i, n := range t.nodes {
			if n.path == parentPath {
				t.cursor = i
				break
			}
		}
		t.rebuild()
	}
}

// ChangeDepth adjusts maxDepth by delta and resets to a clean auto-expanded state.
func (t *treePanelModel) ChangeDepth(delta int) {
	newDepth := t.maxDepth + delta
	if newDepth < 0 {
		newDepth = 0
	}
	t.maxDepth = newDepth
	t.collapsed = make(map[string]bool)
	t.expanded = make(map[string]bool)
	t.rebuild()
}

// RootUp moves the tree root one level up toward the filesystem root.
func (t *treePanelModel) RootUp() {
	parent := filepath.Dir(t.root)
	if parent == t.root {
		return // already at filesystem root
	}
	t.SetRoot(parent)
}

// moveUp moves cursor one step up without wrapping (no selection logic).
func (t *treePanelModel) moveUp() {
	if t.cursor > 0 {
		t.cursor--
		if t.cursor < t.renderIdx {
			t.renderIdx = t.cursor
		}
	}
}

// moveDown moves cursor one step down without wrapping (no selection logic).
func (t *treePanelModel) moveDown(visibleH int) {
	if t.cursor < t.EntryCount()-1 {
		t.cursor++
		if t.cursor >= t.renderIdx+visibleH {
			t.renderIdx++
		}
	}
}

// ListUp moves the cursor up, wrapping to bottom, and clears any selection.
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

// ListDown moves the cursor down, wrapping to top, and clears any selection.
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

// ---- Selection methods

// ClearSelection removes all selected paths and resets the anchor.
func (t *treePanelModel) ClearSelection() {
	t.selected = nil
	t.anchor = -1
}

// HasSelection reports whether any paths are selected.
func (t *treePanelModel) HasSelection() bool {
	return len(t.selected) > 0
}

// SelectedPaths returns a slice of all currently selected paths (unordered).
func (t *treePanelModel) SelectedPaths() []string {
	paths := make([]string, 0, len(t.selected))
	for p := range t.selected {
		paths = append(paths, p)
	}
	return paths
}

// ToggleSelected toggles the selection state of a single path.
func (t *treePanelModel) ToggleSelected(path string) {
	if t.selected == nil {
		t.selected = make(map[string]bool)
	}
	if t.selected[path] {
		delete(t.selected, path)
	} else {
		t.selected[path] = true
	}
}

func (t *treePanelModel) setAnchorIfUnset() {
	if t.anchor == -1 {
		t.anchor = t.cursor
	}
}

func (t *treePanelModel) applyRangeSelection() {
	if t.anchor < 0 || t.anchor >= len(t.nodes) {
		return
	}
	lo, hi := t.anchor, t.cursor
	if lo > hi {
		lo, hi = hi, lo
	}
	t.selected = make(map[string]bool, hi-lo+1)
	for i := lo; i <= hi && i < len(t.nodes); i++ {
		t.selected[t.nodes[i].path] = true
	}
}

// ShiftListUp extends or shrinks the range selection one step up.
func (t *treePanelModel) ShiftListUp(visibleH int) {
	if t.mode == treePanelModeDetail || len(t.nodes) == 0 {
		return
	}
	t.setAnchorIfUnset()
	t.moveUp()
	t.applyRangeSelection()
}

// ShiftListDown extends or shrinks the range selection one step down.
func (t *treePanelModel) ShiftListDown(visibleH int) {
	if t.mode == treePanelModeDetail || len(t.nodes) == 0 {
		return
	}
	t.setAnchorIfUnset()
	t.moveDown(visibleH)
	t.applyRangeSelection()
}

// GetSelectedNode returns a copy of the node at cursor, or nil if empty.
func (t *treePanelModel) GetSelectedNode() *treeNode {
	if len(t.nodes) == 0 || t.cursor >= len(t.nodes) {
		return nil
	}
	n := t.nodes[t.cursor]
	return &n
}

// IsExpanded reports whether path is currently expanded (not in collapsed set).
func (t *treePanelModel) IsExpanded(path string) bool {
	return !t.collapsed[path]
}

// HasChildren reports whether dir at path has at least one visible child.
func (t *treePanelModel) HasChildren(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if len(e.Name()) > 0 && e.Name()[0] != '.' {
			return true
		}
	}
	return false
}

// EntryCount returns the number of navigable entries in the current mode.
// Tree mode → len(nodes); detail mode → len(detailEntries).
func (t *treePanelModel) EntryCount() int {
	if t.mode == treePanelModeDetail {
		return len(t.detailEntries)
	}
	return len(t.nodes)
}
