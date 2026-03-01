//go:build !windows

package internal

import (
	"path/filepath"
	"syscall"
	"testing"

	"github.com/fsncps/hyperfile/src/internal/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilePreviewWithInvalidMode(t *testing.T) {
	curTestDir := t.TempDir()
	file := filepath.Join(curTestDir, "testf")

	err := syscall.Mkfifo(file, 0644)
	require.NoError(t, err)

	m := defaultTestModel(curTestDir)

	res := stripPreviewBorder(m.filePreviewPanelRenderWithDimensions(10+2, 100+2))
	assert.Contains(t, res, common.FilePreviewUnsupportedFileMode)
}
