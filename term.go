package goterm

// A Term represents a terminal with a primary and an alternate screen buffer.
// Both screens share the same dimensions; Current points at the screen that
// input is currently applied to.  Each Screen holds a back-pointer to its Term
// (Screen.term) so sequence handlers can switch the active buffer.
type Term struct {
	Primary   *Screen
	Alternate *Screen
	Current   *Screen

	Bell int // count of BEL (^G) received since the last ClearBell
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
	}
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
