package sidebar

import "github.com/charmbracelet/bubbles/textinput"

type directory struct {
	Location string `json:"location"`
	Name     string `json:"name"`
	usage    string // disk-usage display e.g. "45%" — not persisted (unexported)
	pinned   bool   // user-pinned entry (renameable) — not persisted (unexported)
}

type Model struct {
	directories []directory
	renderIndex int
	cursor      int
	rename      textinput.Model
	renaming    bool
	searchBar   textinput.Model
}
