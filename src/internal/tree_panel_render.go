package internal

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/term/ansi"
	"github.com/fsncps/hyperfile/src/config/icon"
	"github.com/fsncps/hyperfile/src/internal/common"
	"github.com/fsncps/hyperfile/src/internal/ui"
)

// treePanelRender renders the tree panel at the given index (0=left, 1=right).
// Returns an empty string when the panel is closed.
func (m *model) treePanelRender(idx int) string {
	tree := &m.treePanels[idx]
	if !tree.open {
		return ""
	}
	focused := m.focusPanel == nonePanelFocus && m.activeFileArea == fileAreaFocus(idx)
	r := ui.FilePanelRenderer(m.mainPanelHeight+2, tree.width+2, focused)

	// Top bar: path left-aligned, depth right-aligned in the same header row.
	depthStr := "d:" + strconv.Itoa(tree.maxDepth)
	if tree.HasSelection() {
		depthStr += fmt.Sprintf(" [%d]", len(tree.selected))
	}
	iconPart := common.FilePanelTopDirectoryIcon
	iconW := ansi.StringWidth(iconPart)
	depthW := len(depthStr) // ASCII-safe
	cw := r.ContentWidth()
	pathAvail := cw - iconW - depthW - 1 // -1 for minimum gap
	if pathAvail < 4 {
		pathAvail = 4
	}
	truncatedRoot := common.TruncateTextBeginning(tree.root, pathAvail, "...")
	pad := max(1, cw-iconW-ansi.StringWidth(truncatedRoot)-depthW)
	headerLine := iconPart +
		common.FilePanelTopPathStyle.Render(truncatedRoot) +
		strings.Repeat(" ", pad) +
		common.FilePanelStyle.Render(depthStr)
	r.AddLines(headerLine)
	r.AddSection()

	if len(tree.nodes) == 0 {
		r.AddLines(common.FilePanelNoneText)
		return r.Render()
	}

	// Build clipboard set for highlighting copied/cut items.
	clipSet := make(map[string]bool, len(m.copyItems.items))
	for _, p := range m.copyItems.items {
		clipSet[p] = true
	}
	const clipCopyBG = lipgloss.Color("#0A1928") // faint blue
	const clipCutBG = lipgloss.Color("#281400")  // faint orange

	// One fewer overhead row now (depth merged into header), so +1 visible nodes.
	visibleH := m.mainPanelHeight - 2
	end := min(tree.renderIdx+visibleH, len(tree.nodes))

	for i := tree.renderIdx; i < end; i++ {
		node := tree.nodes[i]

		// Cursor indicator
		cursorChar := " "
		if i == tree.cursor {
			cursorChar = icon.Cursor
		}

		// Branch prefix: ancestor continuation lines + own branch character
		branchStr := treeNodeBranchPrefix(tree.nodes, i)

		// Expand/collapse indicator for directories
		var expandIndicator string
		if node.isDir {
			hasKids := tree.HasChildren(node.path)
			if hasKids && tree.IsExpanded(node.path) && node.depth < tree.maxDepth {
				expandIndicator = ""
			} else if hasKids {
				expandIndicator = ""
			} else {
				expandIndicator = " "
			}
		} else {
			expandIndicator = " "
		}

		// Width available for PrettierName (icon + name), accounting for branch prefix.
		// branchStr is pure ASCII/box-chars so byte length = display width here.
		overhead := 2 + ansi.StringWidth(branchStr) + 2 // cursor+space + branch + indicator+space
		nameWidth := r.ContentWidth() - overhead
		if nameWidth < 4 {
			nameWidth = 4
		}

		isSelected := tree.selected[node.path]
		bgColor := common.FilePanelBGColor
		if clipSet[node.path] {
			if m.copyItems.cut {
				bgColor = clipCutBG
			} else {
				bgColor = clipCopyBG
			}
		}
		rendered := common.PrettierName(
			node.name,
			nameWidth,
			node.isDir,
			isSelected,
			bgColor,
		)

		line := common.FilePanelCursorStyle.Render(cursorChar+" ") +
			common.TreeBranchStyle.Render(branchStr) +
			expandIndicator + " " + rendered

		r.AddLines(line)
	}

	return r.Render()
}

// treeNodeBranchPrefix returns the branch-drawing prefix for node at position idx
// in the flat nodes slice.  For each ancestor depth level it emits "│  " (the
// ancestor has more siblings below) or "   " (the ancestor was the last sibling).
// At the node's own depth it appends "├─ " or "└─ " depending on isLast.
func treeNodeBranchPrefix(nodes []treeNode, idx int) string {
	node := nodes[idx]
	depth := node.depth

	var b strings.Builder
	// Ancestor continuation lines (one per ancestor level above this node)
	for d := 0; d < depth; d++ {
		// Scan backward to find the nearest ancestor at depth d.
		ancestorIsLast := false
		for j := idx - 1; j >= 0; j-- {
			if nodes[j].depth == d {
				ancestorIsLast = nodes[j].isLast
				break
			}
		}
		if ancestorIsLast {
			b.WriteString("  ")
		} else {
			b.WriteString("│ ")
		}
	}
	// Own branch connector
	if node.isLast {
		b.WriteString("╰─")
	} else {
		b.WriteString("├─")
	}
	return b.String()
}
