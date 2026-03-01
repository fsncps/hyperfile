package sidebar

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/fsncps/hyperfile/src/internal/utils"

	"github.com/adrg/xdg"
	variable "github.com/fsncps/hyperfile/src/config"
	"github.com/fsncps/hyperfile/src/config/icon"
)

// Return all sidebar directories
func getDirectories() []directory {
	return formDirectorySlice(
		getWellKnownDirectories(),
		getPinnedDirectories(),
		getNetworkDirectories(),
		getDeviceDirectories(),
	)
}

// formDirectorySlice assembles the 3-section sidebar directory list:
// [Places header, wellKnown..., pinned..., Network header, network..., Devices header, devices...]
func formDirectorySlice(wellKnown, pinned, network, devices []directory) []directory {
	totalCapacity := 3 + len(wellKnown) + len(pinned) + len(network) + len(devices)
	dirs := make([]directory, 0, totalCapacity)
	dirs = append(dirs, placesDividerDir)
	dirs = append(dirs, wellKnown...)
	dirs = append(dirs, pinned...)
	dirs = append(dirs, networkDividerDir)
	dirs = append(dirs, network...)
	dirs = append(dirs, devicesDividerDir)
	dirs = append(dirs, devices...)
	return dirs
}

// getWellKnownDirectories returns the standard XDG user directories shown by default.
func getWellKnownDirectories() []directory {
	wellKnownDirectories := []directory{
		{Location: xdg.Home, Name: icon.Home + icon.Space + "Home"},
		{Location: xdg.UserDirs.Download, Name: icon.Download + icon.Space + "Downloads"},
		{Location: xdg.UserDirs.Documents, Name: icon.Documents + icon.Space + "Documents"},
		{Location: xdg.UserDirs.Pictures, Name: icon.Pictures + icon.Space + "Pictures"},
	}

	return slices.DeleteFunc(wellKnownDirectories, func(d directory) bool {
		_, err := os.Stat(d.Location)
		return err != nil
	})
}

// getPinnedDirectories loads user-pinned directories from disk, marking each as pinned.
func getPinnedDirectories() []directory {
	directories := []directory{}
	var paths []string

	jsonData, err := os.ReadFile(variable.PinnedFile)
	if err != nil {
		slog.Error("Error while read superfile data", "error", err)
		return directories
	}

	// Check if the data is in the old format (plain string array)
	if err := json.Unmarshal(jsonData, &paths); err == nil {
		for _, path := range paths {
			directories = append(directories, directory{
				Location: path,
				Name:     filepath.Base(path),
				pinned:   true,
			})
		}
	} else {
		// New format: directory objects
		if err := json.Unmarshal(jsonData, &directories); err != nil {
			slog.Error("Error parsing pinned data", "error", err)
		}
		// Mark all as pinned (unexported field not serialized in JSON)
		for i := range directories {
			directories[i].pinned = true
		}
	}
	return directories
}

// fuzzySearch performs a fuzzy search over a list of directories.
func fuzzySearch(query string, dirs []directory) []directory {
	if len(dirs) == 0 {
		return []directory{}
	}

	var filteredDirs []directory

	haystack := make([]string, len(dirs))
	dirMap := make(map[string]directory, len(dirs))
	for i, dir := range dirs {
		haystack[i] = dir.Name
		dirMap[dir.Name] = dir
	}

	for _, match := range utils.FzfSearch(query, haystack) {
		if d, ok := dirMap[match.Key]; ok {
			filteredDirs = append(filteredDirs, d)
		}
	}

	return filteredDirs
}

// getFilteredDirectories returns directories matching the fuzzy search query.
func getFilteredDirectories(query string) []directory {
	return formDirectorySlice(
		fuzzySearch(query, getWellKnownDirectories()),
		fuzzySearch(query, getPinnedDirectories()),
		fuzzySearch(query, getNetworkDirectories()),
		fuzzySearch(query, getDeviceDirectories()),
	)
}

// TogglePinnedDirectory adds or removes a directory from the pinned directories list.
func TogglePinnedDirectory(dir string) error {
	dirs := getPinnedDirectories()
	unPinned := false

	for i, other := range dirs {
		if other.Location == dir {
			dirs = append(dirs[:i], dirs[i+1:]...)
			unPinned = true
			break
		}
	}

	if !unPinned {
		dirs = append(dirs, directory{
			Location: dir,
			Name:     filepath.Base(dir),
		})
	}

	updatedData, err := json.Marshal(dirs)
	if err != nil {
		return fmt.Errorf("error marshaling pinned directories: %w", err)
	}

	err = os.WriteFile(variable.PinnedFile, updatedData, 0644)
	if err != nil {
		return fmt.Errorf("error writing pinned directories file: %w", err)
	}

	return nil
}
