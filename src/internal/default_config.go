package internal

import (
	"path/filepath"
	"strings"

	"github.com/fsncps/hyperfile/src/internal/ui/metadata"
	"github.com/fsncps/hyperfile/src/internal/ui/processbar"
	"github.com/fsncps/hyperfile/src/internal/ui/sidebar"
	filepreview "github.com/fsncps/hyperfile/src/pkg/file_preview"

	"github.com/fsncps/hyperfile/src/internal/common"
	"github.com/fsncps/hyperfile/src/internal/ui/prompt"
	variable "github.com/fsncps/hyperfile/src/config"
)

// Generate and return model containing default configurations for interface
// Maybe we can replace slice of strings with var args - Should we ?
func defaultModelConfig(toggleDotFile bool, toggleFooter bool, firstUse bool, firstFilePanelDirs []string) *model {
	// Build file panels (kept for file-op infrastructure; not rendered directly)
	panels := filePanelSlice(firstFilePanelDirs)

	// Use first panel's location as cwd, fallback to "/"
	cwd := "/"
	if len(firstFilePanelDirs) > 0 && firstFilePanelDirs[0] != "" {
		cwd = firstFilePanelDirs[0]
	}

	// Primary panel: root is $HOME if cwd is under $HOME, with d=0 and cwd expanded
	primaryRoot := cwd
	if isUnderHome(cwd) {
		primaryRoot = variable.HomeDir
	}
	primaryPanel := defaultTreePanel(primaryRoot)
	primaryPanel.maxDepth = 0
	primaryPanel.showHidden = toggleDotFile
	primaryPanel.showRootNode = true
	primaryPanel.nodes = buildTreeNodesWithRoot(primaryRoot, primaryPanel.maxDepth, primaryPanel.collapsed, primaryPanel.expanded, primaryPanel.showHidden)
	if primaryRoot != cwd {
		primaryPanel.ExpandToPathAndSelect(cwd)
	}

	// Secondary panel: root is cwd with d=0
	secondaryPanel := defaultTreePanel(cwd)
	secondaryPanel.maxDepth = 0
	secondaryPanel.showHidden = toggleDotFile
	secondaryPanel.showRootNode = true
	secondaryPanel.nodes = buildTreeNodesWithRoot(cwd, secondaryPanel.maxDepth, secondaryPanel.collapsed, secondaryPanel.expanded, secondaryPanel.showHidden)

	return &model{
		filePanelFocusIndex: 0,
		focusPanel:          nonePanelFocus,
		activeFileArea:      primaryPanelActive,
		processBarModel:     processbar.New(),
		sidebarModel:        sidebar.New(),
		fileMetaData:        metadata.New(),
		primaryPanel:        primaryPanel,
		secondaryPanel:      secondaryPanel,
		secondaryMode:       secondaryModeTree,
		viewMode:            initialViewMode(common.Config.DefaultOpenFilePreview),
		fileModel: fileModel{
			filePanels: panels,
			filePreview: filePreviewPanel{
				open: common.Config.DefaultOpenFilePreview,
			},
			width: 10,
		},
		helpMenu: helpMenuModal{
			renderIndex: 0,
			cursor:      0,
			filter:      "",
			allData:     getHelpMenuData(),
			data:        getHelpMenuData(),
			open:        false,
		},
		imagePreviewer: filepreview.NewImagePreviewer(),
		promptModal:    prompt.DefaultModel(prompt.PromptMinHeight, prompt.PromptMinWidth),
		modelQuitState: notQuitting,
		toggleDotFile:  toggleDotFile,
		toggleFooter:   toggleFooter,
		firstUse:       firstUse,
	}
}

// isUnderHome returns true if path is under the user's home directory.
func isUnderHome(path string) bool {
	if variable.HomeDir == "" {
		return false
	}
	rel, err := filepath.Rel(variable.HomeDir, path)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..") && rel != "." && rel != ""
}

// initialViewMode returns the default view mode based on preview preference
func initialViewMode(previewOpen bool) viewModeType {
	if previewOpen {
		return viewModeBothWithPreview
	}
	return viewModeBothNoPreview
}

// Return help menu for Hotkeys
func getHelpMenuData() []helpMenuModalData {
	if len(common.HotkeyDisplayGroups) == 0 {
		return []helpMenuModalData{}
	}

	data := make([]helpMenuModalData, 0, 64)
	for _, group := range common.HotkeyDisplayGroups {
		data = append(data, helpMenuModalData{subTitle: group.Title})
		for _, item := range group.Items {
			if item.Key == "" {
				continue
			}
			description := item.Description
			if description == "" {
				description = item.Name
			}
			data = append(data, helpMenuModalData{
				hotkey:         []string{item.Key},
				name:           item.Name,
				description:    description,
				hotkeyWorkType: globalType,
			})
		}
	}
	return data
}
