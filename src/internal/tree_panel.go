package internal

import (
	"log/slog"
	"os"
	"path/filepath"
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

// treePanelModel holds all state for the middle tree panel.
type treePanelModel struct {
	root       string
	nodes      []treeNode      // flattened visible list rebuilt on change
	cursor     int
	renderIdx  int
	maxDepth   int             // auto-expand depth; 0 means root children only
	collapsed  map[string]bool // paths manually collapsed by user
	showHidden bool            // mirrors model.toggleDotFile
	focusType  filePanelFocusType
	open       bool
	width      int
}

func defaultTreePanel(root string) treePanelModel {
	t := treePanelModel{
		root:      root,
		maxDepth:  2,
		collapsed: make(map[string]bool),
		open:      true,
		focusType: noneFocus,
	}
	t.nodes = buildTreeNodes(root, t.maxDepth, t.collapsed, t.showHidden)
	return t
}

// buildTreeNodes recursively walks root up to maxDepth, honouring collapsed set.
// Returns a flat list in display order.
func buildTreeNodes(root string, maxDepth int, collapsed map[string]bool, showHidden bool) []treeNode {
	nodes := make([]treeNode, 0, 64)
	addTreeNodes(&nodes, root, 0, maxDepth, collapsed, showHidden)
	return nodes
}

func addTreeNodes(nodes *[]treeNode, dir string, depth, maxDepth int, collapsed map[string]bool, showHidden bool) {
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
	for i, e := range visible {
		path := filepath.Join(dir, e.Name())
		node := treeNode{
			name:   e.Name(),
			path:   path,
			isDir:  e.IsDir(),
			depth:  depth,
			isLast: i == len(visible)-1,
		}
		*nodes = append(*nodes, node)
		if e.IsDir() && depth < maxDepth && !collapsed[path] {
			addTreeNodes(nodes, path, depth+1, maxDepth, collapsed, showHidden)
		}
	}
}

// SetRoot resets the tree root and rebuilds nodes.
func (t *treePanelModel) SetRoot(root string) {
	if root == t.root {
		return
	}
	t.root = root
	t.cursor = 0
	t.renderIdx = 0
	t.collapsed = make(map[string]bool)
	t.nodes = buildTreeNodes(root, t.maxDepth, t.collapsed, t.showHidden)
}

// rebuild regenerates the node list without changing root or depth settings.
func (t *treePanelModel) rebuild() {
	t.nodes = buildTreeNodes(t.root, t.maxDepth, t.collapsed, t.showHidden)
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
	} else {
		t.collapsed[node.path] = true
	}
	t.rebuild()
}

// ExpandNode ensures the dir at cursor is expanded.
func (t *treePanelModel) ExpandNode() {
	if len(t.nodes) == 0 || t.cursor >= len(t.nodes) {
		return
	}
	node := t.nodes[t.cursor]
	if !node.isDir {
		return
	}
	delete(t.collapsed, node.path)
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
		t.rebuild()
		return
	}
	// Move to and collapse parent dir
	if node.depth > 0 {
		parentPath := filepath.Dir(node.path)
		t.collapsed[parentPath] = true
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

// ChangeDepth adjusts maxDepth by delta and rebuilds with reset collapses.
func (t *treePanelModel) ChangeDepth(delta int) {
	newDepth := t.maxDepth + delta
	if newDepth < 0 {
		newDepth = 0
	}
	t.maxDepth = newDepth
	t.collapsed = make(map[string]bool)
	t.rebuild()
}

// ListUp moves the cursor up, wrapping to bottom.
func (t *treePanelModel) ListUp(visibleHeight int) {
	if len(t.nodes) == 0 {
		return
	}
	if t.cursor > 0 {
		t.cursor--
		if t.cursor < t.renderIdx {
			t.renderIdx--
		}
	} else {
		t.cursor = len(t.nodes) - 1
		maxRender := len(t.nodes) - visibleHeight
		if maxRender < 0 {
			maxRender = 0
		}
		t.renderIdx = maxRender
	}
}

// ListDown moves the cursor down, wrapping to top.
func (t *treePanelModel) ListDown(visibleHeight int) {
	if len(t.nodes) == 0 {
		return
	}
	if t.cursor < len(t.nodes)-1 {
		t.cursor++
		if t.cursor >= t.renderIdx+visibleHeight {
			t.renderIdx++
		}
	} else {
		t.cursor = 0
		t.renderIdx = 0
	}
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
