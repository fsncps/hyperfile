package secondary

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yorukot/superfile/src/config/icon"
)

func (d *DiskUsageModel) Render(width, height int) string {
	d.SetDimensions(width, height)

	var b strings.Builder

	headerHeight := 2
	availableHeight := height - headerHeight
	if availableHeight < 1 {
		availableHeight = 1
	}

	b.WriteString(d.renderHeader(width))
	b.WriteString("\n")

	d.mu.Lock()
	errMsg := d.errorMessage
	isLoading := d.loading
	entries := d.entries
	d.mu.Unlock()

	if errMsg != "" {
		b.WriteString(d.renderError(width))
		return b.String()
	}

	if isLoading {
		b.WriteString(d.renderLoading(width))
		return b.String()
	}

	if len(entries) == 0 {
		b.WriteString(d.renderEmpty(width))
		return b.String()
	}

	start := d.renderIdx
	end := start + availableHeight
	if end > len(entries) {
		end = len(entries)
	}

	barWidth := 10
	if width < 30 {
		barWidth = 5
	}

	for i := start; i < end; i++ {
		entry := entries[i]
		line := d.renderEntry(entry, i == d.cursor, width, barWidth)
		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (d *DiskUsageModel) renderHeader(width int) string {
	totalStr := FormatSize(d.totalSize)
	title := fmt.Sprintf("Disk: %s  %s total", d.rootPath, totalStr)

	if len(title) > width-3 {
		title = title[:width-6] + "..."
	}

	sortStr := "size"
	switch d.sorting {
	case DiskSortByName:
		sortStr = "name"
	case DiskSortByCount:
		sortStr = "count"
	}

	header := fmt.Sprintf("%s [s:%s]", title, sortStr)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62"))

	padding := width - len(header)
	if padding < 0 {
		padding = 0
	}

	return headerStyle.Render(header + strings.Repeat(" ", padding))
}

func (d *DiskUsageModel) renderEntry(entry DiskEntry, selected bool, width, barWidth int) string {
	bgColor := lipgloss.Color("0")
	if selected {
		bgColor = lipgloss.Color("236")
	}

	lineStyle := lipgloss.NewStyle().Background(bgColor)

	var iconChar string
	if entry.IsDir {
		iconChar = icon.Directory + " "
	} else {
		iconChar = "  "
	}

	bar := d.sizeBar(entry.Percent, barWidth, bgColor)
	barStr := lipgloss.NewStyle().Background(bgColor).Render(bar)

	sizeStr := FormatSize(entry.Size)
	percentStr := fmt.Sprintf("%.0f%%", entry.Percent)

	nameWidth := width - barWidth - len(sizeStr) - len(percentStr) - 4
	if nameWidth < 10 {
		nameWidth = 10
	}

	name := entry.Name
	if len(name) > nameWidth {
		name = name[:nameWidth-3] + "..."
	}

	nameStyle := lipgloss.NewStyle().Background(bgColor)
	nameLine := nameStyle.Render(iconChar + name)

	line := fmt.Sprintf("%s %s %s %s", nameLine, barStr, sizeStr, percentStr)

	if len(line) < width {
		line += strings.Repeat(" ", width-len(line))
	}

	if len(line) > width {
		line = line[:width]
	}

	return lineStyle.Render(line)
}

func (d *DiskUsageModel) sizeBar(percent float64, width int, bgColor lipgloss.Color) string {
	if width < 1 {
		width = 1
	}

	filled := int(float64(width) * percent / 100)
	if filled > width {
		filled = width
	}

	barStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("36")).
		Background(bgColor)

	emptyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Background(bgColor)

	bar := barStyle.Render(strings.Repeat("█", filled))
	if filled < width {
		bar += emptyStyle.Render(strings.Repeat("░", width-filled))
	}

	return bar
}

func (d *DiskUsageModel) renderError(width int) string {
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("1")).
		Padding(1, 2)

	msg := icon.Error + " " + d.errorMessage
	if len(msg) > width {
		msg = msg[:width-3] + "..."
	}

	return errorStyle.Render(msg)
}

func (d *DiskUsageModel) renderLoading(width int) string {
	d.mu.Lock()
	progress := d.calcProgress
	d.mu.Unlock()

	loadingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Padding(1, 2)

	return loadingStyle.Render(fmt.Sprintf(icon.InOperation+" Calculating... %.0f%%", progress*100))
}

func (d *DiskUsageModel) renderEmpty(width int) string {
	emptyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(1, 2)

	return emptyStyle.Render("Empty directory")
}
