package sidebar

import (
	"log/slog"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fsncps/hyperfile/src/internal/ui"

	"github.com/fsncps/hyperfile/src/config/icon"
	"github.com/fsncps/hyperfile/src/internal/common"
	"github.com/fsncps/hyperfile/src/internal/ui/rendering"
)

// Render returns the rendered sidebar string
func (s *Model) Render(mainPanelHeight int, sidebarFocussed bool, currentFilePanelLocation string) string {
	if common.Config.SidebarWidth == 0 {
		return ""
	}
	slog.Debug("Rendering sidebar.", "cursor", s.cursor,
		"renderIndex", s.renderIndex, "dirs count", len(s.directories),
		"sidebar focused", sidebarFocussed)

	r := ui.SidebarRenderer(mainPanelHeight+2, common.Config.SidebarWidth+2, sidebarFocussed)

	if s.searchBar.Focused() || s.searchBar.Value() != "" || sidebarFocussed {
		r.AddLines(s.searchBar.View())
	}

	if s.NoActualDir() {
		r.AddLines(common.SideBarNoneText)
	} else {
		s.directoriesRender(mainPanelHeight, currentFilePanelLocation, sidebarFocussed, r)
	}
	return r.Render()
}

func (s *Model) directoriesRender(mainPanelHeight int, curFilePanelFileLocation string,
	sideBarFocussed bool, r *rendering.Renderer) {
	if s.isCursorInvalid() {
		slog.Error("Unexpected situation in sideBar Model. "+
			"Cursor is at invalid position, while there are valid directories", "cursor", s.cursor,
			"directory count", len(s.directories))
	}

	cw := r.ContentWidth()
	totalHeight := sideBarInitialHeight
	for i := s.renderIndex; i < len(s.directories); i++ {
		if totalHeight+s.directories[i].RequiredHeight() > mainPanelHeight {
			break
		}
		totalHeight += s.directories[i].RequiredHeight()

		switch s.directories[i] {
		case placesDividerDir:
			r.AddLines("", sectionHeader("Places", common.SideBarPlacesHeaderStyle, cw))
		case networkDividerDir:
			r.AddLines("", sectionHeader("Network", common.SideBarNetworkHeaderStyle, cw))
		case devicesDividerDir:
			r.AddLines("", sectionHeader("Devices", common.SideBarDevicesHeaderStyle, cw))
		default:
			cursor := " "
			if s.cursor == i && sideBarFocussed && !s.searchBar.Focused() {
				cursor = icon.Cursor
			}
			if s.renaming && s.cursor == i {
				r.AddLines(s.rename.View())
			} else {
				renderStyle := common.SidebarStyle
				if s.directories[i].Location == curFilePanelFileLocation {
					renderStyle = common.SidebarSelectedStyle
				}
				if s.directories[i].usage != "" {
					r.AddLines(deviceLine(cursor, s.directories[i].Name, s.directories[i].usage, renderStyle, cw))
				} else {
					line := common.FilePanelCursorStyle.Render(cursor+" ") + renderStyle.Render(s.directories[i].Name)
					r.AddLineWithCustomTruncate(line, rendering.TailsTruncateRight)
				}
			}
		}
	}
}

// sectionHeader renders a colored "─ Title ──────" header filling contentW chars.
func sectionHeader(title string, style lipgloss.Style, contentW int) string {
	prefix := "─ " + title + " "
	fill := strings.Repeat("─", max(0, contentW-len(prefix)))
	return style.Render(prefix + fill)
}

// deviceLine builds a line with name left-aligned and usage right-aligned within contentW.
func deviceLine(cursor, name, usage string, renderStyle lipgloss.Style, contentW int) string {
	avail := contentW - 2 // 2 chars for cursor prefix
	if avail > len(usage)+4 && len(name)+len(usage)+1 > avail {
		name = name[:avail-len(usage)-4] + "..."
	}
	pad := avail - len(name) - len(usage)
	if pad < 1 {
		pad = 1
	}
	return common.FilePanelCursorStyle.Render(cursor+" ") + renderStyle.Render(name+strings.Repeat(" ", pad)+usage)
}
