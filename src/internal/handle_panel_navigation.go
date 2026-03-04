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

// toggleTree1Panel hides or shows the left tree panel (alt+1).
func (m *model) toggleTree1Panel() {
	m.treePanels[0].open = !m.treePanels[0].open
	if !m.treePanels[0].open && m.activeFileArea == tree1PanelActive {
		if m.treePanels[1].open {
			m.setTree2PanelActive()
		}
	}
	m.recalcPanelWidths()
}

// toggleTree2Panel hides or shows the right tree panel (alt+2).
func (m *model) toggleTree2Panel() {
	m.treePanels[1].open = !m.treePanels[1].open
	if !m.treePanels[1].open && m.activeFileArea == tree2PanelActive {
		if m.treePanels[0].open {
			m.setTree1PanelActive()
		}
	}
	m.recalcPanelWidths()
}

// toggleDetailView switches panel idx between tree mode and detail mode.
// When entering detail mode the entries are loaded from the opposite panel's
// root directory.
func (m *model) toggleDetailView(idx int) {
	tree := &m.treePanels[idx]
	if tree.mode == treePanelModeDetail {
		tree.mode = treePanelModeTree
		return
	}
	otherIdx := 1 - idx
	source := &m.treePanels[otherIdx]
	root := source.root
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
		if m.treePanels[0].open {
			m.setTree1PanelActive()
		} else {
			m.setTree2PanelActive()
		}
		return
	}
	if m.focusPanel == nonePanelFocus && m.activeFileArea == tree1PanelActive && m.treePanels[1].open {
		m.setTree2PanelActive()
	}
}

// focusPreviousPanel moves keyboard focus one panel to the left.
func (m *model) focusPreviousPanel() {
	if m.focusPanel == nonePanelFocus {
		if m.activeFileArea == tree2PanelActive && m.treePanels[0].open {
			m.setTree1PanelActive()
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
