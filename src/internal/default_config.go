package internal

import (
	"github.com/fsncps/hyperfile/src/internal/ui/metadata"
	"github.com/fsncps/hyperfile/src/internal/ui/processbar"
	"github.com/fsncps/hyperfile/src/internal/ui/sidebar"
	filepreview "github.com/fsncps/hyperfile/src/pkg/file_preview"

	"github.com/fsncps/hyperfile/src/internal/common"
	"github.com/fsncps/hyperfile/src/internal/ui/prompt"
)

// Generate and return model containing default configurations for interface
// Maybe we can replace slice of strings with var args - Should we ?
func defaultModelConfig(toggleDotFile bool, toggleFooter bool, firstUse bool, firstFilePanelDirs []string) *model {
	// Build file panels (kept for file-op infrastructure; not rendered directly)
	panels := filePanelSlice(firstFilePanelDirs)

	// Use first panel's location as tree root, fallback to "/"
	treeRoot := "/"
	if len(firstFilePanelDirs) > 0 && firstFilePanelDirs[0] != "" {
		treeRoot = firstFilePanelDirs[0]
	}

	tp0 := defaultTreePanel(treeRoot)
	tp0.showHidden = toggleDotFile
	tp1 := defaultTreePanel(treeRoot)
	tp1.showHidden = toggleDotFile

	return &model{
		filePanelFocusIndex: 0,
		focusPanel:          nonePanelFocus,
		activeFileArea:      tree1PanelActive,
		processBarModel:     processbar.New(),
		sidebarModel:        sidebar.New(),
		fileMetaData:        metadata.New(),
		treePanels:          [2]treePanelModel{tp0, tp1},
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
