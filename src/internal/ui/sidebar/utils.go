package sidebar

func (d directory) IsDivider() bool {
	return d == placesDividerDir || d == networkDividerDir || d == devicesDividerDir
}

func (d directory) RequiredHeight() int {
	if d.IsDivider() {
		return 2
	}
	return 1
}

// NoActualDir returns true when only dividers are present (no navigable directories).
func (s *Model) NoActualDir() bool {
	for _, d := range s.directories {
		if !d.IsDivider() {
			return false
		}
	}
	return true
}

func (s *Model) isCursorInvalid() bool {
	return s.cursor < 0 || s.cursor >= len(s.directories) || s.directories[s.cursor].IsDivider()
}

func (s *Model) resetCursor() {
	s.cursor = 0
	for i, d := range s.directories {
		if !d.IsDivider() {
			s.cursor = i
			return
		}
	}
	// If all directories are dividers, cursor stays at 0
}

// SearchBarFocused returns whether the search bar is focused
func (s *Model) SearchBarFocused() bool {
	return s.searchBar.Focused()
}

// SearchBarBlur removes focus from the search bar
func (s *Model) SearchBarBlur() {
	s.searchBar.Blur()
}

// SearchBarFocus sets focus on the search bar
func (s *Model) SearchBarFocus() {
	s.searchBar.Focus()
}

// IsRenaming returns whether the sidebar is currently in renaming mode
func (s *Model) IsRenaming() bool {
	return s.renaming
}

// GetCurrentDirectoryLocation returns the location of the currently selected directory
func (s *Model) GetCurrentDirectoryLocation() string {
	if s.isCursorInvalid() || s.NoActualDir() {
		return ""
	}
	return s.directories[s.cursor].Location
}
