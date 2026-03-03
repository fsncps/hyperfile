package xdnd

import (
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgb/xtest"
)

// DragonSeamless launches dragon and automatically warps the cursor to its
// window and injects a ButtonPress via the XTest extension, so the user can
// drag to the target immediately without manually clicking dragon's window.
// Falls back to a plain dragon launch if XTest is unavailable or dragon's
// window cannot be found within the timeout.
func DragonSeamless(tool string, paths []string) error {
	conn, err := xgb.NewConn()
	if err != nil {
		slog.Debug("xdnd: dragon: X11 connect failed, launching plain", "err", err)
		return launchDragonPlain(tool, paths)
	}
	defer conn.Close()

	if err := xtest.Init(conn); err != nil {
		slog.Debug("xdnd: dragon: XTest not available, launching plain", "err", err)
		return launchDragonPlain(tool, paths)
	}

	screen := xproto.Setup(conn).DefaultScreen(conn)

	// Snapshot existing top-level windows before launching dragon.
	before := make(map[xproto.Window]bool)
	if tree, err := xproto.QueryTree(conn, screen.Root).Reply(); err == nil {
		for _, w := range tree.Children {
			before[w] = true
		}
	}

	args := append([]string{"--icon-only", "--and-exit"}, paths...)
	cmd := exec.Command(tool, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("dragon: start: %w", err)
	}
	slog.Debug("xdnd: dragon launched", "pid", cmd.Process.Pid, "paths", paths)

	// Poll QueryTree until a new top-level window appears (dragon's WM frame).
	dragonWin := findNewWindow(conn, screen.Root, before, 3*time.Second)
	if dragonWin == 0 {
		slog.Debug("xdnd: dragon window not found within timeout — user must click manually")
		return nil
	}
	slog.Debug("xdnd: dragon window mapped", "win", dragonWin)

	// Give GTK a moment to finish drawing the widget before we click.
	time.Sleep(150 * time.Millisecond)

	// Translate dragon window centre to root coordinates.
	geom, err := xproto.GetGeometry(conn, xproto.Drawable(dragonWin)).Reply()
	if err != nil {
		slog.Debug("xdnd: dragon: GetGeometry failed", "err", err)
		return nil
	}
	trans, err := xproto.TranslateCoordinates(conn, dragonWin, screen.Root,
		int16(geom.Width/2), int16(geom.Height/2)).Reply()
	if err != nil {
		slog.Debug("xdnd: dragon: TranslateCoordinates failed", "err", err)
		return nil
	}
	cx, cy := trans.DstX, trans.DstY
	slog.Debug("xdnd: warping cursor to dragon window", "x", cx, "y", cy,
		"win", dragonWin, "geom", fmt.Sprintf("%dx%d+%d+%d", geom.Width, geom.Height, geom.X, geom.Y))

	// Save cursor position before warping to dragon.
	origX, origY := int16(0), int16(0)
	if qp, err := xproto.QueryPointer(conn, screen.Root).Reply(); err == nil {
		origX, origY = int16(qp.RootX), int16(qp.RootY)
	}

	// Warp the physical cursor to the centre of dragon's window.
	xproto.WarpPointer(conn, xproto.WindowNone, screen.Root, 0, 0, 0, 0, cx, cy)
	conn.Sync()

	// Brief pause so the WarpPointer takes effect before injecting the press.
	time.Sleep(50 * time.Millisecond)

	// Inject a left-button press. FakeInputChecked lets us call Check() correctly.
	if err := xtest.FakeInputChecked(conn, xproto.ButtonPress, 1, 0, screen.Root, cx, cy, 0).Check(); err != nil {
		slog.Debug("xdnd: dragon: FakeInput ButtonPress failed", "err", err)
		return nil
	}
	conn.Sync()
	slog.Debug("xdnd: dragon: ButtonPress injected", "dragonPos", fmt.Sprintf("%d,%d", cx, cy))

	// Warp back — this motion is what GTK uses to cross the drag threshold and
	// start the DnD. Without it the cursor just sits on dragon's window.
	xproto.WarpPointer(conn, xproto.WindowNone, screen.Root, 0, 0, 0, 0, origX, origY)
	conn.Sync()

	// Minimize dragon via awesome-client now that DnD is held by GTK.
	// Match by PID so we don't accidentally touch other windows.
	go func() {
		time.Sleep(100 * time.Millisecond)
		lua := fmt.Sprintf(
			`for _,c in ipairs(client.get()) do if c.pid==%d then c.minimized=true end end`,
			cmd.Process.Pid)
		slog.Debug("xdnd: dragon: minimizing via awesome-client", "pid", cmd.Process.Pid)
		if err := exec.Command("awesome-client", lua).Run(); err != nil {
			slog.Debug("xdnd: dragon: awesome-client failed", "err", err)
		}
	}()

	return nil
}

// findNewWindow polls QueryTree on root until a window appears that wasn't in
// the before snapshot, or the timeout elapses.  Works with reparenting WMs
// (e.g. awesome) where MapNotify events are never delivered to our connection.
func findNewWindow(conn *xgb.Conn, root xproto.Window, before map[xproto.Window]bool, timeout time.Duration) xproto.Window {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		tree, err := xproto.QueryTree(conn, root).Reply()
		if err != nil {
			return 0
		}
		for _, w := range tree.Children {
			if !before[w] {
				return w
			}
		}
	}
	return 0
}

func launchDragonPlain(tool string, paths []string) error {
	args := append([]string{"--icon-only", "--and-exit"}, paths...)
	cmd := exec.Command(tool, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("dragon: %w", err)
	}
	return nil
}
