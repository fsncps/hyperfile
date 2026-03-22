package internal

import (
	"log/slog"
	"slices"
	"time"
	"unicode"

	"github.com/yorukot/superfile/src/internal/backend/git"
	"github.com/yorukot/superfile/src/internal/common"
	"github.com/yorukot/superfile/src/internal/ui/logpanel"

	tea "github.com/charmbracelet/bubbletea"
	variable "github.com/yorukot/superfile/src/config"
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
		m.treeEnterNode(idx)
		if idx == 0 && m.secondaryPanel.mode == SecondaryDetail {
			if node := tree.GetSelectedNode(); node != nil && node.isDir {
				if m.secondaryPanel.detail != nil {
					m.secondaryPanel.detail.root = node.path
					m.secondaryPanel.detail.entries = buildDetailEntries(node.path, m.secondaryPanel.detail.showHidden)
					m.secondaryPanel.detail.cursor = 0
					m.secondaryPanel.detail.renderIdx = 0
				}
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
				m.contentSearchInActiveTree(tree.contentQuery)
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
		if idx == 0 && m.secondaryPanel.IsOpen() {
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

	case slices.Contains(common.Hotkeys.SecPanelModeTree, msg):
		if idx == 1 {
			m.setSecondaryMode(SecondaryTree)
		}

	case slices.Contains(common.Hotkeys.SecPanelModeDetail, msg):
		if idx == 1 {
			m.setSecondaryMode(SecondaryDetail)
		}

	case slices.Contains(common.Hotkeys.SecPanelModeGit, msg):
		if idx == 1 {
			m.setSecondaryMode(SecondaryGit)
		}

	case slices.Contains(common.Hotkeys.SecPanelModeDiskUsage, msg):
		if idx == 1 {
			m.setSecondaryMode(SecondaryDiskUsage)
		}

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

	case slices.Contains(common.Hotkeys.ClearClipboard, msg):
		slog.Info("ClearClipboard hotkey triggered", "msg", msg, "hotkeys", common.Hotkeys.ClearClipboard)
		m.copyItems.reset(false)

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

	// Log level hotkeys: alt+5=debug, alt+6=info, alt+7=warn, alt+8=error
	case msg == "alt+5":
		logpanel.SetLogLevel(slog.LevelDebug)
		slog.Debug("Log level changed", "level", "debug")
	case msg == "alt+6":
		logpanel.SetLogLevel(slog.LevelInfo)
		slog.Info("Log level changed", "level", "info")
	case msg == "alt+7":
		logpanel.SetLogLevel(slog.LevelWarn)
		slog.Warn("Log level changed", "level", "warn")
	case msg == "alt+8":
		logpanel.SetLogLevel(slog.LevelError)
		slog.Error("Log level changed", "level", "error")

	default:
		runes := []rune(msg)
		if len(runes) == 1 && !unicode.IsControl(runes[0]) {
			if tree.contentSearchMode {
				tree.appendContentQueryChar(msg)
				m.contentSearchInActiveTree(tree.contentQuery)
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
//
//nolint:unused
func (m *model) treeEnterNode(idx int) {
	tree := m.treePanelByIndex(idx)
	node := tree.GetSelectedNode()
	if node == nil || !node.isDir {
		return
	}
	tree.ExpandNode()
}

// treePanelByIndex returns a pointer to the tree panel at the given index.
// 0 returns the primary panel, 1 returns the secondary panel.
func (m *model) treePanelByIndex(idx int) *treePanelModel {
	if idx == 0 {
		return &m.primaryPanel
	}
	return m.secondaryPanel.tree
}

func (m *model) setSecondaryMode(mode SecondaryMode) {
	prevMode := m.secondaryPanel.mode
	m.secondaryPanel.mode = mode

	if mode.SyncsWithPrimary() && prevMode != mode {
		m.syncSecondaryToPrimary()
	}
}

func (m *model) syncSecondaryToPrimary() {
	selected := m.primaryPanel.GetSelectedNode()
	if selected == nil {
		return
	}

	switch m.secondaryPanel.mode {
	case SecondaryDetail:
		if m.secondaryPanel.detail != nil {
			m.secondaryPanel.detail.root = selected.path
			m.secondaryPanel.detail.entries = buildDetailEntries(selected.path, m.secondaryPanel.detail.showHidden)
		}
	case SecondaryGit:
		if m.secondaryPanel.git != nil {
			backend := git.NewBackend()
			repoRoot := backend.FindRepoRoot(m.primaryPanel.root)
			m.secondaryPanel.git.LoadFromRepo(backend, repoRoot)
		}
	case SecondaryDiskUsage:
		if m.secondaryPanel.diskUsage != nil {
			m.secondaryPanel.diskUsage.LoadFromDir(selected.path)
		}
	}
}

func (m *model) setPrimaryPanelActive() {
	m.activeFileArea = primaryPanelActive
	m.primaryPanel.focusType = focus
	if m.secondaryPanel.tree != nil {
		m.secondaryPanel.focusType = secondFocus
	}
}

func (m *model) setSecondaryPanelActive() {
	m.activeFileArea = secondaryPanelActive
	m.primaryPanel.focusType = secondFocus
	if m.secondaryPanel.tree != nil {
		m.secondaryPanel.focusType = focus
	}
}

func (m *model) rebuildAllTrees() {
	m.primaryPanel.rebuild()
	if m.secondaryPanel.tree != nil {
		m.secondaryPanel.tree.rebuild()
	}
	if m.primaryPanel.mode == treePanelModeDetail {
		m.primaryPanel.detailEntries = buildDetailEntries(m.primaryPanel.detailRoot, m.primaryPanel.showHidden)
	}
	if m.secondaryPanel.mode == SecondaryDetail && m.secondaryPanel.detail != nil {
		m.secondaryPanel.detail.entries = buildDetailEntries(m.secondaryPanel.detail.root, m.secondaryPanel.detail.showHidden)
	}
}

func (m *model) syncTreeHiddenState() {
	m.primaryPanel.showHidden = m.toggleDotFile
	if m.secondaryPanel.tree != nil {
		m.secondaryPanel.tree.showHidden = m.toggleDotFile
	}
	if m.secondaryPanel.detail != nil {
		m.secondaryPanel.detail.showHidden = m.toggleDotFile
	}
	if m.primaryPanel.mode == treePanelModeDetail {
		m.primaryPanel.detailEntries = buildDetailEntries(m.primaryPanel.detailRoot, m.primaryPanel.showHidden)
	}
	if m.secondaryPanel.mode == SecondaryDetail && m.secondaryPanel.detail != nil {
		m.secondaryPanel.detail.entries = buildDetailEntries(m.secondaryPanel.detail.root, m.secondaryPanel.detail.showHidden)
	}
}

// startPreviewDebounce records the cursor-moved timestamp and returns a command
// that fires previewDebounceDuration later, triggering a View() re-evaluation.
//
//nolint:unused
func (m *model) startPreviewDebounce() tea.Cmd {
	m.lastCursorMovedAt = time.Now()
	return func() tea.Msg {
		time.Sleep(previewDebounceDuration)
		return previewTickMsg{}
	}
}

const previewDebounceDuration = 100 * time.Millisecond

type previewTickMsg struct{}

func (m *model) focusNextPanel() {
	if m.activeFileArea == primaryPanelActive && m.secondaryPanel.IsOpen() {
		m.setSecondaryPanelActive()
	} else {
		m.setPrimaryPanelActive()
	}
}

func (m *model) focusPreviousPanel() {
	if m.activeFileArea == secondaryPanelActive {
		m.setPrimaryPanelActive()
	} else if m.secondaryPanel.IsOpen() {
		m.setSecondaryPanelActive()
	}
}

func (m *model) setViewMode(mode viewModeType) {
	m.viewMode = mode
	switch mode {
	case viewModeBothWithPreview:
		m.secondaryPanel.SetOpen(true)
		m.fileModel.filePreview.open = true
	case viewModeBothNoPreview:
		m.secondaryPanel.SetOpen(true)
		m.fileModel.filePreview.open = false
	case viewModeMainWithPreview:
		m.secondaryPanel.SetOpen(false)
		m.fileModel.filePreview.open = true
	case viewModeMainOnly:
		m.secondaryPanel.SetOpen(false)
		m.fileModel.filePreview.open = false
	}
}

func (m *model) contentSearchInActiveTree(query string) {
	tree := m.treePanelByIndex(0)
	if tree == nil || query == "" {
		return
	}
	tree.setContentFilter(nil, query)
}

func (m *model) toggleDetailView(idx int) {
	if idx == 0 {
		tree := &m.primaryPanel
		if tree.mode == treePanelModeDetail {
			tree.mode = treePanelModeTree
		} else {
			tree.mode = treePanelModeDetail
			if node := tree.GetSelectedNode(); node != nil && node.isDir {
				tree.detailRoot = node.path
				tree.detailEntries = buildDetailEntries(node.path, tree.showHidden)
			}
		}
	} else {
		sp := &m.secondaryPanel
		if sp.mode == SecondaryDetail {
			sp.mode = SecondaryTree
		} else {
			sp.mode = SecondaryDetail
			if sp.detail == nil {
				sp.detail = &detailViewModel{}
			}
			node := m.primaryPanel.GetSelectedNode()
			if node != nil && node.isDir {
				sp.detail.root = node.path
				sp.detail.entries = buildDetailEntries(node.path, sp.detail.showHidden)
			}
		}
	}
}

func (m *model) treePasteCmd(tree *treePanelModel) tea.Cmd {
	return nil
}

func (m *model) copyTreeSelection(tree *treePanelModel, cut bool) {
}

func (m *model) copySingleTreeItem(tree *treePanelModel, cut bool) {
}
