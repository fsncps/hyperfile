package internal

import (
	"time"

	"github.com/yorukot/superfile/src/internal/secondary"
	"github.com/yorukot/superfile/src/internal/ui/metadata"
	"github.com/yorukot/superfile/src/internal/ui/notify"
	"github.com/yorukot/superfile/src/internal/ui/processbar"
	"github.com/yorukot/superfile/src/internal/ui/sidebar"
	filepreview "github.com/yorukot/superfile/src/pkg/file_preview"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/yorukot/superfile/src/internal/ui/prompt"
)

type panelMode uint

type filePanelFocusType uint

type focusPanelType int

type hotkeyType int

type modelQuitStateType int

type SecondaryMode int

const (
	SecondaryTree SecondaryMode = iota
	SecondaryDetail
	SecondaryGit
	SecondaryDiskUsage
)

func (m SecondaryMode) String() string {
	switch m {
	case SecondaryTree:
		return "Tree"
	case SecondaryDetail:
		return "Detail"
	case SecondaryGit:
		return "Git"
	case SecondaryDiskUsage:
		return "Disk Usage"
	default:
		return "Unknown"
	}
}

func (m SecondaryMode) SyncsWithPrimary() bool {
	return m != SecondaryTree
}

type SecondaryPanel struct {
	mode      SecondaryMode
	focusType filePanelFocusType

	tree      *treePanelModel
	detail    *detailViewModel
	git       *secondary.GitViewModel
	diskUsage *secondary.DiskUsageModel
}

func (sp *SecondaryPanel) EntryCount() int {
	switch sp.mode {
	case SecondaryTree:
		if sp.tree != nil {
			return sp.tree.EntryCount()
		}
	case SecondaryDetail:
		if sp.detail != nil {
			return len(sp.detail.entries)
		}
	case SecondaryGit:
		if sp.git != nil {
			return sp.git.EntryCount()
		}
	case SecondaryDiskUsage:
		if sp.diskUsage != nil {
			return sp.diskUsage.EntryCount()
		}
	}
	return 0
}

func (sp *SecondaryPanel) IsOpen() bool {
	return sp.tree != nil && sp.tree.open
}

func (sp *SecondaryPanel) SetOpen(open bool) {
	if sp.tree != nil {
		sp.tree.open = open
	}
}

func (sp *SecondaryPanel) ListUp(visibleH int) {
	if sp.tree != nil {
		sp.tree.ListUp(visibleH)
	}
}

func (sp *SecondaryPanel) ListDown(visibleH int) {
	if sp.tree != nil {
		sp.tree.ListDown(visibleH)
	}
}

func (sp *SecondaryPanel) HasSelection() bool {
	if sp.tree != nil {
		return sp.tree.HasSelection()
	}
	return false
}

func (sp *SecondaryPanel) ShiftListDown(visibleH int) {
	if sp.tree != nil {
		sp.tree.ShiftListDown(visibleH)
	}
}

func newSecondaryPanel(root string) SecondaryPanel {
	tree := defaultTreePanel(root)
	return SecondaryPanel{
		mode:      SecondaryTree,
		tree:      &tree,
		detail:    &detailViewModel{},
		git:       secondary.NewGitViewModel(),
		diskUsage: secondary.NewDiskUsageModel(),
	}
}

type detailViewModel struct {
	root       string
	entries    []detailEntry
	cursor     int
	renderIdx  int
	showHidden bool
}

type viewModeType int

//nolint:unused
const (
	viewModeBothWithPreview viewModeType = iota
	viewModeBothNoPreview
	viewModeMainWithPreview
	viewModeMainOnly
)

// TODO: Convert to integer enum
type sortingKind string

const (
	globalType hotkeyType = iota
	normalType
	selectType
)

// Constants for panel with no focus
const (
	nonePanelFocus focusPanelType = iota
	processBarFocus
	sidebarFocus
	metadataFocus
)

// Constants for file panel with no focus
const (
	noneFocus filePanelFocusType = iota
	secondFocus
	focus
)

// Constants for select mode or browser mode
const (
	selectMode panelMode = iota
	browserMode
)

const (
	notQuitting modelQuitStateType = iota
	quitInitiated
	quitConfirmationInitiated
	quitConfirmationReceived
	quitDone
)

// Main model
// TODO : We could consider using *model as tea.Model, instead of model.
// for reducing re-allocations. The struct is 20K bytes. But this could lead to
// issues like race conditions and whatnot, which are hidden since we are creating
// new model in each tea update.
type model struct {
	fileModel       fileModel
	sidebarModel    sidebar.Model
	processBarModel processbar.Model
	focusPanel      focusPanelType
	copyItems       copyItems

	primaryPanel   treePanelModel
	secondaryPanel SecondaryPanel
	activeFileArea fileAreaFocus //nolint:unused
	viewMode       viewModeType  //nolint:unused

	// Modals
	notifyModel notify.Model
	typingModal typingModal
	helpMenu    helpMenuModal
	promptModal prompt.Model

	fileMetaData         metadata.Model
	ioReqCnt             int
	imagePreviewer       *filepreview.ImagePreviewer
	modelQuitState       modelQuitStateType
	firstTextInput       bool
	toggleDotFile        bool
	updatedToggleDotFile bool
	toggleFooter         bool
	firstLoadingComplete bool
	firstUse             bool
	lastCursorMovedAt    time.Time //nolint:unused

	// This entirely disables metadata fetching. Used in test model
	disableMetatdata    bool
	filePanelFocusIndex int

	// Height in number of lines of actual viewport of
	// main panel and sidebar excluding border
	mainPanelHeight int

	// Height in number of lines of actual viewport of
	// footer panels - process/metadata/clipboard - excluding border
	footerHeight int
	fullWidth    int
	fullHeight   int
}

// Modal
type helpMenuModal struct {
	height      int
	width       int
	open        bool
	renderIndex int
	cursor      int
	data        []helpMenuModalData
}

type helpMenuModalData struct {
	hotkey         []string
	description    string
	hotkeyWorkType hotkeyType
	subTitle       string
}

type typingModal struct {
	location      string
	open          bool
	textInput     textinput.Model
	errorMesssage string
}

// Copied items
type copyItems struct {
	items []string
	cut   bool
}

/* FILE WINDOWS TYPE START*/
// Model for file windows
type fileModel struct {
	filePanels   []filePanel
	width        int
	renaming     bool
	maxFilePanel int
	filePreview  filePreviewPanel
}

type filePreviewPanel struct {
	open  bool
	width int
}

// Panel representing a file
type filePanel struct {
	cursor             int
	render             int
	focusType          filePanelFocusType
	location           string
	sortOptions        sortOptionsModel
	panelMode          panelMode
	selected           []string
	element            []element
	directoryRecords   map[string]directoryRecord
	rename             textinput.Model
	renaming           bool
	searchBar          textinput.Model
	lastTimeGetElement time.Time
}

// Sort options
type sortOptionsModel struct {
	width  int
	height int
	open   bool
	cursor int
	data   sortOptionsModelData
}

type sortOptionsModelData struct {
	options  []string
	selected int
	reversed bool
}

// Record for directory navigation
type directoryRecord struct {
	directoryCursor int
	directoryRender int
}

// Element within a file panel
type element struct {
	name      string
	location  string
	directory bool
	metaData  [][2]string
}

/* FILE WINDOWS TYPE END*/

type editorFinishedMsg struct{ err error }

type sliceOrderFunc func(i, j int) bool
