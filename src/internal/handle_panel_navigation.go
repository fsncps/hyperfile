package internal

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/fsncps/hyperfile/src/internal/ui/sidebar"

	"github.com/fsncps/hyperfile/src/internal/common"

	variable "github.com/fsncps/hyperfile/src/config"
)

// Pinned directory
func (m *model) pinnedDirectory() {
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]
	err := sidebar.TogglePinnedDirectory(panel.location)
	if err != nil {
		slog.Error("Error while toggling pinned directory", "error", err)
	}
}

// Create new file panel
func (m *model) createNewFilePanel(location string) error {
	// In case we have model width and height zero, maxFilePanel would be 0
	// But we would have len() here as 1. Hence there would be discrepency here.
	// Although this is not possible in actual usage, and can be only reproduced in tests.
	if len(m.fileModel.filePanels) == m.fileModel.maxFilePanel {
		// TODO : Define as a predefined error in errors.go
		return errors.New("maximum panel count reached")
	}

	if location == "" {
		location = variable.HomeDir
	}

	if _, err := os.Stat(location); err != nil {
		return fmt.Errorf("cannot access location : %s", location)
	}

	m.fileModel.filePanels = append(m.fileModel.filePanels, filePanel{
		location:         location,
		sortOptions:      m.fileModel.filePanels[m.filePanelFocusIndex].sortOptions,
		panelMode:        browserMode,
		focusType:        secondFocus,
		directoryRecords: make(map[string]directoryRecord),
		searchBar:        common.GenerateSearchBar(),
	})

	m.fileModel.filePanels[m.filePanelFocusIndex].focusType = noneFocus
	m.fileModel.filePanels[m.filePanelFocusIndex+1].focusType = returnFocusType(m.focusPanel)
	m.filePanelFocusIndex++

	m.recalcPanelWidths()
	return nil
}

// Close current focus file panel
func (m *model) closeFilePanel() {
	if len(m.fileModel.filePanels) == 1 {
		return
	}

	m.fileModel.filePanels = append(m.fileModel.filePanels[:m.filePanelFocusIndex],
		m.fileModel.filePanels[m.filePanelFocusIndex+1:]...)

	if m.filePanelFocusIndex != 0 {
		m.filePanelFocusIndex--
	}

	m.fileModel.filePanels[m.filePanelFocusIndex].focusType = returnFocusType(m.focusPanel)
	m.recalcPanelWidths()
}

func (m *model) toggleFilePreviewPanel() {
	m.fileModel.filePreview.open = !m.fileModel.filePreview.open
	m.recalcPanelWidths()
}

// toggleFolderPanel hides or shows the left folder panel (alt+1).
func (m *model) toggleFolderPanel() {
	m.folderPanelOpen = !m.folderPanelOpen
	if !m.folderPanelOpen && m.activeFileArea == folderPanelActive {
		// Switch focus to tree if it's open, otherwise nothing
		if m.treePanel.open {
			m.setTreePanelActive()
		}
	}
	m.recalcPanelWidths()
}

// toggleTreePanel hides or shows the middle tree panel (alt+2).
func (m *model) toggleTreePanel() {
	m.treePanel.open = !m.treePanel.open
	if !m.treePanel.open && m.activeFileArea == treePanelActive {
		// Switch focus to folder panel if it's open
		if m.folderPanelOpen {
			m.setFolderPanelActive()
		}
	}
	m.recalcPanelWidths()
}

// Focus on next file panel
func (m *model) nextFilePanel() {
	m.fileModel.filePanels[m.filePanelFocusIndex].focusType = noneFocus
	if m.filePanelFocusIndex == (len(m.fileModel.filePanels) - 1) {
		m.filePanelFocusIndex = 0
	} else {
		m.filePanelFocusIndex++
	}

	m.fileModel.filePanels[m.filePanelFocusIndex].focusType = returnFocusType(m.focusPanel)
}

// Focus on previous file panel
func (m *model) previousFilePanel() {
	m.fileModel.filePanels[m.filePanelFocusIndex].focusType = noneFocus
	if m.filePanelFocusIndex == 0 {
		m.filePanelFocusIndex = (len(m.fileModel.filePanels) - 1)
	} else {
		m.filePanelFocusIndex--
	}

	m.fileModel.filePanels[m.filePanelFocusIndex].focusType = returnFocusType(m.focusPanel)
}

// Focus on sidebar
func (m *model) focusOnSideBar() {
	if common.Config.SidebarWidth == 0 {
		return
	}
	if m.focusPanel == sidebarFocus {
		m.focusPanel = nonePanelFocus
		m.fileModel.filePanels[m.filePanelFocusIndex].focusType = focus
	} else {
		m.focusPanel = sidebarFocus
		m.fileModel.filePanels[m.filePanelFocusIndex].focusType = secondFocus
	}
}

// Focus on processbar
func (m *model) focusOnProcessBar() {
	if !m.toggleFooter {
		return
	}

	if m.focusPanel == processBarFocus {
		m.focusPanel = nonePanelFocus
		m.fileModel.filePanels[m.filePanelFocusIndex].focusType = focus
	} else {
		m.focusPanel = processBarFocus
		m.fileModel.filePanels[m.filePanelFocusIndex].focusType = secondFocus
	}
}

// focus on metadata
func (m *model) focusOnMetadata() {
	if !m.toggleFooter {
		return
	}

	if m.focusPanel == metadataFocus {
		m.focusPanel = nonePanelFocus
		m.fileModel.filePanels[m.filePanelFocusIndex].focusType = focus
	} else {
		m.focusPanel = metadataFocus
		m.fileModel.filePanels[m.filePanelFocusIndex].focusType = secondFocus
	}
}
