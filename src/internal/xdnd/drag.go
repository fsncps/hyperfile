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

	if err := s.internAtoms(); err != nil {
		return err
	}
	if err := s.createWindow(); err != nil {
		return err
	}
	defer xproto.DestroyWindow(conn, s.srcWin)

	// Own XdndSelection so SelectionRequest events route to us.
	xproto.SetSelectionOwner(conn, s.srcWin, s.selection, xproto.TimeCurrentTime)

	// Non-fatal: drag works without cursor change if this fails.
	if err := s.loadFleurCursor(); err != nil {
		slog.Debug("xdnd: cursor load failed", "err", err)
	}
	if s.cursor != 0 {
		defer xproto.FreeCursor(conn, s.cursor)
	}

	if err := s.grabPointer(); err != nil {
		return fmt.Errorf("xdnd: grab pointer: %w", err)
	}
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
	for {
		ev, err := s.conn.WaitForEvent()
		if err != nil {
			return nil // connection closed or error; treat as cancel
		}
		switch e := ev.(type) {
		case xproto.MotionNotifyEvent:
			s.onMotion(e.RootX, e.RootY)

		case xproto.ButtonReleaseEvent:
			s.onRelease()
			return nil

		case xproto.SelectionRequestEvent:
			s.onSelectionRequest(e)

		case xproto.ClientMessageEvent:
			switch e.Type {
			case s.status:
				s.targetAccepts = e.Data.Data32[1]&1 != 0
			case s.finished:
				return nil
			}
		}
	}
}

func (s *state) onMotion(rootX, rootY int16) {
	newTarget := s.topLevelAt()
	if newTarget != s.target {
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
	if s.target == 0 || !s.targetAccepts {
		if s.target != 0 {
			s.sendLeave()
		}
		return
	}
	s.sendDrop()
	s.waitFinished()
}

func (s *state) waitFinished() {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ev, _ := s.conn.PollForEvent()
		if ev == nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		switch e := ev.(type) {
		case xproto.ClientMessageEvent:
			if e.Type == s.finished {
				return
			}
		case xproto.SelectionRequestEvent:
			s.onSelectionRequest(e)
		}
	}
}

// ---- Window detection ----

// topLevelAt returns the top-level (direct child of root) X11 window
// currently under the pointer, or 0 if none or no XdndAware property.
func (s *state) topLevelAt() xproto.Window {
	r, err := xproto.QueryPointer(s.conn, s.screen.Root).Reply()
	if err != nil || r.Child == xproto.WindowNone {
		return 0
	}
	win := r.Child
	if win == s.srcWin {
		return 0
	}
	// Check XdndAware
	prop, err := xproto.GetProperty(s.conn, false, win,
		s.aware, xproto.AtomAtom, 0, 1).Reply()
	if err != nil || prop.ValueLen == 0 {
		return 0
	}
	return win
}

// ---- XDND messages ----

func (s *state) sendEnter() {
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
	s.clientMsg(s.target, s.leave, [5]uint32{uint32(s.srcWin), 0, 0, 0, 0})
}

func (s *state) sendDrop() {
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
}

// ---- Helpers ----

func u32bytes(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}
