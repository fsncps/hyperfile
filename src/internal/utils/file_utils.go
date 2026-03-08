package utils

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/adrg/xdg"
	"github.com/pelletier/go-toml/v2"
)

// Utility functions related to file operations

func WriteTomlData(filePath string, data interface{}) error {
	tomlData, err := toml.Marshal(data)
	if err != nil {
		// return a wrapped error
		return fmt.Errorf("error encoding data : %w", err)
	}
	err = os.WriteFile(filePath, tomlData, 0644)
	if err != nil {
		return fmt.Errorf("error writing file : %w", err)
	}
	return nil
}

// Helper function to load and validate TOML files with field checking
// errorPrefix is appended before every error message
func LoadTomlFile(filePath string, defaultData string, target interface{}, fixFlag bool) error {
	// Initialize with default config
	_ = toml.Unmarshal([]byte(defaultData), target)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return &TomlLoadError{
			userMessage:  "config file doesn't exist",
			wrappedError: err,
		}
	}

	// Create a map to track which fields are present
	var rawData map[string]interface{}
	err = toml.Unmarshal(data, &rawData)
	if err != nil {
		return &TomlLoadError{
			userMessage:  "error decoding TOML file",
			wrappedError: err,
			isFatal:      true,
		}
	}

	// Replace default values with file values
	err = toml.Unmarshal(data, target)
	if err != nil {
		var decodeErr *toml.DecodeError
		if errors.As(err, &decodeErr) {
			row, col := decodeErr.Position()
			return &TomlLoadError{
				userMessage:  fmt.Sprintf("error in field at line %d column %d", row, col),
				wrappedError: decodeErr,
				isFatal:      true,
			}
		}
		return &TomlLoadError{
			userMessage:  "error unmarshalling data",
			wrappedError: err,
			isFatal:      true,
		}
	}

	// Check for missing fields. Explicitly set default value to false
	ignoreMissing := false
	if config, ok := target.(MissingFieldIgnorer); ok {
		ignoreMissing = config.GetIgnoreMissingFields()
	}

	// Check for missing fields
	targetType := reflect.TypeOf(target).Elem()
	missingFields := []string{}

	for i := range targetType.NumField() {
		field := targetType.Field(i)
		var fieldName string
		tag := field.Tag.Get("toml")
		if tag != "" {
			// Discard options such as ",omitempty"
			fieldName = strings.Split(tag, ",")[0]
		} else {
			fieldName = field.Name
		}
		if _, exists := rawData[fieldName]; !exists {
			missingFields = append(missingFields, fieldName)
		}
	}

	if len(missingFields) == 0 {
		return nil
	}
	if !fixFlag && ignoreMissing {
		// nil error if we dont wanna fix, and dont wanna print
		return nil
	}

	resultErr := &TomlLoadError{
		missingFields: true,
	}
	if !fixFlag {
		resultErr.userMessage = fmt.Sprintf("missing fields: %v", missingFields)
		return resultErr
	}

	// Start fixing
	return fixTomlFile(resultErr, filePath, target)
}

func fixTomlFile(resultErr *TomlLoadError, filePath string, target interface{}) error {
	resultErr.isFatal = true
	// Create a unique backup of the current config file
	backupFile, err := os.CreateTemp(filepath.Dir(filePath), filepath.Base(filePath)+".bak-")
	if err != nil {
		resultErr.UpdateMessageAndError("failed to create backup file", err)
		return resultErr
	}

	backupPath := backupFile.Name()
	needsBackupFileRemoval := true
	defer func() {
		backupFile.Close()
		// Remove backup in case of unsuccessful write
		if needsBackupFileRemoval {
			if errRem := os.Remove(backupPath); errRem != nil {
				// Modify result Error
				resultErr.AddMessageAndError("warning: failed to remove backup file, backupPath : "+backupPath, errRem)
			}
		}
	}()
	// Copy the original file to the backup
	// Open it in read write mode
	origFile, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		resultErr.UpdateMessageAndError("failed to open original file for backup", err)
		return resultErr
	}
	defer origFile.Close()

	_, err = io.Copy(backupFile, origFile)
	if err != nil {
		resultErr.UpdateMessageAndError("failed to copy original file to backup", err)
		return resultErr
	}

	tomlData, err := toml.Marshal(target)
	if err != nil {
		resultErr.UpdateMessageAndError("failed to marshal config to TOML", err)
		return resultErr
	}
	_, err = origFile.WriteAt(tomlData, 0)

	if err != nil {
		resultErr.UpdateMessageAndError("failed to write TOML data to original file", err)
		return resultErr
	}

	// Fix done
	// Inform user about backup location
	resultErr.userMessage = "config file had issues. Its fixed successfully. Original backed up to : " + backupPath
	resultErr.isFatal = false
	// Do not remove backup; user may want to restore manually
	needsBackupFileRemoval = false

	return resultErr
}

// If path is not absolute, then append to currentDir and get absolute path
// Resolve paths starting with "~"
// currentDir should be an absolute path
func ResolveAbsPath(currentDir string, path string) string {
	if !filepath.IsAbs(currentDir) {
		slog.Warn("currentDir is not absolute", "currentDir", currentDir)
	}
	if strings.HasPrefix(path, "~") {
		// We dont use variable.HomeDir here, as the util package cannot have dependency
		// on variable package
		path = strings.Replace(path, "~", xdg.Home, 1)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(currentDir, path)
	}
	return filepath.Clean(path)
}

// Get directory total size
// TODO: Uni test this
func DirSize(path string) int64 {
	var size int64
	// Its named walkErr to prevent shadowing
	walkErr := filepath.WalkDir(path, func(_ string, entry os.DirEntry, err error) error {
		if err != nil {
			slog.Error("Dir size function error", "error", err)
		}
		if !entry.IsDir() {
			info, infoErr := entry.Info()
			if infoErr == nil {
				size += info.Size()
			}
		}
		return err
	})
	if walkErr != nil {
		slog.Error("errors during WalkDir", "error", walkErr)
	}
	return size
}

// MergeTomlContent merges default TOML content with existing file content.
// User values are preserved, new fields from default are added.
// Returns the merged TOML content as bytes.
func MergeTomlContent(defaultContent []byte, existingPath string) ([]byte, error) {
	// Parse default into a map
	var defaultMap map[string]interface{}
	if err := toml.Unmarshal(defaultContent, &defaultMap); err != nil {
		return nil, fmt.Errorf("error parsing default TOML: %w", err)
	}

	// Check if existing file exists
	existingData, err := os.ReadFile(existingPath)
	if err != nil {
		// File doesn't exist, return default content
		return defaultContent, nil
	}

	// Parse existing into a map
	var existingMap map[string]interface{}
	if err := toml.Unmarshal(existingData, &existingMap); err != nil {
		// Can't parse existing, return default (backup handled by caller if needed)
		slog.Warn("Could not parse existing TOML file, using defaults", "path", existingPath, "error", err)
		return defaultContent, nil
	}

	// Merge: add missing fields from default to existing
	mergeMaps(defaultMap, existingMap)

	// Marshal merged result
	merged, err := toml.Marshal(existingMap)
	if err != nil {
		return nil, fmt.Errorf("error marshaling merged TOML: %w", err)
	}

	return merged, nil
}

// mergeMaps recursively adds missing keys from src to dst
func mergeMaps(src, dst map[string]interface{}) {
	for key, srcVal := range src {
		if _, exists := dst[key]; !exists {
			// Key doesn't exist in dst, add it
			dst[key] = srcVal
		} else {
			// Key exists, check if both are maps and recurse
			srcMap, srcOk := srcVal.(map[string]interface{})
			dstMap, dstOk := dst[key].(map[string]interface{})
			if srcOk && dstOk {
				mergeMaps(srcMap, dstMap)
			}
			// Otherwise keep dst's value (user override)
		}
	}
}
