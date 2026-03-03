// Package xdnd implements the X11 XDND drag-and-drop source protocol.
// It grabs the pointer, drives the XDND state machine, and provides
// file paths via text/uri-list on SelectionRequest.
// Call Drag in a goroutine; it blocks until the drop completes or is cancelled.
package xdnd

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

const xdndVersion = 5

type state struct {
	conn   *xgb.Conn
	screen *xproto.ScreenInfo
	srcWin xproto.Window
	cursor xproto.Cursor

	uriList []byte

	// interned atoms
	aware      xproto.Atom
	typeList   xproto.Atom
	enter      xproto.Atom
	position   xproto.Atom
	status     xproto.Atom
	leave      xproto.Atom
	drop       xproto.Atom
	finished   xproto.Atom
	selection  xproto.Atom
	actionCopy xproto.Atom
	uriListT   xproto.Atom

	// drag state
	target        xproto.Window
	targetAccepts bool
}

// Drag performs an X11 XDND drag for the given file paths.
// Must be called in a goroutine. Returns nil on successful drop or cancel.
func Drag(paths []string) error {
	slog.Debug("xdnd: Drag start", "paths", paths)

	conn, err := xgb.NewConn()
	if err != nil {
		return fmt.Errorf("xdnd: connect: %w", err)
	}
	defer conn.Close()

	setup := xproto.Setup(conn)
	s := &state{
		conn:    conn,
		screen:  setup.DefaultScreen(conn),
		uriList: buildURIList(paths),
	}
	slog.Debug("xdnd: X11 connected", "root", s.screen.Root, "uri_list", string(s.uriList))

	if err := s.internAtoms(); err != nil {
		return err
	}
	slog.Debug("xdnd: atoms interned",
		"aware", s.aware, "enter", s.enter, "position", s.position,
		"status", s.status, "leave", s.leave, "drop", s.drop,
		"finished", s.finished, "selection", s.selection,
		"uriListT", s.uriListT)

	if err := s.createWindow(); err != nil {
		return err
	}
	slog.Debug("xdnd: source window created", "srcWin", s.srcWin)
	defer xproto.DestroyWindow(conn, s.srcWin)

	// Own XdndSelection so SelectionRequest events route to us.
	xproto.SetSelectionOwner(conn, s.srcWin, s.selection, xproto.TimeCurrentTime)
	// Verify ownership
	owner, err := xproto.GetSelectionOwner(conn, s.selection).Reply()
	if err != nil {
		slog.Warn("xdnd: could not verify selection ownership", "err", err)
	} else {
		slog.Debug("xdnd: XdndSelection owner", "owner", owner.Owner, "srcWin", s.srcWin, "match", owner.Owner == s.srcWin)
	}

	// Non-fatal: drag works without cursor change if this fails.
	if err := s.loadFleurCursor(); err != nil {
		slog.Debug("xdnd: cursor load failed", "err", err)
	} else {
		slog.Debug("xdnd: fleur cursor loaded", "cursor", s.cursor)
	}
	if s.cursor != 0 {
		defer xproto.FreeCursor(conn, s.cursor)
	}

	if err := s.grabPointer(); err != nil {
		return fmt.Errorf("xdnd: grab pointer: %w", err)
	}
	slog.Debug("xdnd: pointer grabbed — entering event loop")
	defer xproto.UngrabPointer(conn, 0)

	return s.loop()
}

// ---- Setup ----

func buildURIList(paths []string) []byte {
	var b strings.Builder
	for _, p := range paths {
		u := url.URL{Scheme: "file", Path: p}
		b.WriteString(u.String())
		b.WriteString("\r\n")
	}
	return []byte(b.String())
}

func (s *state) internAtoms() error {
	names := []string{
		"XdndAware", "XdndTypeList", "XdndEnter", "XdndPosition",
		"XdndStatus", "XdndLeave", "XdndDrop", "XdndFinished",
		"XdndSelection", "XdndActionCopy", "text/uri-list",
	}
	cookies := make([]xproto.InternAtomCookie, len(names))
	for i, n := range names {
		cookies[i] = xproto.InternAtom(s.conn, false, uint16(len(n)), n)
	}
	atoms := make([]xproto.Atom, len(names))
	for i, c := range cookies {
		r, err := c.Reply()
		if err != nil {
			return fmt.Errorf("xdnd: intern %s: %w", names[i], err)
		}
		atoms[i] = r.Atom
	}
	s.aware, s.typeList, s.enter, s.position = atoms[0], atoms[1], atoms[2], atoms[3]
	s.status, s.leave, s.drop, s.finished = atoms[4], atoms[5], atoms[6], atoms[7]
	s.selection, s.actionCopy, s.uriListT = atoms[8], atoms[9], atoms[10]
	return nil
}

func (s *state) createWindow() error {
	wid, err := xproto.NewWindowId(s.conn)
	if err != nil {
		return err
	}
	s.srcWin = wid

	err = xproto.CreateWindowChecked(
		s.conn, s.screen.RootDepth, wid, s.screen.Root,
		-1, -1, 1, 1, 0,
		xproto.WindowClassInputOutput,
		s.screen.RootVisual,
		xproto.CwOverrideRedirect, []uint32{1},
	).Check()
	if err != nil {
		return fmt.Errorf("xdnd: create window: %w", err)
	}

	// XdndAware = version 5
	ver := u32bytes(uint32(xdndVersion))
	xproto.ChangeProperty(s.conn, xproto.PropModeReplace, wid,
		s.aware, xproto.AtomAtom, 32, 1, ver)

	// XdndTypeList = [text/uri-list]
	xproto.ChangeProperty(s.conn, xproto.PropModeReplace, wid,
		s.typeList, xproto.AtomAtom, 32, 1, u32bytes(uint32(s.uriListT)))

	xproto.MapWindow(s.conn, wid)
	return nil
}

func (s *state) loadFleurCursor() error {
	fid, err := xproto.NewFontId(s.conn)
	if err != nil {
		return err
	}
	if err := xproto.OpenFontChecked(s.conn, fid, 6, "cursor").Check(); err != nil {
		return err
	}
	defer xproto.CloseFont(s.conn, fid)

	cid, err := xproto.NewCursorId(s.conn)
	if err != nil {
		return err
	}
	const xcFleur = 52
	err = xproto.CreateGlyphCursorChecked(
		s.conn, cid, fid, fid, xcFleur, xcFleur+1,
		0, 0, 0, 0xffff, 0xffff, 0xffff,
	).Check()
	if err != nil {
		return err
	}
	s.cursor = cid
	return nil
}

func (s *state) grabPointer() error {
	mask := uint16(xproto.EventMaskButtonRelease | xproto.EventMaskPointerMotion)
	r, err := xproto.GrabPointer(
		s.conn, false, s.screen.Root, mask,
		xproto.GrabModeAsync, xproto.GrabModeAsync,
		xproto.WindowNone, s.cursor, 0,
	).Reply()
	if err != nil {
		return err
	}
	if r.Status != 0 {
		return fmt.Errorf("grab status %d", r.Status)
	}
	return nil
}

// ---- Event loop ----

func (s *state) loop() error {
	motionCount := 0
	for {
		ev, err := s.conn.WaitForEvent()
		if err != nil {
			slog.Debug("xdnd: WaitForEvent error, ending drag", "err", err)
			return nil // connection closed or error; treat as cancel
		}
		switch e := ev.(type) {
		case xproto.MotionNotifyEvent:
			motionCount++
			if motionCount%20 == 1 { // log every 20th motion to avoid log spam
				slog.Debug("xdnd: MotionNotify", "rootX", e.RootX, "rootY", e.RootY,
					"target", s.target, "targetAccepts", s.targetAccepts, "count", motionCount)
			}
			s.onMotion(e.RootX, e.RootY)

		case xproto.ButtonReleaseEvent:
			slog.Debug("xdnd: ButtonRelease", "target", s.target, "targetAccepts", s.targetAccepts)
			s.onRelease()
			slog.Debug("xdnd: drag ended after release")
			return nil

		case xproto.SelectionRequestEvent:
			slog.Debug("xdnd: SelectionRequest",
				"requestor", e.Requestor, "selection", e.Selection,
				"target_atom", e.Target, "property", e.Property)
			s.onSelectionRequest(e)

		case xproto.ClientMessageEvent:
			slog.Debug("xdnd: ClientMessage", "type", e.Type,
				"status_atom", s.status, "finished_atom", s.finished,
				"data32[0]", e.Data.Data32[0], "data32[1]", e.Data.Data32[1])
			switch e.Type {
			case s.status:
				accepts := e.Data.Data32[1]&1 != 0
				slog.Debug("xdnd: XdndStatus received", "from", e.Data.Data32[0], "accepts", accepts)
				s.targetAccepts = accepts
			case s.finished:
				slog.Debug("xdnd: XdndFinished received")
				return nil
			default:
				slog.Debug("xdnd: unhandled ClientMessage type", "type", e.Type)
			}

		default:
			slog.Debug("xdnd: unhandled event", "type", fmt.Sprintf("%T", ev))
		}
	}
}

func (s *state) onMotion(rootX, rootY int16) {
	newTarget := s.topLevelAt()
	if newTarget != s.target {
		slog.Debug("xdnd: target changed", "old", s.target, "new", newTarget, "pos", fmt.Sprintf("%d,%d", rootX, rootY))
		if s.target != 0 {
			s.sendLeave()
		}
		s.target = newTarget
		s.targetAccepts = false
		if newTarget != 0 {
			s.sendEnter()
		}
	}
	if s.target != 0 {
		s.sendPosition(rootX, rootY)
	}
}

func (s *state) onRelease() {
	slog.Debug("xdnd: onRelease", "target", s.target, "targetAccepts", s.targetAccepts)
	if s.target == 0 || !s.targetAccepts {
		if s.target != 0 {
			slog.Debug("xdnd: sending Leave (target did not accept)")
			s.sendLeave()
		} else {
			slog.Debug("xdnd: released over no XDND-aware window")
		}
		return
	}
	slog.Debug("xdnd: sending Drop to target", "target", s.target)
	s.sendDrop()
	s.waitFinished()
}

func (s *state) waitFinished() {
	slog.Debug("xdnd: waiting for XdndFinished (2s timeout)")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ev, _ := s.conn.PollForEvent()
		if ev == nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		switch e := ev.(type) {
		case xproto.ClientMessageEvent:
			slog.Debug("xdnd: waitFinished ClientMessage", "type", e.Type, "finished_atom", s.finished)
			if e.Type == s.finished {
				slog.Debug("xdnd: got XdndFinished")
				return
			}
		case xproto.SelectionRequestEvent:
			slog.Debug("xdnd: SelectionRequest during waitFinished", "requestor", e.Requestor)
			s.onSelectionRequest(e)
		}
	}
	slog.Debug("xdnd: waitFinished timed out")
}

// ---- Window detection ----

// topLevelAt returns the top-level (direct child of root) X11 window
// currently under the pointer, or 0 if it (or none of its descendants
// under the cursor) has XdndAware set.
//
// Reparenting window managers wrap each app in a frame window that lacks
// XdndAware; the property lives on the inner application window.
// xdndAwareUnderCursor walks down the pointer-child chain to find it.
func (s *state) topLevelAt() xproto.Window {
	r, err := xproto.QueryPointer(s.conn, s.screen.Root).Reply()
	if err != nil || r.Child == xproto.WindowNone {
		return 0
	}
	topLevel := r.Child
	if topLevel == s.srcWin {
		return 0
	}
	found := s.xdndAwareUnderCursor(topLevel, 8)
	if !found {
		slog.Debug("xdnd: topLevelAt: no XdndAware in window chain", "topLevel", topLevel)
	}
	if found {
		return topLevel
	}
	return 0
}

// xdndAwareUnderCursor walks down the window tree following the pointer,
// returning true if any window in the chain has XdndAware set.
func (s *state) xdndAwareUnderCursor(win xproto.Window, maxDepth int) bool {
	for depth := range maxDepth {
		prop, err := xproto.GetProperty(s.conn, false, win,
			s.aware, xproto.AtomAtom, 0, 1).Reply()
		if err == nil && prop.ValueLen > 0 {
			slog.Debug("xdnd: XdndAware found", "win", win, "depth", depth)
			return true
		}
		qp, err := xproto.QueryPointer(s.conn, win).Reply()
		if err != nil || qp.Child == xproto.WindowNone {
			slog.Debug("xdnd: xdndAwareUnderCursor: no more children", "win", win, "depth", depth)
			return false
		}
		slog.Debug("xdnd: xdndAwareUnderCursor: descending", "from", win, "to", qp.Child, "depth", depth)
		win = qp.Child
	}
	return false
}

// ---- XDND messages ----

func (s *state) sendEnter() {
	slog.Debug("xdnd: sending XdndEnter", "target", s.target, "srcWin", s.srcWin, "uriListT", s.uriListT)
	s.clientMsg(s.target, s.enter, [5]uint32{
		uint32(s.srcWin),
		uint32(xdndVersion) << 24, // flags: type count ≤ 3
		uint32(s.uriListT),
		0, 0,
	})
}

func (s *state) sendPosition(rootX, rootY int16) {
	s.clientMsg(s.target, s.position, [5]uint32{
		uint32(s.srcWin),
		0,
		uint32(rootX)<<16 | uint32(rootY),
		0, // CurrentTime
		uint32(s.actionCopy),
	})
}

func (s *state) sendLeave() {
	slog.Debug("xdnd: sending XdndLeave", "target", s.target)
	s.clientMsg(s.target, s.leave, [5]uint32{uint32(s.srcWin), 0, 0, 0, 0})
}

func (s *state) sendDrop() {
	slog.Debug("xdnd: sending XdndDrop", "target", s.target)
	s.clientMsg(s.target, s.drop, [5]uint32{uint32(s.srcWin), 0, 0, 0, 0})
}

func (s *state) clientMsg(win xproto.Window, typ xproto.Atom, data [5]uint32) {
	ev := xproto.ClientMessageEvent{
		Format: 32,
		Window: win,
		Type:   typ,
		Data:   xproto.ClientMessageDataUnionData32New(data[:]),
	}
	xproto.SendEvent(s.conn, false, win, xproto.EventMaskNoEvent, string(ev.Bytes()))
}

// ---- Selection (provides file data to target) ----

func (s *state) onSelectionRequest(e xproto.SelectionRequestEvent) {
	slog.Debug("xdnd: serving SelectionRequest",
		"requestor", e.Requestor, "property", e.Property,
		"uri_list_len", len(s.uriList), "uri_list", string(s.uriList))
	xproto.ChangeProperty(s.conn, xproto.PropModeReplace, e.Requestor,
		e.Property, s.uriListT, 8, uint32(len(s.uriList)), s.uriList)

	notify := xproto.SelectionNotifyEvent{
		Time:      e.Time,
		Requestor: e.Requestor,
		Selection: e.Selection,
		Target:    e.Target,
		Property:  e.Property,
	}
	xproto.SendEvent(s.conn, false, e.Requestor,
		xproto.EventMaskNoEvent, string(notify.Bytes()))
	slog.Debug("xdnd: SelectionNotify sent", "requestor", e.Requestor)
}

// ---- Helpers ----

func u32bytes(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}
