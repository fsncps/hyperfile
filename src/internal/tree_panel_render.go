package internal

import (
	"strconv"
	"strings"

	"github.com/fsncps/hyperfile/src/config/icon"
	"github.com/fsncps/hyperfile/src/internal/common"
	"github.com/fsncps/hyperfile/src/internal/ui"
)

// treePanelRender renders the middle tree panel and returns the resulting string.
// Returns an empty string when the panel is closed.
func (m *model) treePanelRender() string {
	if !m.treePanel.open {
		return ""
	}
	focused := m.focusPanel == nonePanelFocus && m.activeFileArea == treePanelActive
	r := ui.FilePanelRenderer(m.mainPanelHeight+2, m.treePanel.width+2, focused)

	// Top bar: path of tree root
	truncatedRoot := common.TruncateTextBeginning(m.treePanel.root, m.treePanel.width-2, "...")
	r.AddLines(common.FilePanelTopDirectoryIcon + common.FilePanelTopPathStyle.Render(truncatedRoot))
	r.AddSection()

	// Depth indicator line (acts as the "search bar" row for spacing)
	r.AddLines(common.FilePanelStyle.Render(" depth:" + strconv.Itoa(m.treePanel.maxDepth)))

	if len(m.treePanel.nodes) == 0 {
		r.AddLines(common.FilePanelNoneText)
		return r.Render()
	}

	visibleH := panelElementHeight(m.mainPanelHeight)
	end := min(m.treePanel.renderIdx+visibleH, len(m.treePanel.nodes))

	for i := m.treePanel.renderIdx; i < end; i++ {
		node := m.treePanel.nodes[i]

		// Cursor indicator
		cursorChar := " "
		if i == m.treePanel.cursor {
			cursorChar = icon.Cursor
		}

		// Indent based on depth
		indent := strings.Repeat("  ", node.depth)

		// Expand/collapse indicator
		var expandIndicator string
		if node.isDir {
			hasKids := m.treePanel.HasChildren(node.path)
			if hasKids && m.treePanel.IsExpanded(node.path) && node.depth < m.treePanel.maxDepth {
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
		nameWidth := m.treePanel.width - overhead
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

