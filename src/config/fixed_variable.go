package variable

import (
	"os"
	"path/filepath"

	"github.com/fsncps/hyperfile/src/internal/utils"
	"github.com/urfave/cli/v3"

	"github.com/adrg/xdg"
)

const (
	CurrentVersion      = "v1.3.3"
	LatestVersionURL    = "https://api.github.com/repos/fsncps/hyperfile/releases/latest"
	LatestVersionGithub = "github.com/fsncps/hyperfile/releases/latest"

	// This will not break in windows. This is a relative path for Embed FS. It uses "/" only
	EmbedConfigDir           = "src/hyperfile_config"
	EmbedConfigFile          = EmbedConfigDir + "/config.toml"
	EmbedHotkeysFile         = EmbedConfigDir + "/hotkeys.toml"
	EmbedThemeDir            = EmbedConfigDir + "/theme"
	EmbedThemeCatppuccinFile = EmbedThemeDir + "/catppuccin.toml"
)

var (
	HomeDir           = xdg.Home
	HyperFileMainDir  = filepath.Join(xdg.ConfigHome, "hyperfile")
	HyperFileCacheDir = filepath.Join(xdg.CacheHome, "hyperfile")
	HyperFileDataDir  = filepath.Join(xdg.DataHome, "hyperfile")
	HyperFileStateDir = filepath.Join(xdg.StateHome, "hyperfile")

	// MainDir files
	ThemeFolder = filepath.Join(HyperFileMainDir, "theme")

	// DataDir files
	LastCheckVersion = filepath.Join(HyperFileDataDir, "lastCheckVersion")
	ThemeFileVersion = filepath.Join(HyperFileDataDir, "themeFileVersion")
	FirstUseCheck    = filepath.Join(HyperFileDataDir, "firstUseCheck")
	PinnedFile       = filepath.Join(HyperFileDataDir, "pinned.json")
	ToggleDotFile    = filepath.Join(HyperFileDataDir, "toggleDotFile")
	ToggleFooter     = filepath.Join(HyperFileDataDir, "toggleFooter")

	// StateDir files
	LogFile     = filepath.Join(HyperFileStateDir, "hyperfile.log")
	LastDirFile = filepath.Join(HyperFileStateDir, "lastdir")

	// Trash Directories
	DarwinTrashDirectory      = filepath.Join(HomeDir, ".Trash")
	CustomTrashDirectory      = filepath.Join(xdg.DataHome, "Trash")
	CustomTrashDirectoryFiles = filepath.Join(xdg.DataHome, "Trash", "files")
	CustomTrashDirectoryInfo  = filepath.Join(xdg.DataHome, "Trash", "info")
)

// These variables are actually not fixed, they are sometimes updated dynamically
var (
	ConfigFile  = filepath.Join(HyperFileMainDir, "config.toml")
	HotkeysFile = filepath.Join(HyperFileMainDir, "hotkeys.toml")

	// ChooserFile is the path where hyperfile will write the file's path, which is to be
	// opened, before exiting
	ChooserFile = ""

	// Other state variables
	FixHotkeys    = false
	FixConfigFile = false
	LastDir       = ""
	PrintLastDir  = false
)

// Still we are preventing other packages to directly modify them via reassign linter

func SetLastDir(path string) {
	LastDir = path
}

func SetChooserFile(path string) {
	ChooserFile = path
}

func UpdateVarFromCliArgs(c *cli.Command) {
	// Setting the config file path
	configFileArg := c.String("config-file")

	// Validate the config file exists
	if configFileArg != "" {
		if _, err := os.Stat(configFileArg); err != nil {
			utils.PrintfAndExit("Error: While reading config file '%s' from argument : %v", configFileArg, err)
		}
		ConfigFile = configFileArg
	}

	hotkeyFileArg := c.String("hotkey-file")

	if hotkeyFileArg != "" {
		if _, err := os.Stat(hotkeyFileArg); err != nil {
			utils.PrintfAndExit("Error: While reading hotkey file '%s' from argument : %v", hotkeyFileArg, err)
		}
		HotkeysFile = hotkeyFileArg
	}

	// It could be non existent. We are writing to the file. If file doesn't exists, we would attempt to create it.
	SetChooserFile(c.String("chooser-file"))

	FixHotkeys = c.Bool("fix-hotkeys")
	FixConfigFile = c.Bool("fix-config-file")
	PrintLastDir = c.Bool("print-last-dir")
}
