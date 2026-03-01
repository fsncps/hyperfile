package internal

import (
	"strconv"
	"strings"

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

	// Top bar: path of tree root
	truncatedRoot := common.TruncateTextBeginning(tree.root, tree.width-2, "...")
	r.AddLines(common.FilePanelTopDirectoryIcon + common.FilePanelTopPathStyle.Render(truncatedRoot))
	r.AddSection()

	// Depth indicator line (acts as the "search bar" row for spacing)
	r.AddLines(common.FilePanelStyle.Render(" depth:" + strconv.Itoa(tree.maxDepth)))

	if len(tree.nodes) == 0 {
		r.AddLines(common.FilePanelNoneText)
		return r.Render()
	}

	visibleH := panelElementHeight(m.mainPanelHeight)
	end := min(tree.renderIdx+visibleH, len(tree.nodes))

	for i := tree.renderIdx; i < end; i++ {
		node := tree.nodes[i]

		// Cursor indicator
		cursorChar := " "
		if i == tree.cursor {
			cursorChar = icon.Cursor
		}

		// Indent based on depth
		indent := strings.Repeat("  ", node.depth)

		// Expand/collapse indicator
		var expandIndicator string
		if node.isDir {
			hasKids := tree.HasChildren(node.path)
			if hasKids && tree.IsExpanded(node.path) && node.depth < tree.maxDepth {
				expandIndicator = "▾"
			} else if hasKids {
				expandIndicator = "▸"
			} else {
				expandIndicator = " "
			}
		} else {
			expandIndicator = " "
		}

		// Width available for PrettierName (which handles icon+name)
		overhead := 2 + len(indent) + 2 // cursor+space + indent + indicator+space
		nameWidth := tree.width - overhead
		if nameWidth < 4 {
			nameWidth = 4
		}

		rendered := common.PrettierName(
			node.name,
			nameWidth,
			node.isDir,
			false,
			common.FilePanelBGColor,
		)

		line := common.FilePanelCursorStyle.Render(cursorChar+" ") +
			indent + expandIndicator + " " + rendered

		r.AddLines(line)
	}

	return r.Render()
}
