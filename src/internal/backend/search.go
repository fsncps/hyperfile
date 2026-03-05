package backend

import (
	"bufio"
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

// ContentSearcher defines the interface for content search operations.
type ContentSearcher interface {
	Search(directory, query string) ([]string, error)
}

// RipgrepSearcher implements ContentSearcher using ripgrep (rg).
type RipgrepSearcher struct{}

// NewRipgrepSearcher creates a new ripgrep-based content searcher.
func NewRipgrepSearcher() *RipgrepSearcher {
	return &RipgrepSearcher{}
}

// Search runs ripgrep to find files containing the query string.
// Returns a list of unique file paths that contain matches.
func (r *RipgrepSearcher) Search(directory, query string) ([]string, error) {
	// Check if rg is available
	if _, err := exec.LookPath("rg"); err != nil {
		return nil, err
	}

	// Run ripgrep with:
	// -l: list file paths only
	// --: end of options
	// -i: case insensitive
	cmd := exec.Command("rg", "-l", "-i", "--", query, directory)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		// rg returns exit code 1 when no matches found
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, err
	}

	// Parse unique paths from output
	seen := make(map[string]bool)
	var paths []string
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		path := strings.TrimSpace(scanner.Text())
		if path != "" && !seen[path] {
			seen[path] = true
			paths = append(paths, path)
		}
	}

	return paths, scanner.Err()
}
