package secondary

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type DiskUsageModel struct {
	rootPath     string
	entries      []DiskEntry
	cursor       int
	renderIdx    int
	sorting      DiskSortMode
	loading      bool
	totalSize    int64
	calcProgress float64
	errorMessage string
	width        int
	height       int
	calcCtx      context.Context
	calcCancel   context.CancelFunc
	mu           sync.Mutex
}

type DiskEntry struct {
	Path      string
	Name      string
	Size      int64
	FileCount int
	DirCount  int
	Percent   float64
	IsDir     bool
}

type DiskSortMode int

const (
	DiskSortBySize DiskSortMode = iota
	DiskSortByName
	DiskSortByCount
)

func NewDiskUsageModel() *DiskUsageModel {
	return &DiskUsageModel{
		sorting: DiskSortBySize,
	}
}

func (d *DiskUsageModel) LoadFromDir(path string) error {
	d.mu.Lock()
	d.rootPath = path
	d.loading = true
	d.errorMessage = ""
	d.entries = nil
	d.cursor = 0
	d.renderIdx = 0
	d.calcProgress = 0
	d.mu.Unlock()

	if d.calcCancel != nil {
		d.calcCancel()
	}

	d.calcCtx, d.calcCancel = context.WithCancel(context.Background())

	go d.calculateSizes(d.calcCtx, path)

	return nil
}

func (d *DiskUsageModel) calculateSizes(ctx context.Context, root string) {
	entries, totalSize, err := d.readDirEntries(root)
	if err != nil {
		d.mu.Lock()
		d.loading = false
		d.errorMessage = err.Error()
		d.mu.Unlock()
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	for i := range entries {
		if entries[i].IsDir {
			entries[i].Size, entries[i].FileCount, entries[i].DirCount = d.calculateRecursive(ctx, entries[i].Path)
		} else {
			entries[i].FileCount = 1
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		d.mu.Lock()
		d.calcProgress = float64(i+1) / float64(len(entries))
		d.mu.Unlock()
	}

	for i := range entries {
		entries[i].Percent = float64(entries[i].Size) / float64(totalSize) * 100
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Size > entries[j].Size
	})

	d.mu.Lock()
	d.entries = entries
	d.totalSize = totalSize
	d.loading = false
	d.sortByMode()
	d.mu.Unlock()
}

func (d *DiskUsageModel) readDirEntries(root string) ([]DiskEntry, int64, error) {
	files, err := os.ReadDir(root)
	if err != nil {
		return nil, 0, err
	}

	var entries []DiskEntry
	var totalSize int64

	for _, file := range files {
		if file.Name() == "." || file.Name() == ".." {
			continue
		}

		path := filepath.Join(root, file.Name())
		info, err := file.Info()
		if err != nil {
			continue
		}

		entry := DiskEntry{
			Path:  path,
			Name:  file.Name(),
			IsDir: file.IsDir(),
		}

		if !file.IsDir() {
			entry.Size = info.Size()
			entry.FileCount = 1
		}

		entries = append(entries, entry)
		totalSize += entry.Size
	}

	return entries, totalSize, nil
}

func (d *DiskUsageModel) calculateRecursive(ctx context.Context, path string) (size int64, files, dirs int) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0, 0, 0
	}

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return size, files, dirs
		default:
		}

		fullPath := filepath.Join(path, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if entry.IsDir() {
			subSize, subFiles, subDirs := d.calculateRecursive(ctx, fullPath)
			size += subSize
			files += subFiles
			dirs += subDirs + 1
		} else {
			size += info.Size()
			files++
		}
	}

	return size, files, dirs
}

func (d *DiskUsageModel) sortByMode() {
	switch d.sorting {
	case DiskSortBySize:
		sort.Slice(d.entries, func(i, j int) bool {
			return d.entries[i].Size > d.entries[j].Size
		})
	case DiskSortByName:
		sort.Slice(d.entries, func(i, j int) bool {
			return d.entries[i].Name < d.entries[j].Name
		})
	case DiskSortByCount:
		sort.Slice(d.entries, func(i, j int) bool {
			return d.entries[i].FileCount+d.entries[i].DirCount >
				d.entries[j].FileCount+d.entries[j].DirCount
		})
	}
}

func (d *DiskUsageModel) EntryCount() int {
	return len(d.entries)
}

func (d *DiskUsageModel) ListUp(visibleH int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.entries) == 0 {
		return
	}
	if d.cursor > 0 {
		d.cursor--
	}
	if d.renderIdx > d.cursor {
		d.renderIdx = d.cursor
	}
}

func (d *DiskUsageModel) ListDown(visibleH int) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.entries) == 0 {
		return
	}
	if d.cursor < len(d.entries)-1 {
		d.cursor++
	}
	if d.cursor >= d.renderIdx+visibleH {
		d.renderIdx = d.cursor - visibleH + 1
	}
}

func (d *DiskUsageModel) CycleSort() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.sorting = (d.sorting + 1) % 3
	d.sortByMode()
	d.cursor = 0
	d.renderIdx = 0
}

func (d *DiskUsageModel) SetDimensions(width, height int) {
	d.width = width
	d.height = height
}

func (d *DiskUsageModel) GetSelectedPath() string {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.entries) == 0 || d.cursor < 0 || d.cursor >= len(d.entries) {
		return ""
	}
	return d.entries[d.cursor].Path
}

func (d *DiskUsageModel) IsLoading() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.loading
}

func (d *DiskUsageModel) GetProgress() float64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.calcProgress
}

func (d *DiskUsageModel) Cancel() {
	d.mu.Lock()
	if d.calcCancel != nil {
		d.calcCancel()
	}
	d.mu.Unlock()
}

func FormatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case size >= TB:
		return fmt.Sprintf("%.1fTB", float64(size)/float64(TB))
	case size >= GB:
		return fmt.Sprintf("%.1fGB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.1fMB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.1fKB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%dB", size)
	}
}
