package secondary

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yorukot/superfile/src/config/icon"
	"github.com/yorukot/superfile/src/internal/backend/git"
)

func (g *GitViewModel) Render(width, height int) string {
	g.SetDimensions(width, height)

	var b strings.Builder

	headerHeight := 2
	availableHeight := height - headerHeight
	if availableHeight < 1 {
		availableHeight = 1
	}

	b.WriteString(g.renderHeader(width))
	b.WriteString("\n")

	if g.errorMessage != "" {
		b.WriteString(g.renderError(width))
		return b.String()
	}

	if g.loading {
		b.WriteString(g.renderLoading(width))
		return b.String()
	}

	entries := g.filteredEntries()
	if len(entries) == 0 {
		b.WriteString(g.renderEmpty(width))
		return b.String()
	}

	start := g.renderIdx
	end := start + availableHeight
	if end > len(entries) {
		end = len(entries)
	}

	for i := start; i < end; i++ {
		entry := entries[i]
		line := g.renderEntry(entry, i == g.cursor, width)
		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (g *GitViewModel) renderHeader(width int) string {
	staged, modified, untracked := g.Summary()

	var parts []string
	if staged > 0 {
		parts = append(parts, fmt.Sprintf("+%d", staged))
	}
	if modified > 0 {
		parts = append(parts, fmt.Sprintf("M%d", modified))
	}
	if untracked > 0 {
		parts = append(parts, fmt.Sprintf("?%d", untracked))
	}

	summary := "clean"
	if len(parts) > 0 {
		summary = strings.Join(parts, " ")
	}

	branch := "no branch"
	if g.repoRoot != "" {
		backend := git.NewBackend()
		info, err := backend.GetRepoInfo(g.repoRoot)
		if err == nil {
			branch = info.BranchDisplay()
		}
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62"))

	filterStr := fmt.Sprintf("[%s]", g.FilterString())
	title := fmt.Sprintf("Git: %s (%s) %s", branch, summary, filterStr)

	if len(title) > width {
		title = title[:width-3] + "..."
	}

	padding := width - len(title)
	if padding < 0 {
		padding = 0
	}

	return headerStyle.Render(title + strings.Repeat(" ", padding))
}

func (g *GitViewModel) renderEntry(entry *git.FileStatus, selected bool, width int) string {
	statusCode := entry.DisplayCode()

	var statusColor lipgloss.Color
	switch {
	case entry.IsStaged:
		statusColor = lipgloss.Color("2")
	case entry.IsModified():
		statusColor = lipgloss.Color("3")
	case entry.IsUntracked():
		statusColor = lipgloss.Color("8")
	case entry.IsDeleted():
		statusColor = lipgloss.Color("1")
	case entry.IsRenamed():
		statusColor = lipgloss.Color("6")
	default:
		statusColor = lipgloss.Color("7")
	}

	statusStyle := lipgloss.NewStyle().Foreground(statusColor)

	bgColor := lipgloss.Color("0")
	if selected {
		bgColor = lipgloss.Color("236")
	}

	lineStyle := lipgloss.NewStyle().Background(bgColor)

	status := statusStyle.Render(statusCode)
	path := lipgloss.NewStyle().Background(bgColor).Render(" " + entry.Path)

	line := fmt.Sprintf("%s %s", status, path)

	if len(line) < width {
		line += strings.Repeat(" ", width-len(line))
	}

	return lineStyle.Render(line[:min(len(line), width)])
}

func (g *GitViewModel) renderError(width int) string {
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("1")).
		Padding(1, 2)

	msg := icon.Error + " " + g.errorMessage
	if len(msg) > width {
		msg = msg[:width-3] + "..."
	}

	return errorStyle.Render(msg)
}

func (g *GitViewModel) renderLoading(width int) string {
	loadingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Padding(1, 2)

	return loadingStyle.Render("⏳ Loading git status...")
}

func (g *GitViewModel) renderEmpty(width int) string {
	emptyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(1, 2)

	return emptyStyle.Render("✓ No changes found")
}
