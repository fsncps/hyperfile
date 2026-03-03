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

	// Subscribe to root for SubstructureNotify so we get MapNotify events
	// for every direct child of root that gets mapped.
	xproto.ChangeWindowAttributes(conn, screen.Root,
		xproto.CwEventMask, []uint32{uint32(xproto.EventMaskSubstructureNotify)})

	args := append([]string{"--on-top", "--and-exit"}, paths...)
	cmd := exec.Command(tool, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("dragon: start: %w", err)
	}
	slog.Debug("xdnd: dragon launched", "pid", cmd.Process.Pid, "paths", paths)

	// Wait for dragon's window to appear and be mapped (up to 3s).
	dragonWin := waitForMappedWindow(conn, 3*time.Second)
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

	// Warp the physical cursor to the centre of dragon's window.
	xproto.WarpPointer(conn, xproto.WindowNone, screen.Root, 0, 0, 0, 0, cx, cy)
	conn.Sync()

	// Brief pause so the WarpPointer takes effect before injecting the press.
	time.Sleep(50 * time.Millisecond)

	// Inject a left-button press at that position.
	// The user can then move the cursor to the drop target and release.
	if err := xtest.FakeInput(conn, xproto.ButtonPress, 1, 0, screen.Root, cx, cy, 0).Check(); err != nil {
		slog.Debug("xdnd: dragon: FakeInput ButtonPress failed", "err", err)
		return nil
	}
	conn.Sync()
	slog.Debug("xdnd: dragon: ButtonPress injected — cursor is on dragon window, user can drag now")

	return nil
}

// waitForMappedWindow returns the first non-override-redirect window mapped
// as a direct child of root within the given timeout.
func waitForMappedWindow(conn *xgb.Conn, timeout time.Duration) xproto.Window {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ev, err := conn.PollForEvent()
		if err != nil {
			return 0
		}
		if ev == nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		if mn, ok := ev.(xproto.MapNotifyEvent); ok && !mn.OverrideRedirect {
			return mn.Window
		}
	}
	return 0
}

func launchDragonPlain(tool string, paths []string) error {
	args := append([]string{"--on-top", "--and-exit"}, paths...)
	cmd := exec.Command(tool, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("dragon: %w", err)
	}
	return nil
}
