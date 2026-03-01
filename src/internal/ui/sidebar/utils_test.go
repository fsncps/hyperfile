package sidebar

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func dirSlice(count int) []directory {
	res := make([]directory, count)
	for i := range count {
		res[i] = directory{Name: "Dir" + strconv.Itoa(i), Location: "/a/" + strconv.Itoa(i)}
	}
	return res
}

func fullDirSlice(count int) []directory {
	return formDirectorySlice(dirSlice(count), dirSlice(count), nil, dirSlice(count))
}

// TODO : Use t.Run(tt.name
// TODO : Get rid of global vars, use testdata in each test, even if there is a bit of
// duplication.
// TODO : Add tt.names

func Test_noActualDir(t *testing.T) {
	testcases := []struct {
		name     string
		sidebar  Model
		expected bool
	}{
		{
			"Empty invalid sidebar should have no actual directories",
			Model{},
			true,
		},
		{
			"Empty sidebar should have no actual directories",
			Model{
				directories: fullDirSlice(0),
				renderIndex: 0,
				cursor:      0,
			},
			true,
		},
		{
			"Non-Empty Sidebar with only pinned directories",
			Model{
				directories: formDirectorySlice(nil, dirSlice(10), nil, nil),
			},
			false,
		},
		{
			"Non-Empty Sidebar with all directories",
			Model{
				directories: fullDirSlice(10),
			},
			false,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.sidebar.NoActualDir())
		})
	}
}

func Test_isCursorInvalid(t *testing.T) {
	testcases := []struct {
		name     string
		sidebar  Model
		expected bool
	}{
		{
			"Empty invalid sidebar",
			Model{},
			true,
		},
		{
			"Cursor after all directories",
			Model{
				directories: fullDirSlice(10),
				renderIndex: 0,
				cursor:      33, // len=33, index 33 is out of bounds
			},
			true,
		},
		{
			"Cursor points to section divider",
			Model{
				directories: fullDirSlice(10),
				cursor:      21, // networkDivider
			},
			true,
		},
		{
			"Non-Empty Sidebar with all directories",
			Model{
				directories: fullDirSlice(10),
				cursor:      5,
			},
			false,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.sidebar.isCursorInvalid())
		})
	}
}

func Test_resetCursor(t *testing.T) {
	data := []struct {
		name              string
		curSideBar        Model
		expectedCursorPos int
	}{
		{
			name: "Only Pinned directories",
			curSideBar: Model{
				directories: formDirectorySlice(nil, dirSlice(10), nil, nil),
			},
			expectedCursorPos: 1, // After placesDivider
		},
		{
			name: "All kind of directories",
			curSideBar: Model{
				directories: fullDirSlice(10),
			},
			expectedCursorPos: 1, // First wellKnown (index 0 is placesDivider)
		},
		{
			name: "Only Disk",
			curSideBar: Model{
				directories: formDirectorySlice(nil, nil, nil, dirSlice(10)),
			},
			expectedCursorPos: 3, // After placesDivider, netDivider, devDivider
		},
		{
			name: "Empty Sidebar",
			curSideBar: Model{
				directories: fullDirSlice(0),
			},
			expectedCursorPos: 0, // Empty sidebar, cursor should reset to 0
		},
	}

	for _, tt := range data {
		t.Run(tt.name, func(t *testing.T) {
			tt.curSideBar.resetCursor()
			assert.Equal(t, tt.expectedCursorPos, tt.curSideBar.cursor)
		})
	}
}
