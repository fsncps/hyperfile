package internal

import (
	"log/slog"
	"slices"
	"time"

	"github.com/fsncps/hyperfile/src/internal/common"

	tea "github.com/charmbracelet/bubbletea"
	variable "github.com/fsncps/hyperfile/src/config"
)

// handleTreePanelKey handles all keyboard input when a tree panel has focus.
// idx is 0 for the left tree and 1 for the right tree.
//
//nolint:cyclop,funlen // large dispatch switch
func (m *model) handleTreePanelKey(msg string, idx int) tea.Cmd {
	tree := &m.treePanels[idx]
	visibleH := m.mainPanelHeight - 2
	if tree.rgSearchBar.Focused() || tree.rgSearchBar.Value() != "" {
		visibleH -= 2
	}

	// If the rg search bar is focused, route all input through it.
	if tree.rgSearchBar.Focused() {
		return m.handleRgSearchBarKey(msg, idx)
	}

	switch {

	// ---- Tree navigation ----
	case slices.Contains(common.Hotkeys.ListUp, msg):
		tree.ListUp(visibleH)
		return m.startPreviewDebounce()

	case slices.Contains(common.Hotkeys.ListDown, msg):
		tree.ListDown(visibleH)
		return m.startPreviewDebounce()

	case slices.Contains(common.Hotkeys.Confirm, msg):
		// Expand directory node in the tree. Also sync the file panel location
		// (used by file-op infrastructure and navigation tests).
		// For regular files: only act when in chooser mode (never xdg-open from right arrow).
		m.treeEnterNode(idx) //nolint:errcheck // returns nil cmd
		// Sync opposite panel if it is in detail mode.
		otherIdx := 1 - idx
		if m.treePanels[otherIdx].mode == treePanelModeDetail {
			if node := m.treePanels[idx].GetSelectedNode(); node != nil && node.isDir {
				other := &m.treePanels[otherIdx]
				other.detailRoot = node.path
				other.detailEntries = buildDetailEntries(node.path, other.showHidden)
				other.cursor = 0
				other.renderIdx = 0
			}
		}
		panel := m.getFocusedFilePanel()
		if len(panel.element) > 0 {
			item := panel.getSelectedItem()
			if item.directory {
				if err := panel.updateCurrentFilePanelDir(item.location); err != nil {
					slog.Error("Error while changing to directory", "error", err, "target", item.location)
				}
			} else if variable.ChooserFile != "" {
				// Chooser-file mode: write the path and quit.
				if err := m.chooserFileWriteAndQuit(panel.element[panel.cursor].location); err != nil {
					slog.Error("Error while writing to chooser file", "error", err)
				}
			}
		}
		return nil

	case msg == "backspace":
		m.treePanels[idx].RootUp()

	case slices.Contains(common.Hotkeys.ParentDirectory, msg):
		// Collapse tree node; also drive file panel navigation.
		tree.CollapseNode()
		m.parentDirectory()

	case msg == "alt+=", msg == "alt++":
		tree.ChangeDepth(+1)
		m.syncTreeHiddenState()

	case msg == "alt+-":
		tree.ChangeDepth(-1)
		m.syncTreeHiddenState()

	// ---- Focus cycling ----
	case msg == "tab":
		if idx == 0 && m.treePanels[1].open {
			m.setTree2PanelActive()
		} else {
			m.setTree1PanelActive()
		}

	case msg == "ctrl+right":
		m.focusNextPanel()

	case msg == "ctrl+left":
		m.focusPreviousPanel()

	case msg == "ctrl+up":
		m.focusOnProcessBar()

	// ---- Panel visibility toggles ----
	case msg == "alt+1":
		m.toggleTree1Panel()

	case msg == "alt+2":
		m.toggleTree2Panel()

	case slices.Contains(common.Hotkeys.ToggleFilePreviewPanel, msg):
		m.toggleFilePreviewPanel()

	case slices.Contains(common.Hotkeys.ToggleDetailView, msg):
		m.toggleDetailView(1) // detail view always in right panel

	case slices.Contains(common.Hotkeys.ContentSearch, msg):
		tree.rgSearchBar.Focus()
		tree.rgSearchBar.Width = tree.width - 8
		return nil

	case slices.Contains(common.Hotkeys.ToggleDotFile, msg):
		m.toggleDotFileController()
		m.syncTreeHiddenState()
		m.treePanels[idx].rebuild()

	case slices.Contains(common.Hotkeys.ToggleFooter, msg):
		m.toggleFooterController()

	// ---- Footer / sidebar focus ----
	case slices.Contains(common.Hotkeys.FocusOnSidebar, msg):
		m.focusOnSideBar()

	case slices.Contains(common.Hotkeys.FocusOnProcessBar, msg):
		m.focusOnProcessBar()

	case slices.Contains(common.Hotkeys.FocusOnMetaData, msg):
		m.focusOnMetadata()

	// ---- Modals / menus ----
	case slices.Contains(common.Hotkeys.OpenHelpMenu, msg):
		m.openHelpMenu()

	case slices.Contains(common.Hotkeys.OpenCommandLine, msg):
		m.promptModal.Open(true)

	case slices.Contains(common.Hotkeys.OpenSPFPrompt, msg):
		m.promptModal.Open(false)

	case slices.Contains(common.Hotkeys.OpenSortOptionsMenu, msg):
		m.openSortOptionsMenu()

	case slices.Contains(common.Hotkeys.ToggleReverseSort, msg):
		m.toggleReverseSort()

	// ---- Editor ----
	case slices.Contains(common.Hotkeys.OpenFileWithEditor, msg):
		return m.openFileWithEditor()

	case slices.Contains(common.Hotkeys.OpenCurrentDirectoryWithEditor, msg):
		return m.openDirectoryWithEditor()

	// ---- Directory management ----
	case slices.Contains(common.Hotkeys.PinnedDirectory, msg):
		m.pinnedDirectory()

	// ---- File operations (act on focused file panel) ----
	case slices.Contains(common.Hotkeys.PasteItems, msg):
		return m.treePasteCmd(tree)

	case slices.Contains(common.Hotkeys.DragItems, msg):
		return m.dragItems(tree)

	case slices.Contains(common.Hotkeys.FilePanelItemCreate, msg):
		m.panelCreateNewFile()

	case slices.Contains(common.Hotkeys.ExtractFile, msg):
		return m.getExtractFileCmd()

	case slices.Contains(common.Hotkeys.CompressFile, msg):
		return m.getCompressSelectedFilesCmd()

	case slices.Contains(common.Hotkeys.CopyPath, msg):
		m.copyPath()

	case slices.Contains(common.Hotkeys.CopyPWD, msg):
		m.copyPWD()

	case slices.Contains(common.Hotkeys.DeleteItems, msg):
		return m.getDeleteTriggerCmd()

	case slices.Contains(common.Hotkeys.FilePanelItemRename, msg):
		m.panelItemRename()

	case slices.Contains(common.Hotkeys.CopyItems, msg):
		if tree.HasSelection() {
			m.copyTreeSelection(tree, false)
		} else {
			m.copySingleTreeItem(tree, false)
		}

	case slices.Contains(common.Hotkeys.CutItems, msg):
		if tree.HasSelection() {
			m.copyTreeSelection(tree, true)
		} else {
			m.copySingleTreeItem(tree, true)
		}

	case slices.Contains(common.Hotkeys.FilePanelSelectAllItem, msg):
		m.selectAllItem()

	case slices.Contains(common.Hotkeys.FilePanelSelectModeItemsSelectUp, msg):
		tree.ShiftListUp(visibleH)
		return m.startPreviewDebounce()

	case slices.Contains(common.Hotkeys.FilePanelSelectModeItemsSelectDown, msg):
		tree.ShiftListDown(visibleH)
		return m.startPreviewDebounce()

	case slices.Contains(common.Hotkeys.NextFilePanel, msg):
		m.nextFilePanel()

	case slices.Contains(common.Hotkeys.PreviousFilePanel, msg):
		m.previousFilePanel()

	case slices.Contains(common.Hotkeys.CloseFilePanel, msg):
		m.closeFilePanel()

	case slices.Contains(common.Hotkeys.CreateNewFilePanel, msg):
		err := m.createNewFilePanel(variable.HomeDir)
		if err != nil {
			slog.Error("error while creating new panel", "error", err)
		}
	}

	return nil
}

// treeEnterNode expands a directory node at the cursor of the given tree.
func (m *model) treeEnterNode(idx int) tea.Cmd {
	node := m.treePanels[idx].GetSelectedNode()
	if node == nil || !node.isDir {
		return nil
	}
	m.treePanels[idx].ExpandNode()
	return nil
}

// setTree1PanelActive switches keyboard focus to the left tree (index 0).
func (m *model) setTree1PanelActive() {
	m.activeFileArea = tree1PanelActive
	m.treePanels[0].focusType = focus
	m.treePanels[1].focusType = secondFocus
}

// setTree2PanelActive switches keyboard focus to the right tree (index 1).
func (m *model) setTree2PanelActive() {
	m.activeFileArea = tree2PanelActive
	m.treePanels[0].focusType = secondFocus
	m.treePanels[1].focusType = focus
}

// syncTreeHiddenState pushes the current toggleDotFile value into both tree panels.
// rebuildAllTrees refreshes both tree panels from disk, e.g. after file operations.
func (m *model) rebuildAllTrees() {
	for i := range m.treePanels {
		m.treePanels[i].rebuild()
	}
	// Refresh detail entries for any panel currently in detail mode.
	for i := range m.treePanels {
		if m.treePanels[i].mode == treePanelModeDetail {
			m.treePanels[i].detailEntries = buildDetailEntries(m.treePanels[i].detailRoot, m.treePanels[i].showHidden)
		}
	}
}

func (m *model) syncTreeHiddenState() {
	m.treePanels[0].showHidden = m.toggleDotFile
	m.treePanels[1].showHidden = m.toggleDotFile
	// Refresh detail entries for any panel currently in detail mode.
	for i := range m.treePanels {
		if m.treePanels[i].mode == treePanelModeDetail {
			m.treePanels[i].detailEntries = buildDetailEntries(m.treePanels[i].detailRoot, m.treePanels[i].showHidden)
		}
	}
}

// startPreviewDebounce records the cursor-moved timestamp and returns a command
// that fires previewDebounceDuration later, triggering a View() re-evaluation.
func (m *model) startPreviewDebounce() tea.Cmd {
	m.lastCursorMovedAt = time.Now()
	return func() tea.Msg {
		time.Sleep(previewDebounceDuration)
		return previewTickMsg{}
	}
}

// handleRgSearchBarKey handles all input while the tree panel's rg search bar
// is focused. Navigation keys pass through to the tree; cancel keys clear the
// filter; all other keys are forwarded to the text input (via updateFilePanelsState)
// and a debounce tick is posted.
func (m *model) handleRgSearchBarKey(msg string, idx int) tea.Cmd {
	tree := &m.treePanels[idx]
	switch {
	case slices.Contains(common.Hotkeys.ContentSearch, msg), msg == "esc":
		tree.rgSearchBar.Blur()
		tree.rgSearchBar.SetValue("")
		tree.rgMatches = nil
		tree.cursor = 0
		tree.renderIdx = 0
		tree.rebuild()
		return nil
	case slices.Contains(common.Hotkeys.ListUp, msg):
		tree.ListUp(m.mainPanelHeight - 2)
		return m.startPreviewDebounce()
	case slices.Contains(common.Hotkeys.ListDown, msg):
		tree.ListDown(m.mainPanelHeight - 2)
		return m.startPreviewDebounce()
	}
	// All other keys (printable chars, backspace) are handled by
	// updateFilePanelsState; we just schedule a debounce tick.
	return m.startRgDebounce(idx)
}
