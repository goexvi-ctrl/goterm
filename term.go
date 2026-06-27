package goterm

// A Term represents a terminal with a primary and an alternate screen buffer.
// Both screens share the same dimensions; Current points at the screen that
// input is currently applied to.  Each Screen holds a back-pointer to its Term
// (Screen.term) so sequence handlers can switch the active buffer.
type Term struct {
	Primary   *Screen
	Alternate *Screen
	Current   *Screen

	savedRow int // cursor saved when entering the alternate screen (?1049)
	savedCol int
}

// NewTerm returns a Term of the given size with the primary screen active.
func NewTerm(rows, cols int) *Term {
	t := &Term{
		Primary:   New(rows, cols),
		Alternate: New(rows, cols),
	}
	t.Primary.term = t
	t.Alternate.term = t
	t.Current = t.Primary
	return t
}

// enterAlternate switches to the alternate screen, clearing it.  When save is
// true the current cursor position is remembered for a later exit (?1049).
//
// All of ?1049/?47/?1047 clear the alternate buffer on entry here; this matches
// ?1049 (the mode editors use) and is a deliberate simplification for ?47/?1047.
func (t *Term) enterAlternate(save bool) {
	if t.Current == t.Alternate {
		return
	}
	if save {
		t.savedRow, t.savedCol = t.Current.Row, t.Current.Col
	}
	t.Current = t.Alternate
	t.Alternate.clear()
}

// exitAlternate switches back to the primary screen.  When restore is true the
// cursor saved by enterAlternate is restored (?1049).
func (t *Term) exitAlternate(restore bool) {
	if t.Current == t.Primary {
		return
	}
	t.Current = t.Primary
	if restore {
		t.Primary.Row, t.Primary.Col = t.savedRow, t.savedCol
	}
}
