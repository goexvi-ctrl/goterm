package goterm

import (
	"sync"
	"sync/atomic"
	"time"
)

// chanBuf is the buffer size for the Term's outbound channel.  Buffering lets a
// single-goroutine test feed input that triggers a response and then read the
// response afterward without deadlocking.
const chanBuf = 256

// A Term represents a terminal with a primary and an alternate screen buffer.
// Both screens share the same dimensions; Current points at the screen that
// input is currently applied to.  Each Screen holds a back-pointer to its Term
// (Screen.term) so sequence handlers can switch the active buffer.
//
// Concurrency: Write processes input synchronously and takes mu, so direct
// Write callers see the screen settle when Write returns.  Once Start launches
// an application, a background goroutine pumps its output through Write, so the
// screen must be read with the locking accessors (Dump, WaitFor), not by
// touching Current directly.
type Term struct {
	Primary   *Screen
	Alternate *Screen
	Current   *Screen

	Bell int // count of BEL (^G) received since the last ClearBell

	pending []byte // an escape sequence truncated at the end of a Write, held for the next one

	// Out is the terminal's single return byte stream to the program, like the
	// one serial line a real terminal has.  Query responses (DSR cursor
	// position, DA) multiplex onto it along with anything else the terminal
	// emits.  It is buffered and a caller is expected to drain it.  All sends
	// go through Send so the blocking policy lives in one place.
	Out chan []byte

	mu  sync.Mutex  // guards Write and screen reads while an application runs on the PTY
	pty *ptySession // the running application, if any (see Start)
	// lastActivity is the UnixNano of the most recent Write or Send, for
	// WaitQuiet.  It is atomic, not guarded by mu, because Send updates it from
	// within Write (DSR/DA responses) while mu is held.
	lastActivity atomic.Int64
}

// touch records activity now, so WaitQuiet measures idle time from this point.
func (t *Term) touch() { t.lastActivity.Store(time.Now().UnixNano()) }

// Send writes data onto the terminal's return stream (Out).  It is the single
// choke point for Out sends so the blocking/draining policy can evolve in one
// place; for now it is a plain buffered send.
func (t *Term) Send(data []byte) {
	t.touch() // input activity resets the WaitQuiet idle clock
	t.Out <- data
}

// ClearBell resets the bell counter to zero.
func (t *Term) ClearBell() {
	t.Bell = 0
}

// New returns a Term of the given size with the primary screen active.
func New(rows, cols int) *Term {
	t := &Term{
		Primary:   NewScreen(rows, cols),
		Alternate: NewScreen(rows, cols),
		Out:       make(chan []byte, chanBuf),
	}
	t.touch() // so WaitQuiet does not read the zero time as long-idle
	t.Primary.term = t
	t.Alternate.term = t
	t.Current = t.Primary
	return t
}

// enterAlternate switches to the alternate screen, clearing it.  The primary
// screen keeps its own cursor untouched, so no separate cursor save is needed.
//
// All of ?1049/?47/?1047 clear the alternate buffer on entry here; this matches
// ?1049 (the mode editors use) and is a deliberate simplification for ?47/?1047.
func (t *Term) enterAlternate() {
	if t.Current == t.Alternate {
		return
	}
	t.Current = t.Alternate
	t.Alternate.clear()
}

// exitAlternate switches back to the primary screen.  With restore (?1049) the
// primary's preserved cursor is left in place; without it (?47/?1047) the
// cursor carries over from the alternate screen.
func (t *Term) exitAlternate(restore bool) {
	if t.Current == t.Primary {
		return
	}
	if !restore {
		t.Primary.Row, t.Primary.Col = t.Alternate.Row, t.Alternate.Col
	}
	t.Current = t.Primary
}
