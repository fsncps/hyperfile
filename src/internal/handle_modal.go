package internal

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// Cancel typing modal e.g. create file or directory
func (m *model) cancelTypingModal() {
	m.typingModal.textInput.Blur()
	m.typingModal.open = false
}

// Confirm to create file or directory
func (m *model) createItem() {
	if err := checkFileNameValidity(m.typingModal.textInput.Value()); err != nil {
		m.typingModal.errorMesssage = err.Error()
		slog.Error("Errow while createItem during item creation", "error", err)

		return
	}

	defer func() {
		m.typingModal.errorMesssage = ""
		m.typingModal.open = false
		m.typingModal.textInput.Blur()
	}()

	path := filepath.Join(m.typingModal.location, m.typingModal.textInput.Value())
	if !strings.HasSuffix(m.typingModal.textInput.Value(), string(filepath.Separator)) {
		path, _ = renameIfDuplicate(path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			slog.Error("Error while createItem during directory creation", "error", err)
			return
		}
		f, err := os.Create(path)
		if err != nil {
			slog.Error("Error while createItem during file creation", "error", err)
			return
		}
		defer f.Close()
	} else {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			slog.Error("Error while createItem during directory creation", "error", err)
			return
		}
	}
}

// Cancel rename file or directory
func (m *model) cancelRename() {
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]
	panel.rename.Blur()
	panel.renaming = false
	m.fileModel.renaming = false
}

// Connfirm rename file or directory
func (m *model) confirmRename() {
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]

	// Although we dont expect this to happen based on our current flow
	// Just adding it here to be safe
	if len(panel.element) == 0 {
		slog.Error("confirmRename called on empty panel")
		return
	}

	oldPath := panel.element[panel.cursor].location
	newPath := filepath.Join(panel.location, panel.rename.Value())

	// Rename the file
	err := os.Rename(oldPath, newPath)
	if err != nil {
		slog.Error("Error while confirmRename during rename", "error", err)
		// Dont return. We have to also reset the panel and model information
	}
	m.fileModel.renaming = false
	panel.rename.Blur()
	panel.renaming = false
	m.rebuildAllTrees()
}

func (m *model) openSortOptionsMenu() {
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]
	panel.sortOptions.open = true
}

func (m *model) cancelSortOptions() {
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]
	panel.sortOptions.cursor = panel.sortOptions.data.selected
	panel.sortOptions.open = false
}

func (m *model) confirmSortOptions() {
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]
	panel.sortOptions.data.selected = panel.sortOptions.cursor
	panel.sortOptions.open = false
}

// Move the cursor up in the sort options menu
func (m *model) sortOptionsListUp() {
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]
	if panel.sortOptions.cursor > 0 {
		panel.sortOptions.cursor--
	} else {
		panel.sortOptions.cursor = len(panel.sortOptions.data.options) - 1
	}
}

// Move the cursor down in the sort options menu
func (m *model) sortOptionsListDown() {
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]
	if panel.sortOptions.cursor < len(panel.sortOptions.data.options)-1 {
		panel.sortOptions.cursor++
	} else {
		panel.sortOptions.cursor = 0
	}
}

func (m *model) toggleReverseSort() {
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]
	panel.sortOptions.data.reversed = !panel.sortOptions.data.reversed
}

// Cancel search, this will clear all searchbar input
func (m *model) cancelSearch() {
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]
	panel.searchBar.Blur()
	panel.searchBar.SetValue("")
}

// Confirm search. This will exit the search bar and filter the files
func (m *model) confirmSearch() {
	panel := &m.fileModel.filePanels[m.filePanelFocusIndex]
	panel.searchBar.Blur()
}

// Help menu panel list up
func (m *model) helpMenuListUp() {
	if len(m.helpMenu.data) == 0 {
		return
	}
	if m.helpMenu.cursor <= 0 {
		m.helpMenu.cursor = len(m.helpMenu.data) - 1
	} else {
		m.helpMenu.cursor--
	}
	for len(m.helpMenu.data) > 0 && m.helpMenu.data[m.helpMenu.cursor].subTitle != "" {
		if m.helpMenu.cursor <= 0 {
			m.helpMenu.cursor = len(m.helpMenu.data) - 1
		} else {
			m.helpMenu.cursor--
		}
	}
	if m.helpMenu.cursor < m.helpMenu.renderIndex {
		m.helpMenu.renderIndex = m.helpMenu.cursor
	}
}

// Help menu panel list down
func (m *model) helpMenuListDown() {
	if len(m.helpMenu.data) == 0 {
		return
	}
	if m.helpMenu.cursor >= len(m.helpMenu.data)-1 {
		m.helpMenu.cursor = 0
	} else {
		m.helpMenu.cursor++
	}
	for len(m.helpMenu.data) > 0 && m.helpMenu.data[m.helpMenu.cursor].subTitle != "" {
		if m.helpMenu.cursor >= len(m.helpMenu.data)-1 {
			m.helpMenu.cursor = 0
		} else {
			m.helpMenu.cursor++
		}
	}
	if m.helpMenu.cursor > m.helpMenu.renderIndex+m.helpMenu.height-1 {
		m.helpMenu.renderIndex = m.helpMenu.cursor - m.helpMenu.height + 1
		if m.helpMenu.renderIndex < 0 {
			m.helpMenu.renderIndex = 0
		}
	}
}

// Toggle help menu
func (m *model) openHelpMenu() {
	if m.helpMenu.open {
		m.helpMenu.open = false
		return
	}

	m.helpMenu.allData = getHelpMenuData()
	m.helpMenu.filter = ""
	m.applyHelpMenuFilter()
	m.helpMenu.open = true
}

// Quit help menu
func (m *model) quitHelpMenu() {
	m.helpMenu.open = false
	m.lastCursorMovedAt = time.Now()
}

func (m *model) applyHelpMenuFilter() {
	filter := strings.ToLower(strings.TrimSpace(m.helpMenu.filter))
	if filter == "" {
		m.helpMenu.data = m.helpMenu.allData
		m.helpMenu.renderIndex = 0
		m.helpMenu.cursor = 0
		for m.helpMenu.cursor < len(m.helpMenu.data) && m.helpMenu.data[m.helpMenu.cursor].subTitle != "" {
			m.helpMenu.cursor++
		}
		if m.helpMenu.cursor >= len(m.helpMenu.data) {
			m.helpMenu.cursor = 0
		}
		return
	}

	filtered := make([]helpMenuModalData, 0, len(m.helpMenu.allData))
	for _, row := range m.helpMenu.allData {
		if row.subTitle != "" {
			filtered = append(filtered, row)
			continue
		}
		text := strings.ToLower(strings.Join([]string{
			strings.Join(row.hotkey, " "),
			row.name,
			row.description,
		}, " "))
		if strings.Contains(text, filter) {
			filtered = append(filtered, row)
		}
	}
	cleaned := make([]helpMenuModalData, 0, len(filtered))
	for i := range filtered {
		if filtered[i].subTitle != "" {
			if i == len(filtered)-1 || filtered[i+1].subTitle != "" {
				continue
			}
		}
		cleaned = append(cleaned, filtered[i])
	}
	if len(cleaned) == 0 {
		m.helpMenu.data = []helpMenuModalData{{subTitle: "No matches"}}
		m.helpMenu.cursor = 0
		m.helpMenu.renderIndex = 0
		return
	}
	m.helpMenu.data = cleaned
	m.helpMenu.cursor = 0
	m.helpMenu.renderIndex = 0
	for m.helpMenu.cursor < len(m.helpMenu.data) && m.helpMenu.data[m.helpMenu.cursor].subTitle != "" {
		m.helpMenu.cursor++
	}
	if m.helpMenu.cursor >= len(m.helpMenu.data) {
		m.helpMenu.cursor = 0
	}
}

func (m *model) appendHelpMenuFilterRune(msg string) {
	runes := []rune(msg)
	if len(runes) != 1 {
		return
	}
	r := runes[0]
	if !unicode.IsPrint(r) {
		return
	}
	m.helpMenu.filter += string(r)
	m.applyHelpMenuFilter()
}

func (m *model) deleteHelpMenuFilterRune() {
	if m.helpMenu.filter == "" {
		return
	}
	runes := []rune(m.helpMenu.filter)
	m.helpMenu.filter = string(runes[:len(runes)-1])
	m.applyHelpMenuFilter()
}
