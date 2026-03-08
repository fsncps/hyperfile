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

// setViewMode configures panel visibility for one of the four fixed layouts:
// 1: both tree panels + preview
// 2: both tree panels, no preview
// 3: main (left) tree panel + preview
// 4: main (left) tree panel only
func (m *model) setViewMode(mode viewModeType) {
	m.viewMode = mode

	switch mode {
	case viewModeBothWithPreview:
		m.primaryPanel.open = true
		m.secondaryPanel.open = true
		m.fileModel.filePreview.open = true
	case viewModeBothNoPreview:
		m.primaryPanel.open = true
		m.secondaryPanel.open = true
		m.fileModel.filePreview.open = false
	case viewModeMainWithPreview:
		m.primaryPanel.open = true
		m.secondaryPanel.open = false
		m.fileModel.filePreview.open = true
	case viewModeMainOnly:
		m.primaryPanel.open = true
		m.secondaryPanel.open = false
		m.fileModel.filePreview.open = false
	}

	// Ensure focus stays on a visible panel
	if !m.primaryPanel.open && m.activeFileArea == primaryPanelActive {
		m.setSecondaryPanelActive()
	}
	if !m.secondaryPanel.open && m.activeFileArea == secondaryPanelActive {
		m.setPrimaryPanelActive()
	}

	m.recalcPanelWidths()
}

// toggleDetailView switches panel idx between tree mode and detail mode.
// When entering detail mode the entries are loaded from the opposite panel's
// currently selected directory (or its root if the cursor is on a file).
func (m *model) toggleDetailView(idx int) {
	tree := m.treePanelByIndex(idx)
	if tree.mode == treePanelModeDetail {
		tree.mode = treePanelModeTree
		return
	}
	source := m.treePanelByIndex(1 - idx)
	root := source.root
	if node := source.GetSelectedNode(); node != nil && node.isDir {
		root = node.path
	}
	tree.detailRoot = root
	tree.detailEntries = buildDetailEntries(root, tree.showHidden)
	tree.cursor = 0
	tree.renderIdx = 0
	tree.mode = treePanelModeDetail
}

// focusNextPanel moves keyboard focus one panel to the right.
func (m *model) focusNextPanel() {
	if m.focusPanel == sidebarFocus {
		m.focusPanel = nonePanelFocus
		if m.primaryPanel.open {
			m.setPrimaryPanelActive()
		} else {
			m.setSecondaryPanelActive()
		}
		return
	}
	if m.focusPanel == nonePanelFocus && m.activeFileArea == primaryPanelActive && m.secondaryPanel.open {
		m.setSecondaryPanelActive()
	}
}

// focusPreviousPanel moves keyboard focus one panel to the left.
func (m *model) focusPreviousPanel() {
	if m.focusPanel == nonePanelFocus {
		if m.activeFileArea == secondaryPanelActive && m.primaryPanel.open {
			m.setPrimaryPanelActive()
		} else {
			m.focusOnSideBar()
		}
	}
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
