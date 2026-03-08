package internal

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"image"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	"github.com/fsncps/hyperfile/src/internal/ui"
	"github.com/fsncps/hyperfile/src/internal/ui/rendering"

	"github.com/fsncps/hyperfile/src/internal/common"
	"github.com/fsncps/hyperfile/src/internal/utils"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/term/ansi"
	"github.com/fsncps/hyperfile/src/config/icon"
	"github.com/yorukot/ansichroma"
)

func (m *model) sidebarRender() string {
	return m.sidebarModel.Render(m.mainPanelHeight, m.focusPanel == sidebarFocus,
		m.fileModel.filePanels[m.filePanelFocusIndex].location)
}

// filePanelRender renders only the first file panel (the folder panel) in the
// new 3-panel layout. The folder panel uses m.fileModel.width as its inner width.
func (m *model) filePanelRender() string {
	if len(m.fileModel.filePanels) == 0 {
		return ""
	}
	panel := m.fileModel.filePanels[0]
	// Validate cursor bounds
	if panel.cursor > len(panel.element)-1 {
		panel.cursor = 0
		panel.render = 0
	}
	m.fileModel.filePanels[0] = panel

	focused := panel.focusType != noneFocus
	return panel.Render(m.mainPanelHeight, m.fileModel.width, focused)
}

func (panel *filePanel) Render(mainPanelHeight int, filePanelWidth int, focussed bool) string {
	r := ui.FilePanelRenderer(mainPanelHeight+2, filePanelWidth+2, focussed)

	panel.renderTopBar(r, filePanelWidth)
	panel.renderSearchBar(r)
	panel.renderFooter(r)
	panel.renderFileEntries(r, mainPanelHeight, filePanelWidth)

	return r.Render()
}

func (panel *filePanel) renderTopBar(r *rendering.Renderer, filePanelWidth int) {
	// TODO - Add ansitruncate left in renderer and remove truncation here
	truncatedPath := common.TruncateTextBeginning(panel.location, filePanelWidth-4, "...")
	r.AddLines(common.FilePanelTopDirectoryIcon + common.FilePanelTopPathStyle.Render(truncatedPath))
	r.AddSection()
}

func (panel *filePanel) renderSearchBar(r *rendering.Renderer) {
	r.AddLines(" " + panel.searchBar.View())
}

// TODO : Unit test this
func (panel *filePanel) renderFooter(r *rendering.Renderer) {
	sortLabel, sortIcon := panel.getSortInfo()
	modeLabel, modeIcon := panel.getPanelModeInfo()
	cursorStr := panel.getCursorString()

	if common.Config.Nerdfont {
		sortLabel = sortIcon + icon.Space + sortLabel
		modeLabel = modeIcon + icon.Space + modeLabel
	} else {
		// TODO : Figure out if we can set icon.Space to " " if nerdfont is false
		// That would simplify code
		sortLabel = sortIcon + " " + sortLabel
	}

	if common.Config.ShowPanelFooterInfo {
		r.SetBorderInfoItems(sortLabel, modeLabel, cursorStr)
		if r.AreInfoItemsTruncated() {
			r.SetBorderInfoItems(sortIcon, modeIcon, cursorStr)
		}
	} else {
		r.SetBorderInfoItems(cursorStr)
	}
}

func (panel *filePanel) renderFileEntries(r *rendering.Renderer, mainPanelHeight, filePanelWidth int) {
	if len(panel.element) == 0 {
		r.AddLines(common.FilePanelNoneText)
		return
	}

	end := min(panel.render+panelElementHeight(mainPanelHeight), len(panel.element))

	for i := panel.render; i < end; i++ {
		// TODO : Fix this, this is O(n^2) complexity. Considered a file panel with 200 files, and 100 selected
		// We will be doing a search in 100 item slice for all 200 files.
		isSelected := arrayContains(panel.selected, panel.element[i].location)

		if panel.renaming && i == panel.cursor {
			r.AddLines(panel.rename.View())
			continue
		}

		cursor := " "
		if i == panel.cursor && !panel.searchBar.Focused() {
			cursor = icon.Cursor
		}

		// Performance TODO: Remove or cache this if not needed at render time
		// This will unnecessarily slow down rendering. There should be a way to avoid this at render
		_, err := os.ReadDir(panel.element[i].location)
		dirExists := err == nil || panel.element[i].directory

		renderedName := common.PrettierName(
			panel.element[i].name,
			filePanelWidth-5,
			dirExists,
			isSelected,
			common.FilePanelBGColor,
		)

		r.AddLines(common.FilePanelCursorStyle.Render(cursor+" ") + renderedName)
	}
}

func (panel *filePanel) getSortInfo() (string, string) {
	opts := panel.sortOptions.data
	selected := opts.options[opts.selected]
	label := selected
	if selected == string(sortingDateModified) {
		label = "Date"
	}

	iconStr := icon.SortAsc

	if opts.reversed {
		iconStr = icon.SortDesc
	}
	return label, iconStr
}

func (panel *filePanel) getPanelModeInfo() (string, string) {
	switch panel.panelMode {
	case browserMode:
		return "Browser", icon.Browser
	case selectMode:
		return "Select", icon.Select
	default:
		return "", ""
	}
}

func (panel *filePanel) getCursorString() string {
	cursor := panel.cursor
	if len(panel.element) > 0 {
		cursor++ // Convert to 1-based
	}
	return fmt.Sprintf("%d/%d", cursor, len(panel.element))
}

func (m *model) processBarRender() string {
	return m.processBarModel.Render(m.focusPanel == processBarFocus)
}

func (m *model) clipboardRender() string {
	// render
	var bottomWidth int
	if m.fullWidth%3 != 0 {
		bottomWidth = utils.FooterWidth(m.fullWidth + m.fullWidth%3 + 2)
	} else {
		bottomWidth = utils.FooterWidth(m.fullWidth)
	}
	r := ui.ClipboardRenderer(m.footerHeight+2, bottomWidth+2)
	if len(m.copyItems.items) == 0 {
		// TODO move this to a string
		r.AddLines("", " "+icon.Error+"  No content in clipboard")
	} else {
		for i := 0; i < len(m.copyItems.items) && i < m.footerHeight; i++ {
			if i == m.footerHeight-1 && i != len(m.copyItems.items)-1 {
				// Last Entry we can render, but there are more that one left
				r.AddLines(strconv.Itoa(len(m.copyItems.items)-i) + " item left....")
			} else {
				fileInfo, err := os.Stat(m.copyItems.items[i])
				if err != nil {
					slog.Error("Clipboard render function get item state ", "error", err)
				}
				if !os.IsNotExist(err) {
					// TODO : There is an inconsistency in parameter that is being passed,
					// and its name in ClipboardPrettierName function
					r.AddLines(common.ClipboardPrettierName(m.copyItems.items[i],
						utils.FooterWidth(m.fullWidth)-3, fileInfo.IsDir(), false))
				}
			}
		}
	}
	return r.Render()
}

func (m *model) terminalSizeWarnRender() string {
	fullWidthString := strconv.Itoa(m.fullWidth)
	fullHeightString := strconv.Itoa(m.fullHeight)
	minimumWidthString := strconv.Itoa(common.MinimumWidth)
	minimumHeightString := strconv.Itoa(common.MinimumHeight)
	if m.fullHeight < common.MinimumHeight {
		fullHeightString = common.TerminalTooSmall.Render(fullHeightString)
	}
	if m.fullWidth < common.MinimumWidth {
		fullWidthString = common.TerminalTooSmall.Render(fullWidthString)
	}
	fullHeightString = common.TerminalCorrectSize.Render(fullHeightString)
	fullWidthString = common.TerminalCorrectSize.Render(fullWidthString)

	heightString := common.MainStyle.Render(" Height = ")
	return common.FullScreenStyle(m.fullHeight, m.fullWidth).Render(`Terminal size too small:` + "\n" +
		"Width = " + fullWidthString +
		heightString + fullHeightString + "\n\n" +
		"Needed for current config:" + "\n" +
		"Width = " + common.TerminalCorrectSize.Render(minimumWidthString) +
		heightString + common.TerminalCorrectSize.Render(minimumHeightString))
}

func (m *model) terminalSizeWarnAfterFirstRender() string {
	minimumWidthInt := common.Config.SidebarWidth + 20*len(m.fileModel.filePanels) + 20 - 1
	minimumWidthString := strconv.Itoa(minimumWidthInt)
	fullWidthString := strconv.Itoa(m.fullWidth)
	fullHeightString := strconv.Itoa(m.fullHeight)
	minimumHeightString := strconv.Itoa(common.MinimumHeight)

	if m.fullHeight < common.MinimumHeight {
		fullHeightString = common.TerminalTooSmall.Render(fullHeightString)
	}
	if m.fullWidth < minimumWidthInt {
		fullWidthString = common.TerminalTooSmall.Render(fullWidthString)
	}
	fullHeightString = common.TerminalCorrectSize.Render(fullHeightString)
	fullWidthString = common.TerminalCorrectSize.Render(fullWidthString)

	heightString := common.MainStyle.Render(" Height = ")
	return common.FullScreenStyle(m.fullHeight, m.fullWidth).Render(`You change your terminal size too small:` + "\n" +
		"Width = " + fullWidthString +
		heightString + fullHeightString + "\n\n" +
		"Needed for current config:" + "\n" +
		"Width = " + common.TerminalCorrectSize.Render(minimumWidthString) +
		heightString + common.TerminalCorrectSize.Render(minimumHeightString))
}

func (m *model) typineModalRender() string {
	previewPath := filepath.Join(m.typingModal.location, m.typingModal.textInput.Value())

	fileLocation := common.FilePanelTopDirectoryIconStyle.Render(" "+icon.Directory+icon.Space) +
		common.FilePanelTopPathStyle.Render(common.TruncateTextBeginning(previewPath, common.ModalWidth-4, "...")) + "\n"

	confirm := common.ModalConfirm.Render(" (" + common.Hotkeys.ConfirmTyping[0] + ") Create ")
	cancel := common.ModalCancel.Render(" (" + common.Hotkeys.CancelTyping[0] + ") Cancel ")

	tip := confirm +
		lipgloss.NewStyle().Background(common.ModalBGColor).Render("           ") +
		cancel

	var err string
	if m.typingModal.errorMesssage != "" {
		err = "\n\n" + common.ModalErrorStyle.Render(m.typingModal.errorMesssage)
	}
	// TODO : Move this all to rendering package to avoid specifying newlines manually
	return common.ModalBorderStyle(common.ModalHeight, common.ModalWidth).
		Render(fileLocation + "\n" + m.typingModal.textInput.View() + "\n\n" + tip + err)
}

func (m *model) introduceModalRender() string {
	title := common.SidebarTitleStyle.Render(" Thanks for using superfile!!") +
		common.ModalStyle.Render("\n You can read the following information before starting to use it!")
	vimUserWarn := common.ProcessErrorStyle.Render("  ** Very importantly ** If you are a Vim/Nvim user, go to:\n" +
		"  https://superfile.netlify.app/configure/custom-hotkeys/ to change your hotkey settings!")
	subOne := common.SidebarTitleStyle.Render("  (1)") +
		common.ModalStyle.Render(" If this is your first time, make sure you read:\n"+
			"      https://superfile.netlify.app/getting-started/tutorial/")
	subTwo := common.SidebarTitleStyle.Render("  (2)") +
		common.ModalStyle.Render(" If you forget the relevant keys during use,\n"+
			"      you can press \"?\" (shift+/) at any time to query the keys!")
	subThree := common.SidebarTitleStyle.Render("  (3)") +
		common.ModalStyle.Render(" For more customization you can refer to:\n"+
			"      https://superfile.netlify.app/")
	subFour := common.SidebarTitleStyle.Render("  (4)") +
		common.ModalStyle.Render(" Thank you again for using superfile.\n"+
			"      If you have any questions, please feel free to ask at:\n"+
			"      https://github.com/fsncps/hyperfile\n"+
			"      Of course, you can always open a new issue to share your idea \n"+
			"      or report a bug!")
	return common.FirstUseModal(m.helpMenu.height, m.helpMenu.width).
		Render(title + "\n\n" + vimUserWarn + "\n\n" + subOne + "\n\n" +
			subTwo + "\n\n" + subThree + "\n\n" + subFour + "\n\n")
}

func (m *model) promptModalRender() string {
	return m.promptModal.Render()
}

func (m *model) helpMenuRender() string {
	contentWidth := max(20, m.helpMenu.width-2)
	cursorColWidth := 2
	columnGapWidth := 2
	columnAreaWidth := max(12, contentWidth-cursorColWidth-(2*columnGapWidth))
	keyColWidth, nameColWidth, descColWidth := m.getHelpMenuColumnWidths(columnAreaWidth)
	helpMenuContent := m.getHelpMenuContent(contentWidth, cursorColWidth,
		columnGapWidth, keyColWidth, nameColWidth, descColWidth)

	selectableCount := 0
	selectedIdx := 0
	for i, data := range m.helpMenu.data {
		if data.subTitle != "" {
			continue
		}
		selectableCount++
		if i <= m.helpMenu.cursor {
			selectedIdx = selectableCount
		}
	}
	if selectableCount == 0 {
		selectedIdx = 0
	}

	bottomBorder := common.GenerateFooterBorder(
		fmt.Sprintf("%d/%d", selectedIdx, selectableCount),
		m.helpMenu.width-2,
	)

	return common.HelpMenuModalBorderStyle(m.helpMenu.height, m.helpMenu.width, bottomBorder).Render(helpMenuContent)
}

func (m *model) getHelpMenuColumnWidths(columnAreaWidth int) (int, int, int) {
	if columnAreaWidth <= 0 {
		return len("Hotkey"), len("Name"), len("Description")
	}

	const desiredKeyColWidth = 15
	const desiredNameColWidth = 30
	const desiredDescColWidth = 50

	keyColWidth := min(desiredKeyColWidth, columnAreaWidth)
	remainingAfterKey := max(0, columnAreaWidth-keyColWidth)
	nameColWidth := min(desiredNameColWidth, remainingAfterKey)
	remainingAfterName := max(0, remainingAfterKey-nameColWidth)
	descColWidth := min(desiredDescColWidth, remainingAfterName)

	if keyColWidth < len("Hotkey") {
		keyColWidth = min(len("Hotkey"), columnAreaWidth)
	}
	remainingAfterKey = max(0, columnAreaWidth-keyColWidth)

	if nameColWidth < len("Name") {
		nameColWidth = min(len("Name"), remainingAfterKey)
	}
	remainingAfterName = max(0, remainingAfterKey-nameColWidth)

	if descColWidth < len("Description") {
		descColWidth = min(len("Description"), remainingAfterName)
	}

	used := keyColWidth + nameColWidth + descColWidth
	if used < columnAreaWidth {
		descColWidth += columnAreaWidth - used
	}

	return keyColWidth, nameColWidth, descColWidth
}

func (m *model) getHelpMenuContent(contentWidth, cursorColWidth, columnGapWidth,
	keyColWidth, nameColWidth, descColWidth int) string {
	var b strings.Builder

	titleText := " Help"
	title := common.HelpMenuTitleStyle.Render(titleText)
	titleW := ansi.StringWidth(titleText)

	filterPrefix := "🔍 "
	filterPrefixWidth := ansi.StringWidth(filterPrefix)
	availableForFilter := max(6, contentWidth-titleW-1)
	preferredInnerWidth := min(24, max(12, contentWidth/3))
	filterInnerWidth := min(preferredInnerWidth, max(4, availableForFilter-2))
	filterTextWidth := max(1, filterInnerWidth-filterPrefixWidth)
	filterText := truncateAndPad(m.helpMenu.filter, filterTextWidth)
	filterInner := truncateAndPad(filterPrefix+filterText, filterInnerWidth)
	filterBox := "[" + filterInner + "]"
	filterW := ansi.StringWidth(filterBox)

	gap := max(1, contentWidth-titleW-filterW)
	b.WriteString(title)
	b.WriteString(strings.Repeat(" ", gap))
	b.WriteString(common.HelpMenuHotkeyStyle.Render(filterBox))
	b.WriteString("\n")

	headerPrefix := strings.Repeat(" ", cursorColWidth)
	headKey := truncateAndPad("Hotkey", keyColWidth)
	headName := truncateAndPad("Name", nameColWidth)
	headDesc := truncateAndPad("Description", descColWidth)
	headGap := strings.Repeat(" ", columnGapWidth)
	b.WriteString(common.HelpMenuTitleStyle.Render(headerPrefix + headKey + headGap + headName + headGap + headDesc))

	for i := m.helpMenu.renderIndex; i < m.helpMenu.height+m.helpMenu.renderIndex && i < len(m.helpMenu.data); i++ {
		b.WriteString("\n")

		if m.helpMenu.data[i].subTitle != "" {
			b.WriteString(common.HelpMenuTitleStyle.Render(strings.Repeat(" ", cursorColWidth) + m.helpMenu.data[i].subTitle))
			continue
		}

		hotkeyRaw := strings.Join(m.helpMenu.data[i].hotkey, " | ")
		nameRaw := m.helpMenu.data[i].name
		descRaw := m.helpMenu.data[i].description

		hotkeyCol := common.HelpMenuHotkeyStyle.Render(truncateAndPad(hotkeyRaw, keyColWidth))
		nameCol := common.ModalStyle.Render(truncateAndPad(nameRaw, nameColWidth))
		descCol := common.ModalStyle.Render(truncateAndPad(descRaw, descColWidth))

		cursor := strings.Repeat(" ", cursorColWidth)
		if m.helpMenu.cursor == i {
			cursor = common.FilePanelCursorStyle.Render(icon.Cursor + " ")
		}
		b.WriteString(cursor)
		b.WriteString(hotkeyCol)
		b.WriteString(headGap)
		b.WriteString(nameCol)
		b.WriteString(headGap)
		b.WriteString(descCol)
	}
	return b.String()
}

func truncateAndPad(s string, width int) string {
	if width <= 0 {
		return ""
	}
	truncated := ansi.Truncate(s, width, "...")
	return padToWidth(truncated, width)
}

func padToWidth(s string, width int) string {
	w := ansi.StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func (m *model) sortOptionsRender() string {
	panel := m.fileModel.filePanels[m.filePanelFocusIndex]
	sortOptionsContent := common.ModalTitleStyle.Render(" Sort Options") + "\n\n"
	for i, option := range panel.sortOptions.data.options {
		cursor := " "
		if i == panel.sortOptions.cursor {
			cursor = common.FilePanelCursorStyle.Render(icon.Cursor)
		}
		sortOptionsContent += cursor + common.ModalStyle.Render(" "+option) + "\n"
	}
	bottomBorder := common.GenerateFooterBorder(fmt.Sprintf("%s/%s", strconv.Itoa(panel.sortOptions.cursor+1),
		strconv.Itoa(len(panel.sortOptions.data.options))), panel.sortOptions.width-2)

	return common.SortOptionsModalBorderStyle(panel.sortOptions.height, panel.sortOptions.width,
		bottomBorder).Render(sortOptionsContent)
}

func readFileContent(filepath string, maxLineLength int, previewLine int) (string, error) {
	var resultBuilder strings.Builder
	file, err := os.Open(filepath)
	if err != nil {
		return resultBuilder.String(), err
	}
	defer file.Close()

	reader := transform.NewReader(file, unicode.BOMOverride(unicode.UTF8.NewDecoder()))
	scanner := bufio.NewScanner(reader)
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		line = ansi.Truncate(line, maxLineLength, "")
		resultBuilder.WriteString(line)
		resultBuilder.WriteRune('\n')
		lineCount++
		if previewLine > 0 && lineCount >= previewLine {
			break
		}
	}
	// returns the first non-EOF error that was encountered by the [Scanner]
	return resultBuilder.String(), scanner.Err()
}

func (m *model) filePreviewPanelRender() string {
	// Recompute preview width to exactly fill remaining horizontal space.
	// This corrects for integer-division rounding in recalcPanelWidths().
	used := common.Config.SidebarWidth + 2 // sidebar outer width
	if m.primaryPanel.open {
		used += m.primaryPanel.width + 2
	}
	if m.secondaryPanel.open {
		used += m.secondaryPanel.width + 2
	}
	m.fileModel.filePreview.width = m.fullWidth - used
	if m.fileModel.filePreview.width < 4 {
		m.fileModel.filePreview.width = 4
	}

	return m.filePreviewPanelRenderWithDimensions(m.mainPanelHeight+2, m.fileModel.filePreview.width)
}

// Helper function to handle empty panel case
func (m *model) renderEmptyFilePreview(r *rendering.Renderer) string {
	return r.Render()
}

// Helper function to handle file info errors
func (m *model) renderFileInfoError(r *rendering.Renderer, _ lipgloss.Style, err error) string {
	slog.Error("Error get file info", "error", err)
	return r.Render()
}

// Helper function to handle unsupported formats
func (m *model) renderUnsupportedFormat(box lipgloss.Style) string {
	return box.Render(common.FilePreviewUnsupportedFormatText)
}

// Helper function to handle unsupported mode
func (m *model) renderUnsupportedFileMode(r *rendering.Renderer) string {
	r.AddLines(common.FilePreviewUnsupportedFileMode)
	return r.Render()
}

// renderDirectoryPreview shows folder metadata instead of a raw file listing.
func (m *model) renderDirectoryPreview(r *rendering.Renderer, itemPath string, previewHeight int) string {
	info, err := os.Stat(itemPath)
	if err != nil {
		slog.Error("Error render directory preview", "error", err)
		r.AddLines(common.FilePreviewDirectoryUnreadableText)
		return r.Render()
	}

	entries, err := os.ReadDir(itemPath)
	if err != nil {
		r.AddLines(common.FilePreviewDirectoryUnreadableText)
		return r.Render()
	}

	// Count and size pass over direct children only (fast, no deep walk).
	var fileCount, dirCount, hiddenCount int
	var totalSize int64
	for _, e := range entries {
		if len(e.Name()) > 0 && e.Name()[0] == '.' {
			hiddenCount++
			continue
		}
		if e.IsDir() {
			dirCount++
		} else {
			fileCount++
			if fi, err2 := e.Info(); err2 == nil {
				totalSize += fi.Size()
			}
		}
	}

	label := lipgloss.NewStyle().Foreground(common.FilePanelBorderActiveColor).Background(common.FilePanelBGColor)
	value := common.FilePanelStyle

	row := func(lbl, val string) string {
		return label.Render(fmt.Sprintf("  %-12s", lbl)) + value.Render(val)
	}

	itemStr := fmt.Sprintf("%d files, %d dirs", fileCount, dirCount)
	if hiddenCount > 0 {
		itemStr += fmt.Sprintf(" (+%d hidden)", hiddenCount)
	}

	sizeStr := common.FormatFileSize(totalSize)
	if dirCount > 0 {
		sizeStr += "  (files only)"
	}

	modStr := info.ModTime().Format("2006-01-02  15:04")
	modeStr := info.Mode().String()

	r.AddLines(row("Items", itemStr))
	r.AddLines(row("Size", sizeStr))
	r.AddLines(row("Modified", modStr))
	r.AddLines(row("Mode", modeStr))

	return r.Render()
}

// renderImagePreview renders an image file into the preview panel.
// r already has the header section added (if applicable); imageW/imageH are the
// cell dimensions available for the image content; headerRows is 0 or 2.
func (m *model) renderImagePreview(r *rendering.Renderer, box lipgloss.Style, itemPath string,
	imageW, imageH, sideAreaWidth, headerRows int, clearCmd string) string {
	if !m.fileModel.filePreview.open {
		return box.Render("\n --- Preview panel is closed ---")
	}

	if !common.Config.ShowImagePreview {
		return box.Render("\n --- Image preview is disabled ---")
	}

	imageRender, err := m.imagePreviewer.ImagePreview(itemPath, imageW, imageH,
		common.Theme.FilePanelBG, sideAreaWidth)
	if errors.Is(err, image.ErrFormat) {
		return box.Render("\n --- " + icon.Error + " Unsupported image formats ---")
	}

	if err != nil {
		slog.Error("Error convert image to ansi", "error", err)
		return box.Render("\n --- " + icon.Error + " Error convert image to ansi ---")
	}

	if strings.HasPrefix(imageRender, "\x1b_G") {
		// Kitty protocol draws at cursor position via terminal escape sequences.
		// Emit the full panel first (header + empty rows + borders), then use an
		// absolute cursor-position command to place the image below the header.
		// imageStartRow (1-indexed) = top-border(1) + headerRows + 1
		// imageStartCol (1-indexed) = sideAreaWidth(left-border col) + 1
		positionCmd := fmt.Sprintf("\x1b[%d;%dH", headerRows+2, sideAreaWidth+1)
		return r.Render() + positionCmd + imageRender
	}

	// ANSI art: each line is colored characters that fit within the content width.
	r.AddLines(imageRender)
	return r.Render() + clearCmd
}

// Helper function to handle text file preview
func (m *model) renderTextPreview(r *rendering.Renderer, box lipgloss.Style, itemPath string,
	previewWidth, previewHeight int) string {
	format := lexers.Match(filepath.Base(itemPath))
	if format == nil {
		isText, err := common.IsTextFile(itemPath)
		if err != nil {
			slog.Error("Error while checking text file", "error", err)
			return box.Render(common.FilePreviewError)
		} else if !isText {
			return box.Render(common.FilePreviewUnsupportedFormatText)
		}
	}

	fileContent, err := readFileContent(itemPath, previewWidth, previewHeight)
	if err != nil {
		slog.Error("Error open file", "error", err)
		return box.Render(common.FilePreviewError)
	}

	if fileContent == "" {
		return box.Render(common.FilePreviewEmptyText)
	}

	if format != nil {
		background := ""
		if !common.Config.TransparentBackground {
			background = common.Theme.FilePanelBG
		}
		if common.Config.CodePreviewer == "bat" {
			if batCmd == "" {
				return box.Render("\n --- " + icon.Error +
					" 'bat' is not installed or not found. ---\n --- Cannot render file preview. ---")
			}
			fileContent, err = getBatSyntaxHighlightedContent(itemPath, previewHeight, background)
		} else {
			fileContent, err = ansichroma.HightlightString(fileContent, format.Config().Name,
				common.Theme.CodeSyntaxHighlightTheme, background)
		}
		if err != nil {
			slog.Error("Error render code highlight", "error", err)
			return box.Render("\n" + common.FilePreviewError)
		}
	}

	r.AddLines(fileContent)
	return r.Render()
}

// getPreviewItemPath returns the filesystem path of the currently selected node
// in the active tree panel.
func (m *model) getPreviewItemPath() string {
	tree := m.treePanelByIndex(int(m.activeFileArea))
	if tree.open {
		if node := tree.GetSelectedNode(); node != nil {
			return node.path
		}
	}
	return ""
}

func (m *model) filePreviewPanelRenderWithDimensions(previewHeight int, previewWidth int) string {
	box := common.FilePreviewBox(previewHeight, previewWidth)
	r := ui.FilePreviewPanelRenderer(previewHeight, previewWidth)
	clearCmd := m.imagePreviewer.ClearKittyImages()
	if m.helpMenu.open {
		return r.Render() + clearCmd
	}

	itemPath := m.getPreviewItemPath()
	if itemPath == "" {
		return m.renderEmptyFilePreview(r) + clearCmd
	}

	// Debounce: show empty panel until the cursor has been idle for previewDebounceDuration.
	if time.Since(m.lastCursorMovedAt) < previewDebounceDuration {
		return r.Render() + clearCmd
	}

	// Header: always show item name for panels large enough to have content below it.
	// Threshold > 7 excludes unit-test dimensions (≤7) while always firing in real usage (≥16).
	if previewHeight > 7 {
		name := filepath.Base(itemPath)
		iconW := ansi.StringWidth(common.FilePanelTopDirectoryIcon)
		truncated := common.TruncateTextBeginning(name, r.ContentWidth()-iconW, "...")
		r.AddLines(common.FilePanelTopDirectoryIcon + common.FilePanelTopPathStyle.Render(truncated))
		r.AddSection()
	}

	fileInfo, infoErr := os.Stat(itemPath)
	if infoErr != nil {
		return m.renderFileInfoError(r, box, infoErr) + clearCmd
	}
	slog.Debug("Attempting to render preview", "itemPath", itemPath,
		"mode", fileInfo.Mode().String(), "isRegular", fileInfo.Mode().IsRegular())

	// For non regular files which are not directories Dont try to read them
	// See Issue
	if !fileInfo.Mode().IsRegular() && (fileInfo.Mode()&fs.ModeDir) == 0 {
		return m.renderUnsupportedFileMode(r) + clearCmd
	}

	ext := filepath.Ext(itemPath)
	if slices.Contains(common.UnsupportedPreviewFormats, ext) {
		return m.renderUnsupportedFormat(box) + clearCmd
	}

	if fileInfo.IsDir() {
		return m.renderDirectoryPreview(r, itemPath, previewHeight) + clearCmd
	}

	if isImageFile(itemPath) {
		// headerRows = number of lines consumed by the header section (0 when header not shown).
		headerRows := 0
		if previewHeight > 7 {
			headerRows = 2 // 1 header line + 1 section divider
		}
		imageW := previewWidth - 4               // 1-col margin each side inside border
		imageH := previewHeight - 2 - headerRows // inner height minus header
		sideAreaWidth := m.fullWidth - previewWidth + 1
		return m.renderImagePreview(r, box, itemPath, imageW, imageH, sideAreaWidth, headerRows, clearCmd)
	}

	return m.renderTextPreview(r, box, itemPath, previewWidth, previewHeight) + clearCmd
}

func getBatSyntaxHighlightedContent(itemPath string, previewLine int, background string) (string, error) {
	// --plain: use the plain style without line numbers and decorations
	// --force-colorization: force colorization for non-interactive shell output
	// --line-range <:m>: only read from line 1 to line "m"
	batArgs := []string{itemPath, "--plain", "--force-colorization", "--line-range", fmt.Sprintf(":%d", previewLine-1)}

	// set timeout for the external command execution to 500ms max
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, batCmd, batArgs...)

	fileContentBytes, err := cmd.Output()
	if err != nil {
		slog.Error("Error render code highlight", "error", err)
		return "", err
	}

	fileContent := string(fileContentBytes)
	if !common.Config.TransparentBackground {
		fileContent = setBatBackground(fileContent, background)
	}
	return fileContent, nil
}

func setBatBackground(input string, background string) string {
	tokens := strings.Split(input, "\x1b[0m")
	backgroundStyle := lipgloss.NewStyle().Background(lipgloss.Color(background))
	for idx, token := range tokens {
		tokens[idx] = backgroundStyle.Render(token)
	}
	return strings.Join(tokens, "\x1b[0m")
}
