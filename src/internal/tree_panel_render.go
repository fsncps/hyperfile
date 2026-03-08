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
	rightStr := "d:" + strconv.Itoa(tree.maxDepth)
	if tree.HasSelection() {
		rightStr += fmt.Sprintf(" [%d]", len(tree.selected))
	}
	iconPart := common.FilePanelTopDirectoryIcon
	iconW := ansi.StringWidth(iconPart)
	rightW := ansi.StringWidth(rightStr)
	cw := r.ContentWidth()
	pathAvail := cw - iconW - rightW - 1 // -1 for minimum gap
	if pathAvail < 4 {
		pathAvail = 4
	}
	displayRoot := tree.root
	if tree.mode == treePanelModeDetail {
		displayRoot = tree.detailRoot
	}
	truncatedRoot := common.TruncateTextBeginning(displayRoot, pathAvail, "...")
	pad := max(1, cw-iconW-ansi.StringWidth(truncatedRoot)-rightW)
	headerLine := iconPart +
		common.FilePanelTopPathStyle.Render(truncatedRoot) +
		strings.Repeat(" ", pad) +
		common.FilePanelStyle.Render(rightStr)
	r.AddLines(headerLine)
	r.AddSection()

	filterBarRows := 0
	if tree.contentSearchMode || tree.contentQuery != "" || tree.filter != "" {
		filterBarRows = 1
		inputLabel := "" + icon.Space + tree.filter
		if tree.contentSearchMode || tree.contentQuery != "" {
			inputLabel = icon.Search + ": " + tree.contentQuery
		}
		inputWidth := ansi.StringWidth(inputLabel)
		popupPad := max(0, r.ContentWidth()-inputWidth)
		r.AddLines(strings.Repeat(" ", popupPad) + common.FilePanelStyle.Render(inputLabel))
		r.AddSection()
	}

	// ── Detail view mode ───────────────────────────────────────────────────
	if tree.mode == treePanelModeDetail {
		if len(tree.detailEntries) == 0 {
			r.AddLines(common.FilePanelNoneText)
			return r.Render()
		}
		visibleH := m.mainPanelHeight - 2 - filterBarRows
		end := min(tree.renderIdx+visibleH, len(tree.detailEntries))
		// Fixed column widths: perms(10) + space + size(8) + space + date(12) + space = 32
		const fixedCols = 32
		nameWidth := r.ContentWidth() - fixedCols
		if nameWidth < 4 {
			nameWidth = 4
		}
		for i := tree.renderIdx; i < end; i++ {
			e := tree.detailEntries[i]
			isCursorRow := i == tree.cursor

			// Determine background color - cursor line takes precedence
			bgColor := common.FilePanelBGColor
			if isCursorRow {
				bgColor = common.FilePanelCursorLineBGColor
			}

			name := common.PrettierName(e.name, nameWidth, e.isDir, false, bgColor)
			perms := e.mode.String() // "-rwxr-xr-x" = 10 chars
			size := formatDetailSize(e.size)
			date := e.modTime.Format("Jan 02 15:04")

			// Style the fixed columns with the background color
			fixedStyle := lipgloss.NewStyle().Background(bgColor)
			line := name + " " + fixedStyle.Render(perms+" "+size+" "+date)

			if isCursorRow {
				// Pad to full content width so background fills the line
				lineW := ansi.StringWidth(line)
				if lineW < cw {
					line += fixedStyle.Render(strings.Repeat(" ", cw-lineW))
				}
			}
			r.AddLines(line)
		}
		return r.Render()
	}
	// ── Tree mode (existing render below) ──────────────────────────────────

	nodes := tree.filteredNodes()
	if len(nodes) == 0 {
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
	visibleH := m.mainPanelHeight - 2 - filterBarRows
	end := min(tree.renderIdx+visibleH, len(nodes))

	for i := tree.renderIdx; i < end; i++ {
		node := nodes[i]

		// Branch prefix: ancestor continuation lines + own branch character
		branchStr := treeNodeBranchPrefix(nodes, i)

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
		overhead := ansi.StringWidth(branchStr) + 2 // branch + indicator+space
		nameWidth := r.ContentWidth() - overhead
		if nameWidth < 4 {
			nameWidth = 4
		}

		isSelected := tree.selected[node.path]
		isCursorRow := i == tree.cursor
		
		// Determine background color - cursor line takes precedence
		bgColor := common.FilePanelBGColor
		if isCursorRow {
			bgColor = common.FilePanelCursorLineBGColor
		} else if clipSet[node.path] {
			if m.copyItems.cut {
				bgColor = clipCutBG
			} else {
				bgColor = clipCopyBG
			}
		}
		
		// Use cursor line background for branch when cursor row
		branchStyle := common.TreeBranchStyle
		if isCursorRow {
			branchStyle = common.TreeBranchCursorLineStyle
		}
		
		rendered := common.PrettierNameWithBG(
			node.name,
			nameWidth,
			node.isDir,
			isSelected,
			bgColor,
		)

		// Style for expand indicator (needs background for cursor row)
		expandStyle := lipgloss.NewStyle().Background(bgColor)
		line := branchStyle.Render(branchStr) +
			expandStyle.Render(expandIndicator+" ") + rendered
		if isCursorRow {
			// Pad to full content width so background fills the line
			lineW := ansi.StringWidth(line)
			cw := r.ContentWidth()
			if lineW < cw {
				line += expandStyle.Render(strings.Repeat(" ", cw-lineW))
			}
		}

		r.AddLines(line)
	}

	return r.Render()
}

// formatDetailSize returns a human-readable size string right-aligned to 8 chars.
func formatDetailSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	var s string
	switch {
	case bytes >= gb:
		s = fmt.Sprintf("%.1fG", float64(bytes)/gb)
	case bytes >= mb:
		s = fmt.Sprintf("%.1fM", float64(bytes)/mb)
	case bytes >= kb:
		s = fmt.Sprintf("%.1fK", float64(bytes)/kb)
	default:
		s = fmt.Sprintf("%dB", bytes)
	}
	return fmt.Sprintf("%8s", s)
}

// treeNodeBranchPrefix returns the branch-drawing prefix for node at position idx
// in the flat nodes slice.  For each ancestor depth level it emits "│  " (the
// ancestor has more siblings below) or "   " (the ancestor was the last sibling).
// At the node's own depth it appends "├─ " or "└─ " depending on isLast.
func treeNodeBranchPrefix(nodes []treeNode, idx int) string {
	node := nodes[idx]
	depth := node.depth
	if depth < 0 {
		return ""
	}

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
			b.WriteString(" │ ")
		}
	}
	// Own branch connector
	if node.isLast {
		b.WriteString(" ╰─")
	} else {
		b.WriteString(" ├─")
	}
	return b.String()
}
