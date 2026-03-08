package internal

import (
	"log/slog"
	"slices"
	"time"
	"unicode"

	"github.com/fsncps/hyperfile/src/internal/common"

	tea "github.com/charmbracelet/bubbletea"
	variable "github.com/fsncps/hyperfile/src/config"
)

// handleTreePanelKey handles all keyboard input when a tree panel has focus.
// idx is 0 for the left (primary) tree and 1 for the right (secondary) tree.
//
//nolint:cyclop,funlen // large dispatch switch
func (m *model) handleTreePanelKey(msg string, idx int) tea.Cmd {
	tree := m.treePanelByIndex(idx)
	visibleH := m.mainPanelHeight - 2

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
		other := m.treePanelByIndex(1 - idx)
		if other.mode == treePanelModeDetail {
			if node := tree.GetSelectedNode(); node != nil && node.isDir {
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
		if tree.contentSearchMode {
			if tree.contentQuery != "" {
				tree.deleteContentQueryChar()
				if err := m.contentSearchInActiveTree(tree.contentQuery); err != nil {
					slog.Error("content search failed", "error", err)
				}
			} else {
				tree.clearContentFilter()
			}
			return m.startPreviewDebounce()
		}
		if tree.filter != "" {
			tree.deleteFilterChar()
			return m.startPreviewDebounce()
		}
		tree.RootUp()
		return m.startPreviewDebounce()

	case msg == "esc":
		tree.clearFilter()
		return m.startPreviewDebounce()

	case slices.Contains(common.Hotkeys.ParentDirectory, msg):
		// Collapse tree node; also drive file panel navigation.
		tree.CollapseNode()
		m.parentDirectory()

	case slices.Contains(common.Hotkeys.TreeDepthIncrease, msg):
		tree.ChangeDepth(+1)
		m.syncTreeHiddenState()

	case slices.Contains(common.Hotkeys.TreeDepthDecrease, msg):
		tree.ChangeDepth(-1)
		m.syncTreeHiddenState()

	// ---- Focus cycling ----
	case msg == "tab":
		if idx == 0 && m.secondaryPanel.open {
			m.setSecondaryPanelActive()
		} else {
			m.setPrimaryPanelActive()
		}

	case msg == "ctrl+right":
		m.focusNextPanel()

	case msg == "ctrl+left":
		m.focusPreviousPanel()

	case msg == "ctrl+up":
		m.focusOnProcessBar()

	// ---- Panel view modes ----
	case slices.Contains(common.Hotkeys.ViewMode1, msg):
		m.setViewMode(viewModeBothWithPreview)

	case slices.Contains(common.Hotkeys.ViewMode2, msg):
		m.setViewMode(viewModeBothNoPreview)

	case slices.Contains(common.Hotkeys.ViewMode3, msg):
		m.setViewMode(viewModeMainWithPreview)

	case slices.Contains(common.Hotkeys.ViewMode4, msg):
		m.setViewMode(viewModeMainOnly)

	case slices.Contains(common.Hotkeys.ToggleFilePreviewPanel, msg):
		m.toggleFilePreviewPanel()

	case slices.Contains(common.Hotkeys.ToggleDetailView, msg):
		m.toggleDetailView(idx)

	case slices.Contains(common.Hotkeys.ToggleDotFile, msg):
		m.toggleDotFileController()
		m.syncTreeHiddenState()
		tree.rebuild()

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

	case slices.Contains(common.Hotkeys.ContentSearch, msg):
		tree.beginContentSearch()

	case msg == "." && !tree.contentSearchMode && tree.filter == "":
		node := tree.GetSelectedNode()
		if node != nil && node.isDir {
			tree.NavigateTo(node.path)
			return m.startPreviewDebounce()
		}

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

	default:
		runes := []rune(msg)
		if len(runes) == 1 && !unicode.IsControl(runes[0]) {
			if tree.contentSearchMode {
				tree.appendContentQueryChar(msg)
				if err := m.contentSearchInActiveTree(tree.contentQuery); err != nil {
					slog.Error("content search failed", "error", err)
				}
			} else {
				// Type-to-filter: any printable single character appends to the filter
				tree.appendFilterChar(msg)
			}
			return m.startPreviewDebounce()
		}
	}

	return nil
}

// treeEnterNode expands a directory node at the cursor of the given tree.
func (m *model) treeEnterNode(idx int) tea.Cmd {
	tree := m.treePanelByIndex(idx)
	node := tree.GetSelectedNode()
	if node == nil || !node.isDir {
		return nil
	}
	tree.ExpandNode()
	return nil
}

// treePanelByIndex returns a pointer to the tree panel at the given index.
// 0 returns the primary panel, 1 returns the secondary panel.
func (m *model) treePanelByIndex(idx int) *treePanelModel {
	if idx == 0 {
		return &m.primaryPanel
	}
	return &m.secondaryPanel
}

// setPrimaryPanelActive switches keyboard focus to the primary (left) tree.
func (m *model) setPrimaryPanelActive() {
	m.activeFileArea = primaryPanelActive
	m.primaryPanel.focusType = focus
	m.secondaryPanel.focusType = secondFocus
}

// setSecondaryPanelActive switches keyboard focus to the secondary (right) tree.
func (m *model) setSecondaryPanelActive() {
	m.activeFileArea = secondaryPanelActive
	m.primaryPanel.focusType = secondFocus
	m.secondaryPanel.focusType = focus
}

// syncTreeHiddenState pushes the current toggleDotFile value into both tree panels.
// rebuildAllTrees refreshes both tree panels from disk, e.g. after file operations.
func (m *model) rebuildAllTrees() {
	m.primaryPanel.rebuild()
	m.secondaryPanel.rebuild()
	// Refresh detail entries for any panel currently in detail mode.
	if m.primaryPanel.mode == treePanelModeDetail {
		m.primaryPanel.detailEntries = buildDetailEntries(m.primaryPanel.detailRoot, m.primaryPanel.showHidden)
	}
	if m.secondaryPanel.mode == treePanelModeDetail {
		m.secondaryPanel.detailEntries = buildDetailEntries(m.secondaryPanel.detailRoot, m.secondaryPanel.showHidden)
	}
}

func (m *model) syncTreeHiddenState() {
	m.primaryPanel.showHidden = m.toggleDotFile
	m.secondaryPanel.showHidden = m.toggleDotFile
	// Refresh detail entries for any panel currently in detail mode.
	if m.primaryPanel.mode == treePanelModeDetail {
		m.primaryPanel.detailEntries = buildDetailEntries(m.primaryPanel.detailRoot, m.primaryPanel.showHidden)
	}
	if m.secondaryPanel.mode == treePanelModeDetail {
		m.secondaryPanel.detailEntries = buildDetailEntries(m.secondaryPanel.detailRoot, m.secondaryPanel.showHidden)
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
