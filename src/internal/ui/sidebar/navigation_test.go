package sidebar

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// fullDirSlice(N): [0]=placesDivider(h=2), [1..N]=wellKnown, [N+1..2N]=pinned,
// [2N+1]=netDivider(h=2), [2N+2]=devDivider(h=2), [2N+3..3N+2]=disks. len=3N+3.
//
// For N=10: places@0, well@1-10, pinned@11-20, net@21, dev@22, disks@23-32. len=33.

func Test_lastRenderIndex(t *testing.T) {
	// Setup test data
	sidebarA := Model{
		directories: fullDirSlice(10),
	}
	// [0]=places(h=2), [1]=well, [2]=net(h=2), [3]=dev(h=2), [4-8]=disks. len=9.
	sidebarB := Model{
		directories: formDirectorySlice(
			dirSlice(1), nil, nil, dirSlice(5),
		),
	}

	testCases := []struct {
		name              string
		sidebar           Model
		mainPanelHeight   int
		startIndex        int
		expectedLastIndex int
		explanation       string
	}{
		{
			name:              "Small viewport with home directories",
			sidebar:           sidebarA,
			mainPanelHeight:   10,
			startIndex:        0,
			expectedLastIndex: 7,
			explanation:       "1(initialHeight) + 2(places) + 7(1-7 well dirs) = 10",
		},
		{
			name:              "Medium viewport showing home and some pinned",
			sidebar:           sidebarA,
			mainPanelHeight:   20,
			startIndex:        0,
			expectedLastIndex: 17,
			explanation:       "1(initialHeight) + 2(places) + 10(1-10 well) + 7(11-17 pinned) = 20",
		},
		{
			name:              "Medium viewport starting from pinned dirs",
			sidebar:           sidebarA,
			mainPanelHeight:   20,
			startIndex:        11,
			expectedLastIndex: 27,
			explanation:       "1(init) + 10(11-20 pinned) + 2(net) + 2(dev) + 5(23-27 disks) = 20",
		},
		{
			name:              "Large viewport starting from pinned dirs",
			sidebar:           sidebarA,
			mainPanelHeight:   100,
			startIndex:        11,
			expectedLastIndex: 32,
			explanation:       "Last dir index is 32",
		},
		{
			name:              "Start index beyond directory count",
			sidebar:           sidebarA,
			mainPanelHeight:   100,
			startIndex:        33, // len=33, so 33 is out of bounds
			expectedLastIndex: 32,
			explanation:       "When startIndex >= len(directories), return last valid index",
		},
		{
			name:              "Asymmetric directory distribution",
			sidebar:           sidebarB,
			mainPanelHeight:   12,
			startIndex:        0,
			expectedLastIndex: 7,
			explanation:       "1(init) + 2(places) + 1(well) + 2(net) + 2(dev) + 4(disks 4-7) = 12",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.sidebar.lastRenderedIndex(tt.mainPanelHeight, tt.startIndex)
			assert.Equal(t, tt.expectedLastIndex, result,
				"lastRenderedIndex failed: %s", tt.explanation)
		})
	}
}

func Test_firstRenderIndex(t *testing.T) {
	sidebarA := Model{
		directories: fullDirSlice(10),
	}
	// [0]=places(h=2), [1]=well, [2]=net(h=2), [3]=dev(h=2), [4-8]=disks. len=9.
	sidebarB := Model{
		directories: formDirectorySlice(
			dirSlice(1), nil, nil, dirSlice(5),
		),
	}
	// [0]=places(h=2), [1-5]=pinned, [6]=net(h=2), [7]=dev(h=2), [8-12]=disks. len=13.
	sidebarC := Model{
		directories: formDirectorySlice(
			nil, dirSlice(5), nil, dirSlice(5),
		),
	}
	// [0]=places(h=2), [1]=net(h=2), [2]=dev(h=2), [3-5]=disks. len=6.
	sidebarD := Model{
		directories: formDirectorySlice(
			nil, nil, nil, dirSlice(3),
		),
	}

	// Empty sidebar with only dividers
	sidebarE := Model{
		directories: fullDirSlice(0),
	}

	testCases := []struct {
		name               string
		sidebar            Model
		mainPanelHeight    int
		endIndex           int
		expectedFirstIndex int
		explanation        string
	}{
		{
			name:               "Basic calculation from end index",
			sidebar:            sidebarA,
			mainPanelHeight:    10,
			endIndex:           10,
			expectedFirstIndex: 2,
			explanation:        "1(init) + 9(2-10 well dirs) = 10",
		},
		{
			name:               "Small panel height",
			sidebar:            sidebarA,
			mainPanelHeight:    5,
			endIndex:           15,
			expectedFirstIndex: 12,
			explanation:        "1(init) + 4(12-15 pinned) = 5",
		},
		{
			name:               "End index near beginning",
			sidebar:            sidebarA,
			mainPanelHeight:    20,
			endIndex:           3,
			expectedFirstIndex: 0,
			explanation:        "When end index is near beginning, first index should be 0",
		},
		{
			name:               "End index at network divider",
			sidebar:            sidebarA,
			mainPanelHeight:    15,
			endIndex:           21, // networkDivider
			expectedFirstIndex: 9,
			explanation:        "1(init) + 2(netDiv) + 12(9-20 pinned) = 15",
		},
		{
			name:               "Very large panel height showing all items",
			sidebar:            sidebarA,
			mainPanelHeight:    100,
			endIndex:           31, // Near last disk dir
			expectedFirstIndex: 0,
			explanation:        "Large panel should show all directories from start",
		},
		{
			name:               "Asymmetric sidebar with few directories",
			sidebar:            sidebarB,
			mainPanelHeight:    12,
			endIndex:           4, // First disk dir
			expectedFirstIndex: 0,
			explanation:        "Small sidebar fits in panel height",
		},
		{
			name:               "No wellKnown directories case",
			sidebar:            sidebarC,
			mainPanelHeight:    10,
			endIndex:           6, // networkDivider
			expectedFirstIndex: 0,
			explanation:        "1(init) + 2(places) + 5(1-5 pinned) + 2(netDiv) = 10",
		},
		{
			name:               "Only disk directories case",
			sidebar:            sidebarD,
			mainPanelHeight:    8,
			endIndex:           4, // Second disk dir
			expectedFirstIndex: 1, // 1(init) + 2(net) + 2(dev) + 2(disks 3-4) = 7 ≤ 8; places(h=2) would push to 9 > 8
			explanation:        "1(init) + 2(netDiv) + 2(devDiv) + 2(disks) = 7; adding places(h=2) = 9 > 8, stop at netDiv",
		},
		{
			name:               "Empty sidebar case",
			sidebar:            sidebarE,
			mainPanelHeight:    10,
			endIndex:           1, // netDivider
			expectedFirstIndex: 0,
			explanation:        "Empty sidebar should show all dividers",
		},
		{
			name:               "End index at the start",
			sidebar:            sidebarA,
			mainPanelHeight:    5,
			endIndex:           0,
			expectedFirstIndex: 0,
			explanation:        "When end index is at start, first index should be the same",
		},
		{
			name:               "End index out of bounds",
			sidebar:            sidebarA,
			mainPanelHeight:    20,
			endIndex:           33, // len=33, so 33 is out of bounds
			expectedFirstIndex: 34, // endIndex + 1
			explanation:        "When end index is out of bounds, should return endIndex+1",
		},
		{
			name:               "Very small panel height",
			sidebar:            sidebarA,
			mainPanelHeight:    1, // initialHeight=1, no items fit
			endIndex:           10,
			expectedFirstIndex: 11,
			explanation:        "With panel height equal to initialHeight, no items can be rendered",
		},
		{
			name:               "Panel height exactly matches one section divider",
			sidebar:            sidebarA,
			mainPanelHeight:    3, // 1(init) + 2(netDiv) = 3
			endIndex:           21, // networkDivider
			expectedFirstIndex: 21,
			explanation:        "Panel height exactly fits one divider (init + h=2)",
		},
		{
			name:               "Boundary case at consecutive section headers",
			sidebar:            sidebarA,
			mainPanelHeight:    5, // 1(init) + 2(devDiv) + 2(netDiv) = 5
			endIndex:           22, // devDivider
			expectedFirstIndex: 21,
			explanation:        "1(init) + 2(devDiv) + 2(netDiv) = 5. First index is networkDivider.",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.sidebar.firstRenderedIndex(tt.mainPanelHeight, tt.endIndex)
			assert.Equal(t, tt.expectedFirstIndex, result,
				"firstRenderedIndex failed: %s", tt.explanation)
		})
	}
}

func Test_updateRenderIndex(t *testing.T) {
	testCases := []struct {
		name                string
		sidebar             Model
		mainPanelHeight     int
		initialRenderIndex  int
		initialCursor       int
		expectedRenderIndex int
		explanation         string
	}{
		{
			name: "Case I: Cursor moved above render range",
			sidebar: Model{
				directories: fullDirSlice(10),
				renderIndex: 10,
				cursor:      5, // Cursor moved to wellKnown directory
			},
			mainPanelHeight:     15,
			expectedRenderIndex: 5,
			explanation:         "When cursor moves above render range, renderIndex should be set to cursor",
		},
		{
			name: "Case II: Cursor within render range",
			sidebar: Model{
				directories: fullDirSlice(10),
				renderIndex: 5,
				cursor:      8, // Cursor within visible range
			},
			mainPanelHeight:     15,
			expectedRenderIndex: 5, // No change expected
			explanation:         "When cursor is within render range, renderIndex should not change",
		},
		{
			name: "Case III: Cursor moved below render range",
			sidebar: Model{
				directories: fullDirSlice(10),
				renderIndex: 0,
				cursor:      20, // Cursor moved to a pinned directory outside visible range
			},
			mainPanelHeight:     10,
			expectedRenderIndex: 12, // 1(init) + 9(12-20 pinned) = 10
			explanation:         "When cursor moves below render range, renderIndex should adjust to make cursor visible",
		},
		{
			name: "Edge case: Small panel with cursor at end",
			sidebar: Model{
				directories: fullDirSlice(10),
				renderIndex: 0,
				cursor:      31, // Near last disk directory
			},
			mainPanelHeight:     5,
			expectedRenderIndex: 28, // 1(init) + 4(28-31 disks) = 5
			explanation:         "With small panel and cursor at end, should adjust renderIndex to show cursor",
		},
		{
			name: "Edge case: Large panel showing everything",
			sidebar: Model{
				directories: formDirectorySlice(dirSlice(1), nil, nil, dirSlice(5)),
				renderIndex: 2,
				cursor:      4,
			},
			mainPanelHeight:     50,
			expectedRenderIndex: 2,
			explanation:         "With large panel showing all items, renderIndex should remain unchanged",
		},
		{
			name: "Edge case: Empty sidebar",
			sidebar: Model{
				directories: fullDirSlice(0),
				renderIndex: 0,
				cursor:      1,
			},
			mainPanelHeight:     10,
			expectedRenderIndex: 0,
			explanation:         "With empty sidebar, renderIndex should remain at 0",
		},
		{
			name: "Case I and III overlap: Cursor exactly at current renderIndex",
			sidebar: Model{
				directories: fullDirSlice(10),
				renderIndex: 15,
				cursor:      15,
			},
			mainPanelHeight:     10,
			expectedRenderIndex: 15,
			explanation: "When cursor is exactly at renderIndex, " +
				"Case I takes precedence and renderIndex remains unchanged",
		},
		{
			name: "Boundary case: Cursor at edge of visible range",
			sidebar: Model{
				directories: fullDirSlice(10),
				renderIndex: 5,
				cursor:      9,
			},
			mainPanelHeight:     8,
			expectedRenderIndex: 5,
			explanation:         "When cursor is at the edge of visible range, renderIndex should not change",
		},
		{
			name: "Boundary case: Cursor just beyond visible range",
			sidebar: Model{
				directories: fullDirSlice(10),
				renderIndex: 5,
				cursor:      14, // Just beyond visible range [5-13] for h=10
			},
			mainPanelHeight:     10,
			expectedRenderIndex: 6, // firstRenderedIndex(10, 14) = 6
			explanation:         "When cursor is just beyond visible range, renderIndex should adjust",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			sidebar := tt.sidebar
			sidebar.updateRenderIndex(tt.mainPanelHeight)
			assert.Equal(t, tt.expectedRenderIndex, sidebar.renderIndex,
				"updateRenderIndex failed: %s", tt.explanation)
		})
	}
}

func Test_listUp(t *testing.T) {
	testCases := []struct {
		name                string
		sidebar             Model
		mainPanelHeight     int
		expectedCursor      int
		expectedRenderIndex int
		explanation         string
	}{
		{
			name: "Basic cursor movement from middle position",
			sidebar: Model{
				directories: fullDirSlice(10),
				renderIndex: 5,
				cursor:      5,
			},
			mainPanelHeight:     15,
			expectedCursor:      4,
			expectedRenderIndex: 4,
			explanation:         "When cursor is in the middle, it should move up one position",
		},
		{
			name: "Skip consecutive dividers when moving up",
			sidebar: Model{
				directories: fullDirSlice(10),
				renderIndex: 23,
				cursor:      23, // First disk, just after devDivider(22) and netDivider(21)
			},
			mainPanelHeight:     10,
			expectedCursor:      20, // Skips devDivider(22) and netDivider(21)
			expectedRenderIndex: 20, // Case I: cursor < renderIndex
			explanation:         "When moving up to consecutive dividers, cursor should skip all of them",
		},
		{
			name: "Wrap around from top to bottom",
			sidebar: Model{
				directories: fullDirSlice(10),
				renderIndex: 0,
				cursor:      0, // placesDivider — but we start here to test wrap
			},
			mainPanelHeight:     10,
			expectedCursor:      32, // Wraps to last disk directory (index 32)
			expectedRenderIndex: 24, // 1(init) + 9(24-32 disks) = 10
			explanation:         "When at the top (index 0), cursor should wrap to the bottom",
		},
		{
			name: "Skip multiple consecutive dividers",
			// [0]=places(h=2), [1-5]=well, [6]=net(h=2), [7]=dev(h=2), [8-12]=disks. len=13.
			sidebar: Model{
				directories: formDirectorySlice(dirSlice(5), nil, nil, dirSlice(5)),
				renderIndex: 6,
				cursor:      8, // First disk, just after devDivider(7) and netDivider(6)
			},
			mainPanelHeight:     10,
			expectedCursor:      5, // Skips devDivider(7) and netDivider(6) to land on well[4]
			expectedRenderIndex: 5, // Case I: cursor(5) < renderIndex(6)
			explanation:         "When encountering multiple consecutive dividers, cursor should skip all of them",
		},
		{
			name: "No actual directories case",
			sidebar: Model{
				directories: fullDirSlice(0),
				renderIndex: 0,
				cursor:      0,
			},
			mainPanelHeight:     10,
			expectedCursor:      0,
			expectedRenderIndex: 0,
			explanation:         "When there are no actual directories, cursor should not move",
		},
		{
			name: "Large panel showing all directories",
			// [0]=places(h=2), [1-2]=well, [3-4]=pinned, [5]=net(h=2), [6]=dev(h=2), [7-8]=disks.
			sidebar: Model{
				directories: formDirectorySlice(dirSlice(2), dirSlice(2), nil, dirSlice(2)),
				renderIndex: 0,
				cursor:      3, // pinned[0]
			},
			mainPanelHeight:     50,
			expectedCursor:      2, // well[1] — no divider between well and pinned in Places
			expectedRenderIndex: 0,
			explanation:         "With large panel showing all items, cursor should move up one position",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			sidebar := tt.sidebar
			sidebar.ListUp(tt.mainPanelHeight)
			assert.Equal(t, tt.expectedCursor, sidebar.cursor,
				"listUp cursor position: %s", tt.explanation)
			assert.Equal(t, tt.expectedRenderIndex, sidebar.renderIndex,
				"listUp render index: %s", tt.explanation)
		})
	}
}

func Test_listDown(t *testing.T) {
	testCases := []struct {
		name                string
		sidebar             Model
		mainPanelHeight     int
		expectedCursor      int
		expectedRenderIndex int
		explanation         string
	}{
		{
			name: "Basic cursor movement from middle position",
			sidebar: Model{
				directories: fullDirSlice(10),
				renderIndex: 5,
				cursor:      5,
			},
			mainPanelHeight:     15,
			expectedCursor:      6,
			expectedRenderIndex: 5,
			explanation:         "When cursor is in the middle, it should move down one position",
		},
		{
			name: "Skip consecutive dividers when moving down",
			sidebar: Model{
				directories: fullDirSlice(10),
				renderIndex: 18,
				cursor:      20, // Last pinned, before netDivider(21) and devDivider(22)
			},
			mainPanelHeight:     10,
			expectedCursor:      23, // Skips netDivider(21) and devDivider(22) to land on disks[0]
			expectedRenderIndex: 18, // Still visible (23 ≤ lastRenderedIndex(10,18)=24)
			explanation:         "When moving down to consecutive dividers, cursor should skip all of them",
		},
		{
			name: "Wrap around from bottom to top",
			sidebar: Model{
				directories: fullDirSlice(10),
				renderIndex: 26,
				cursor:      32, // Last disk directory
			},
			mainPanelHeight:     10,
			expectedCursor:      1, // Wraps to 0 (placesDivider), skips to 1 (first wellKnown)
			expectedRenderIndex: 0,
			explanation:         "When at the bottom, cursor should wrap to top and skip placesDivider",
		},
		{
			name: "Skip multiple consecutive dividers",
			// [0]=places(h=2), [1-5]=well, [6]=net(h=2), [7]=dev(h=2), [8-12]=disks. len=13.
			sidebar: Model{
				directories: formDirectorySlice(dirSlice(5), nil, nil, dirSlice(5)),
				renderIndex: 0,
				cursor:      5, // Last wellKnown, before netDivider(6) and devDivider(7)
			},
			mainPanelHeight:     10,
			expectedCursor:      8, // Skips netDivider(6) and devDivider(7) to land on disks[0]
			expectedRenderIndex: 2, // firstRenderedIndex(10, 8)
			explanation:         "When encountering consecutive dividers, cursor should skip all of them",
		},
		{
			name: "No actual directories case",
			sidebar: Model{
				directories: fullDirSlice(0),
				renderIndex: 0,
				cursor:      0,
			},
			mainPanelHeight:     10,
			expectedCursor:      0,
			expectedRenderIndex: 0,
			explanation:         "When there are no actual directories, cursor should not move",
		},
		{
			name: "Move down from Places to Devices section",
			sidebar: Model{
				directories: fullDirSlice(10),
				renderIndex: 12,
				cursor:      20, // Last pinned before networkDivider(21) and devicesDivider(22)
			},
			mainPanelHeight:     10,
			expectedCursor:      23, // First disk (skips net@21 and dev@22)
			expectedRenderIndex: 17, // firstRenderedIndex(10, 23)
			explanation:         "Moving down from last pinned should skip both section headers",
		},
		{
			name: "Large panel showing all directories",
			// [0]=places(h=2), [1-2]=well, [3-4]=pinned, [5]=net(h=2), [6]=dev(h=2), [7-8]=disks.
			sidebar: Model{
				directories: formDirectorySlice(dirSlice(2), dirSlice(2), nil, dirSlice(2)),
				renderIndex: 0,
				cursor:      3,
			},
			mainPanelHeight:     50,
			expectedCursor:      4,
			expectedRenderIndex: 0,
			explanation:         "With large panel showing all items, cursor should move down and renderIndex remain unchanged",
		},
		{
			name: "Cursor at the end of visible range",
			sidebar: Model{
				directories: fullDirSlice(10),
				renderIndex: 5,
				cursor:      18, // Last visible item for h=15 with renderIndex=5
			},
			mainPanelHeight:     15,
			expectedCursor:      19,
			expectedRenderIndex: 6, // firstRenderedIndex(15, 19) = 6
			explanation:         "When cursor is at the end of visible range, moving down should adjust renderIndex",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			sidebar := tt.sidebar
			sidebar.ListDown(tt.mainPanelHeight)
			assert.Equal(t, tt.expectedCursor, sidebar.cursor,
				"listDown cursor position: %s", tt.explanation)
			assert.Equal(t, tt.expectedRenderIndex, sidebar.renderIndex,
				"listDown render index: %s", tt.explanation)
		})
	}
}
