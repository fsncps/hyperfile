package internal

import (
	"log/slog"
	"os"
	"slices"

	"github.com/fsncps/hyperfile/src/internal/common"

	tea "github.com/charmbracelet/bubbletea"
)

// handleTreePanelKey handles all keyboard input when the tree panel has focus.
func (m *model) handleTreePanelKey(msg string) tea.Cmd {
	visibleH := panelElementHeight(m.mainPanelHeight)

	switch {
	case slices.Contains(common.Hotkeys.ListUp, msg):
		m.treePanel.ListUp(visibleH)

	case slices.Contains(common.Hotkeys.ListDown, msg):
		m.treePanel.ListDown(visibleH)

	case slices.Contains(common.Hotkeys.Confirm, msg):
		// right/enter: expand dir, or open file
		return m.treeEnterNode()

	case slices.Contains(common.Hotkeys.ParentDirectory, msg):
		// left/backspace: collapse node or move up
		m.treePanel.CollapseNode()

	case msg == "ctrl+=", msg == "ctrl++":
		m.treePanel.ChangeDepth(+1)
		m.syncTreeHiddenState()

	case msg == "ctrl+-":
		m.treePanel.ChangeDepth(-1)
		m.syncTreeHiddenState()

	case msg == "tab":
		// Switch focus back to folder panel
		m.setFolderPanelActive()

	case msg == "alt+1":
		m.toggleFolderPanel()

	case msg == "alt+2":
		m.toggleTreePanel()

	case slices.Contains(common.Hotkeys.ToggleFilePreviewPanel, msg):
		m.toggleFilePreviewPanel()

	case slices.Contains(common.Hotkeys.ToggleDotFile, msg):
		m.toggleDotFileController()
		m.syncTreeHiddenState()
		m.treePanel.rebuild()

	case slices.Contains(common.Hotkeys.FocusOnSidebar, msg):
		m.focusOnSideBar()

	case slices.Contains(common.Hotkeys.FocusOnProcessBar, msg):
		m.focusOnProcessBar()

	case slices.Contains(common.Hotkeys.FocusOnMetaData, msg):
		m.focusOnMetadata()

	case slices.Contains(common.Hotkeys.OpenHelpMenu, msg):
		m.openHelpMenu()

	case slices.Contains(common.Hotkeys.ToggleFooter, msg):
		m.toggleFooterController()
	}

	return nil
}

// treeEnterNode expands a dir node at cursor, or opens a file node.
func (m *model) treeEnterNode() tea.Cmd {
	node := m.treePanel.GetSelectedNode()
	if node == nil {
		return nil
	}
	if node.isDir {
		m.treePanel.ExpandNode()
		return nil
	}
	// Open file using the existing mechanism
	_, err := os.Stat(node.path)
	if err != nil {
		slog.Error("tree: cannot stat file", "path", node.path, "err", err)
		return nil
	}
	// Temporarily point the folder panel cursor to this file's path
	// so the existing openFile logic can use it. Not ideal but avoids duplication.
	// Actually, just run xdg-open / open directly for the file.
	m.openTreeNodeFile(node.path)
	return nil
}

// openTreeNodeFile opens a file from the tree panel using the OS default handler.
func (m *model) openTreeNodeFile(path string) {
	slog.Debug("tree: opening file", "path", path)
	// Reuse existing logic by temporarily setting the focused panel's element
	// We create a synthetic element and call the open command.
	// This keeps file-open logic centralized in executeOpenCommand().
	panel := m.getFocusedFilePanel()
	originalElements := panel.element
	originalCursor := panel.cursor
	panel.element = []element{{name: "", location: path, directory: false}}
	panel.cursor = 0
	m.executeOpenCommand()
	panel.element = originalElements
	panel.cursor = originalCursor
}

// setFolderPanelActive switches keyboard focus to the folder panel.
func (m *model) setFolderPanelActive() {
	m.activeFileArea = folderPanelActive
	m.fileModel.filePanels[0].focusType = returnFocusType(m.focusPanel)
	m.treePanel.focusType = secondFocus
}

// setTreePanelActive switches keyboard focus to the tree panel.
func (m *model) setTreePanelActive() {
	m.activeFileArea = treePanelActive
	m.fileModel.filePanels[0].focusType = secondFocus
	m.treePanel.focusType = focus
}

// syncTreeHiddenState pushes the current toggleDotFile value into the tree panel.
func (m *model) syncTreeHiddenState() {
	m.treePanel.showHidden = m.toggleDotFile
}
