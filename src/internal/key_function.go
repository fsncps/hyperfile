package internal

import (
	"log/slog"
	"slices"
	"unicode"

	"github.com/fsncps/hyperfile/src/internal/common"
	"github.com/fsncps/hyperfile/src/internal/ui/notify"

	tea "github.com/charmbracelet/bubbletea"
	variable "github.com/fsncps/hyperfile/src/config"
)

// mainKey handles most of key commands in the regular state of the application. For
// keys that performs actions in multiple panels, like going up or down,
// check the state of model m and handle properly.
// TODO: This function has grown too big. It needs to be fixed, via major
// updates and fixes in key handling code
func (m *model) mainKey(msg string) tea.Cmd { //nolint: gocyclo,cyclop,funlen // See above
	// Route all file-area input to the active tree panel handler
	if m.focusPanel == nonePanelFocus {
		return m.handleTreePanelKey(msg, int(m.activeFileArea))
	}

	switch {
	case msg == "ctrl+right":
		m.focusNextPanel()
		return nil
	case msg == "ctrl+left":
		m.focusPreviousPanel()
		return nil
	case msg == "ctrl+up":
		m.focusOnProcessBar()
		return nil
	case msg == "ctrl+down":
		m.focusPanel = nonePanelFocus
		return nil

	// If move up Key is pressed, check the current state and executes
	case slices.Contains(common.Hotkeys.ListUp, msg):
		switch m.focusPanel {
		case sidebarFocus:
			m.sidebarModel.ListUp(m.mainPanelHeight)
		case processBarFocus:
			m.processBarModel.ListUp(m.footerHeight)
		case metadataFocus:
			m.fileMetaData.ListUp()
		case nonePanelFocus:
			m.fileModel.filePanels[m.filePanelFocusIndex].listUp(m.mainPanelHeight)
		}

		// If move down Key is pressed, check the current state and executes
	case slices.Contains(common.Hotkeys.ListDown, msg):
		switch m.focusPanel {
		case sidebarFocus:
			m.sidebarModel.ListDown(m.mainPanelHeight)
		case processBarFocus:
			m.processBarModel.ListDown(m.footerHeight)
		case metadataFocus:
			m.fileMetaData.ListDown()
		case nonePanelFocus:
			m.fileModel.filePanels[m.filePanelFocusIndex].listDown(m.mainPanelHeight)
		}

	case slices.Contains(common.Hotkeys.PageUp, msg):
		m.fileModel.filePanels[m.filePanelFocusIndex].pgUp(m.mainPanelHeight)

	case slices.Contains(common.Hotkeys.PageDown, msg):
		m.fileModel.filePanels[m.filePanelFocusIndex].pgDown(m.mainPanelHeight)

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
	case slices.Contains(common.Hotkeys.ToggleFilePreviewPanel, msg):
		m.toggleFilePreviewPanel()

	case slices.Contains(common.Hotkeys.FocusOnSidebar, msg):
		m.focusOnSideBar()

	case slices.Contains(common.Hotkeys.FocusOnProcessBar, msg):
		m.focusOnProcessBar()

	case slices.Contains(common.Hotkeys.FocusOnMetaData, msg):
		m.focusOnMetadata()

	case slices.Contains(common.Hotkeys.PasteItems, msg):
		return m.getPasteItemCmd()

	case slices.Contains(common.Hotkeys.FilePanelItemCreate, msg):
		m.panelCreateNewFile()
	case slices.Contains(common.Hotkeys.PinnedDirectory, msg):
		m.pinnedDirectory()

	case slices.Contains(common.Hotkeys.ToggleDotFile, msg):
		m.toggleDotFileController()

	case slices.Contains(common.Hotkeys.ToggleFooter, msg):
		m.toggleFooterController()

	case slices.Contains(common.Hotkeys.ExtractFile, msg):
		return m.getExtractFileCmd()

	case slices.Contains(common.Hotkeys.CompressFile, msg):
		return m.getCompressSelectedFilesCmd()

	case slices.Contains(common.Hotkeys.OpenCommandLine, msg):
		m.promptModal.Open(true)
	case slices.Contains(common.Hotkeys.OpenSPFPrompt, msg):
		m.promptModal.Open(false)

	case slices.Contains(common.Hotkeys.OpenHelpMenu, msg):
		m.openHelpMenu()

	case slices.Contains(common.Hotkeys.OpenSortOptionsMenu, msg):
		m.openSortOptionsMenu()

	case slices.Contains(common.Hotkeys.ToggleReverseSort, msg):
		m.toggleReverseSort()

	case slices.Contains(common.Hotkeys.OpenFileWithEditor, msg):
		return m.openFileWithEditor()

	case slices.Contains(common.Hotkeys.OpenCurrentDirectoryWithEditor, msg):
		return m.openDirectoryWithEditor()

	default:
		return m.normalAndBrowserModeKey(msg)
	}

	return nil
}

func (m *model) normalAndBrowserModeKey(msg string) tea.Cmd {
	// if not focus on the filepanel return
	if m.fileModel.filePanels[m.filePanelFocusIndex].focusType != focus {
		if m.focusPanel == sidebarFocus && (slices.Contains(common.Hotkeys.Confirm, msg) ||
			slices.Contains(common.Hotkeys.ConfirmTyping, msg)) {
			m.sidebarSelectDirectory()
		}
		if m.focusPanel == sidebarFocus && slices.Contains(common.Hotkeys.FilePanelItemRename, msg) {
			m.sidebarModel.PinnedItemRename()
		}
		if m.focusPanel == sidebarFocus && slices.Contains(common.Hotkeys.SearchBar, msg) {
			m.sidebarSearchBarFocus()
		}
		return nil
	}
	// Check if in the select mode and focusOn filepanel
	if m.fileModel.filePanels[m.filePanelFocusIndex].panelMode == selectMode {
		switch {
		case slices.Contains(common.Hotkeys.Confirm, msg):
			m.fileModel.filePanels[m.filePanelFocusIndex].singleItemSelect()
		case slices.Contains(common.Hotkeys.FilePanelSelectModeItemsSelectUp, msg):
			m.fileModel.filePanels[m.filePanelFocusIndex].itemSelectUp(m.mainPanelHeight)
		case slices.Contains(common.Hotkeys.FilePanelSelectModeItemsSelectDown, msg):
			m.fileModel.filePanels[m.filePanelFocusIndex].itemSelectDown(m.mainPanelHeight)
		case slices.Contains(common.Hotkeys.DeleteItems, msg):
			return m.getDeleteTriggerCmd()
		case slices.Contains(common.Hotkeys.CopyItems, msg):
			m.copyMultipleItem(false)
		case slices.Contains(common.Hotkeys.CutItems, msg):
			m.copyMultipleItem(true)
		case slices.Contains(common.Hotkeys.FilePanelSelectAllItem, msg):
			m.selectAllItem()
		}
		return nil
	}

	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]

	// esc clears the passive filter when active
	if msg == "esc" {
		if panel.searchBar.Value() != "" {
			panel.searchBar.SetValue("")
		}
		return nil
	}

	switch {
	case slices.Contains(common.Hotkeys.Confirm, msg):
		m.enterPanel()
	case slices.Contains(common.Hotkeys.ParentDirectory, msg):
		// backspace removes filter chars when filter is active; otherwise navigates up
		if msg == "backspace" && panel.searchBar.Value() != "" {
			runes := []rune(panel.searchBar.Value())
			panel.searchBar.SetValue(string(runes[:len(runes)-1]))
		} else {
			m.parentDirectory()
		}
	case slices.Contains(common.Hotkeys.DeleteItems, msg):
		return m.getDeleteTriggerCmd()
	case slices.Contains(common.Hotkeys.CopyItems, msg):
		m.copySingleItem(false)
	case slices.Contains(common.Hotkeys.CutItems, msg):
		m.copySingleItem(true)
	case slices.Contains(common.Hotkeys.FilePanelItemRename, msg):
		m.panelItemRename()
	case slices.Contains(common.Hotkeys.CopyPath, msg):
		m.copyPath()
	case slices.Contains(common.Hotkeys.CopyPWD, msg):
		m.copyPWD()
	default:
		// Passive always-on filter: any printable single character appends to the filter
		runes := []rune(msg)
		if len(runes) == 1 && !unicode.IsControl(runes[0]) {
			panel.searchBar.Width = m.fileModel.width - 4
			panel.searchBar.SetValue(panel.searchBar.Value() + msg)
		}
	}
	return nil
}

// Check the hotkey to cancel operation or create file
func (m *model) typingModalOpenKey(msg string) {
	switch {
	case slices.Contains(common.Hotkeys.CancelTyping, msg):
		m.typingModal.errorMesssage = ""
		m.cancelTypingModal()
	case slices.Contains(common.Hotkeys.ConfirmTyping, msg):
		m.createItem()
	}
}

func (m *model) notifyModelOpenKey(msg string) tea.Cmd {
	isCancel := slices.Contains(common.Hotkeys.CancelTyping, msg) || slices.Contains(common.Hotkeys.Quit, msg)
	isConfirm := slices.Contains(common.Hotkeys.Confirm, msg) || slices.Contains(common.Hotkeys.ConfirmTyping, msg)

	if !isCancel && !isConfirm {
		slog.Warn("Invalid keypress in notifyModel", "msg", msg)
		return nil
	}
	m.notifyModel.Close()
	action := m.notifyModel.GetConfirmAction()
	if isCancel {
		return m.handleNotifyModelCancel(action)
	}
	return m.handleNotifyModelConfirm(action)
}

func (m *model) handleNotifyModelCancel(action notify.ConfirmActionType) tea.Cmd {
	switch action {
	case notify.RenameAction:
		m.cancelRename()
	case notify.QuitAction:
		m.modelQuitState = notQuitting
	case notify.DeleteAction, notify.NoAction:
		// Do nothing
	default:
		slog.Error("Unknown type of action", "action", action)
	}
	return nil
}

func (m *model) handleNotifyModelConfirm(action notify.ConfirmActionType) tea.Cmd {
	switch action {
	case notify.DeleteAction:
		return m.getDeleteCmd()
	case notify.RenameAction:
		m.confirmRename()
	case notify.QuitAction:
		m.modelQuitState = quitConfirmationReceived
	case notify.NoAction:
		// Ignore
	default:
		slog.Error("Unknown type of action", "action", action)
	}
	return nil
}

// Handles key inputs inside sort options menu
func (m *model) sortOptionsKey(msg string) {
	switch {
	case slices.Contains(common.Hotkeys.OpenSortOptionsMenu, msg):
		m.cancelSortOptions()
	case slices.Contains(common.Hotkeys.Quit, msg):
		m.cancelSortOptions()
	case slices.Contains(common.Hotkeys.Confirm, msg):
		m.confirmSortOptions()
	case slices.Contains(common.Hotkeys.ListUp, msg):
		m.sortOptionsListUp()
	case slices.Contains(common.Hotkeys.ListDown, msg):
		m.sortOptionsListDown()
	}
}

func (m *model) renamingKey(msg string) tea.Cmd {
	switch {
	case slices.Contains(common.Hotkeys.CancelTyping, msg):
		m.cancelRename()
	case slices.Contains(common.Hotkeys.ConfirmTyping, msg):
		if m.IsRenamingConflicting() {
			return m.warnModalForRenaming()
		}
		m.confirmRename()
	}

	return nil
}

func (m *model) sidebarRenamingKey(msg string) {
	switch {
	case slices.Contains(common.Hotkeys.CancelTyping, msg):
		m.sidebarModel.CancelSidebarRename()
	case slices.Contains(common.Hotkeys.ConfirmTyping, msg):
		m.sidebarModel.ConfirmSidebarRename()
	}
}

// Check the key input and cancel or confirms the search
func (m *model) focusOnSearchbarKey(msg string) {
	switch {
	case slices.Contains(common.Hotkeys.CancelTyping, msg):
		m.cancelSearch()
	case slices.Contains(common.Hotkeys.ConfirmTyping, msg):
		m.confirmSearch()
	}
}

// Check hotkey input in help menu. Possible actions are moving up, down,
// filtering, and quiting the menu
func (m *model) helpMenuKey(msg string) {
	switch {
	case slices.Contains(common.Hotkeys.ListUp, msg):
		m.helpMenuListUp()
	case slices.Contains(common.Hotkeys.ListDown, msg):
		m.helpMenuListDown()
	case msg == "backspace":
		m.deleteHelpMenuFilterRune()
	case msg == "esc", slices.Contains(common.Hotkeys.Quit, msg):
		if m.helpMenu.filter != "" {
			m.helpMenu.filter = ""
			m.applyHelpMenuFilter()
			return
		}
		m.quitHelpMenu()
	default:
		m.appendHelpMenuFilterRune(msg)
	}
}
