package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildTreeNodesWithRgFilter(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "a"), 0o755)
	_ = os.MkdirAll(filepath.Join(root, "b"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "a", "match.go"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "a", "skip.txt"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "b", "match.go"), []byte("x"), 0o644)

	rgMatches := map[string]bool{
		filepath.Join(root, "a", "match.go"): true,
		filepath.Join(root, "b", "match.go"): true,
	}

	nodes := buildTreeNodes(root, 5, nil, nil, false, rgMatches)
	paths := make([]string, len(nodes))
	for i, n := range nodes {
		paths[i] = n.path
	}

	assert.Contains(t, paths, filepath.Join(root, "a"))
	assert.Contains(t, paths, filepath.Join(root, "a", "match.go"))
	assert.NotContains(t, paths, filepath.Join(root, "a", "skip.txt"))
	assert.Contains(t, paths, filepath.Join(root, "b"))
	assert.Contains(t, paths, filepath.Join(root, "b", "match.go"))
}

func TestBuildTreeNodesNoFilter(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "a.go"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "b.txt"), []byte("x"), 0o644)

	nodes := buildTreeNodes(root, 5, nil, nil, false, nil)
	assert.Len(t, nodes, 2)
}
