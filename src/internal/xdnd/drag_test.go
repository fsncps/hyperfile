package xdnd

import (
	"os"
	"testing"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Pure function tests ----

func TestBuildURIList(t *testing.T) {
	got := buildURIList([]string{"/tmp/foo.txt", "/home/user/bar with spaces.pdf"})
	want := "file:///tmp/foo.txt\r\nfile:///home/user/bar%20with%20spaces.pdf\r\n"
	assert.Equal(t, want, string(got))
}

func TestBuildURIList_Empty(t *testing.T) {
	assert.Equal(t, "", string(buildURIList(nil)))
}

func TestU32bytes_LittleEndian(t *testing.T) {
	assert.Equal(t, []byte{0x04, 0x03, 0x02, 0x01}, u32bytes(0x01020304))
	assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x00}, u32bytes(0))
	assert.Equal(t, []byte{0xff, 0xff, 0xff, 0xff}, u32bytes(0xffffffff))
}

// ---- X11 integration tests (require $DISPLAY) ----

// TestXdndDiagnostic connects to X11 and walks the window tree under the cursor,
// reporting XdndAware status at each level.
// Run with the mouse hovered over a drop target (browser, file manager, etc.):
//
//	go test -v -run TestXdndDiagnostic ./src/internal/xdnd/
func TestXdndDiagnostic(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}

	conn, err := xgb.NewConn()
	require.NoError(t, err, "connect to X11")
	defer conn.Close()

	screen := xproto.Setup(conn).DefaultScreen(conn)

	r, err := xproto.InternAtom(conn, false, uint16(len("XdndAware")), "XdndAware").Reply()
	require.NoError(t, err)
	awareAtom := r.Atom

	qp, err := xproto.QueryPointer(conn, screen.Root).Reply()
	require.NoError(t, err)

	t.Logf("Cursor at root (%d,%d), direct child of root: %d", qp.RootX, qp.RootY, qp.Child)

	if qp.Child == xproto.WindowNone {
		t.Log("No window under cursor — move mouse over an application first")
		return
	}

	topLevel := qp.Child
	win := topLevel
	for depth := range 12 {
		prop, err := xproto.GetProperty(conn, false, win, awareAtom, xproto.AtomAtom, 0, 1).Reply()
		aware := err == nil && prop.ValueLen > 0
		t.Logf("  depth %d  win=0x%x  XdndAware=%v", depth, win, aware)

		sub, err := xproto.QueryPointer(conn, win).Reply()
		if err != nil || sub.Child == xproto.WindowNone {
			t.Logf("  => no deeper child, stopping")
			break
		}
		win = sub.Child
	}

	// Also verify our topLevelAt logic (current code checks only depth 0)
	prop, _ := xproto.GetProperty(conn, false, topLevel, awareAtom, xproto.AtomAtom, 0, 1).Reply()
	if prop.ValueLen == 0 {
		t.Log("DIAGNOSIS: top-level (direct child of root) lacks XdndAware — reparenting WM detected.")
		t.Log("           topLevelAt() currently returns 0, so no XdndEnter is ever sent.")
		t.Log("           Fix: walk down the window tree to find XdndAware.")
	} else {
		t.Log("DIAGNOSIS: top-level window has XdndAware — topLevelAt() should work.")
	}
}

// TestXdndTargetDetection creates a test X window with XdndAware set, warps the
// pointer over it, then verifies that xdndAwareUnderCursor finds it.
func TestXdndTargetDetection(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Skip("no DISPLAY")
	}

	conn, err := xgb.NewConn()
	require.NoError(t, err)
	defer conn.Close()

	screen := xproto.Setup(conn).DefaultScreen(conn)

	// Intern XdndAware
	ar, err := xproto.InternAtom(conn, false, uint16(len("XdndAware")), "XdndAware").Reply()
	require.NoError(t, err)
	awareAtom := ar.Atom

	// Create a visible test window in the corner of the root
	wid, err := xproto.NewWindowId(conn)
	require.NoError(t, err)
	err = xproto.CreateWindowChecked(
		conn, screen.RootDepth, wid, screen.Root,
		0, 0, 200, 200, 0,
		xproto.WindowClassInputOutput,
		screen.RootVisual,
		xproto.CwOverrideRedirect|xproto.CwEventMask,
		[]uint32{1, uint32(xproto.EventMaskNoEvent)},
	).Check()
	require.NoError(t, err)
	defer xproto.DestroyWindow(conn, wid)

	// Set XdndAware on it
	ver := u32bytes(uint32(xdndVersion))
	xproto.ChangeProperty(conn, xproto.PropModeReplace, wid, awareAtom, xproto.AtomAtom, 32, 1, ver)

	xproto.MapWindow(conn, wid)
	conn.Sync() // flush

	// Warp pointer to centre of our window
	xproto.WarpPointer(conn, xproto.WindowNone, screen.Root, 0, 0, 0, 0, 100, 100)
	conn.Sync()

	// Build a minimal state to test xdndAwareUnderCursor
	s := &state{conn: conn, screen: screen, aware: awareAtom, srcWin: 99999}

	found := s.xdndAwareUnderCursor(wid, 5)
	assert.True(t, found, "xdndAwareUnderCursor should detect XdndAware on the test window")

	// topLevelAt should return wid (it's a direct child of root)
	target := s.topLevelAt()
	assert.Equal(t, wid, target, "topLevelAt should return our XdndAware window")
}
