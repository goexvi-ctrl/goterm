package goterm

import (
	"testing"

	"github.com/pborman/ansi"
)

func TestNewTerm(t *testing.T) {
	tm := New(10, 20)
	if tm.Primary == nil || tm.Alternate == nil {
		t.Fatal("Primary/Alternate must be non-nil")
	}
	if tm.Current != tm.Primary {
		t.Error("Current should start as Primary")
	}
	if tm.Primary == tm.Alternate {
		t.Error("Primary and Alternate must be distinct screens")
	}
	if tm.Primary.term != tm || tm.Alternate.term != tm {
		t.Error("both screens must back-point to the Term")
	}
	for _, s := range []*Screen{tm.Primary, tm.Alternate} {
		if s.Rows != 10 || s.Cols != 20 {
			t.Errorf("screen dims = %dx%d, want 10x20", s.Rows, s.Cols)
		}
	}
}

func TestAltScreen1049(t *testing.T) {
	tm := New(5, 10)
	// Put a marker on the primary screen and move its cursor.
	tm.Primary.Lines[0][0].Value = 'P'
	tm.Primary.Row, tm.Primary.Col = 3, 4

	// Enter the alternate screen.
	funcMap[ansi.SM](tm.Current, Params{"?1049"})
	if tm.Current != tm.Alternate {
		t.Fatal("SM ?1049 should make the alternate screen current")
	}
	// The primary keeps its own cursor untouched (it is the "saved" cursor).
	if tm.Primary.Row != 3 || tm.Primary.Col != 4 {
		t.Errorf("primary cursor = %d,%d, want 3,4", tm.Primary.Row, tm.Primary.Col)
	}
	if tm.Current.Row != 0 || tm.Current.Col != 0 {
		t.Errorf("alt cursor = %d,%d, want home", tm.Current.Row, tm.Current.Col)
	}
	if countBlanks(tm.Alternate) != 50 {
		t.Errorf("alternate should be cleared on entry, blanks = %d, want 50", countBlanks(tm.Alternate))
	}

	// Scribble on the alternate screen and move its cursor, then leave.
	tm.Current.Lines[1][1].Value = 'A'
	tm.Current.Row, tm.Current.Col = 2, 2
	funcMap[ansi.RM](tm.Current, Params{"?1049"})
	if tm.Current != tm.Primary {
		t.Fatal("RM ?1049 should restore the primary screen")
	}
	// With restore, the primary's preserved cursor stays put (not the alt's).
	if tm.Current.Row != 3 || tm.Current.Col != 4 {
		t.Errorf("restored cursor = %d,%d, want 3,4", tm.Current.Row, tm.Current.Col)
	}
	if tm.Primary.Lines[0][0].Value != 'P' {
		t.Error("primary contents must survive the alternate-screen round trip")
	}
}

func TestAltScreen47CursorCarriesOver(t *testing.T) {
	tm := New(5, 10)
	tm.Primary.Row, tm.Primary.Col = 3, 4

	funcMap[ansi.SM](tm.Current, Params{"?47"})
	if tm.Current != tm.Alternate {
		t.Fatal("SM ?47 should switch to the alternate screen")
	}

	// Move the cursor on the alternate screen, then leave without restore.
	tm.Current.Row, tm.Current.Col = 1, 2
	funcMap[ansi.RM](tm.Current, Params{"?47"})
	if tm.Current != tm.Primary {
		t.Fatal("RM ?47 should switch back to the primary screen")
	}
	// Without restore the cursor carries over from the alternate screen.
	if tm.Current.Row != 1 || tm.Current.Col != 2 {
		t.Errorf("carried-over cursor = %d,%d, want 1,2", tm.Current.Row, tm.Current.Col)
	}
}

func TestAltScreenIdempotent(t *testing.T) {
	tm := New(5, 10)
	funcMap[ansi.SM](tm.Current, Params{"?1049"})
	tm.Current.Lines[0][0].Value = 'A'
	// A second enter must not clear what we just wrote.
	funcMap[ansi.SM](tm.Current, Params{"?1049"})
	if tm.Current.Lines[0][0].Value != 'A' {
		t.Error("re-entering the alternate screen should be a no-op")
	}
	// Exiting twice should stay on primary.
	funcMap[ansi.RM](tm.Current, Params{"?1049"})
	funcMap[ansi.RM](tm.Current, Params{"?1049"})
	if tm.Current != tm.Primary {
		t.Error("redundant exit should stay on the primary screen")
	}
}

func TestBell(t *testing.T) {
	tm := New(5, 10)
	if tm.Bell != 0 {
		t.Errorf("initial Bell = %d, want 0", tm.Bell)
	}
	funcMap[ansi.BEL](tm.Current, nil)
	funcMap[ansi.BEL](tm.Current, nil)
	if tm.Bell != 2 {
		t.Errorf("Bell = %d, want 2 after two BELs", tm.Bell)
	}
	tm.ClearBell()
	if tm.Bell != 0 {
		t.Errorf("Bell = %d after ClearBell, want 0", tm.Bell)
	}
}

func TestBellStandaloneNoop(t *testing.T) {
	// A standalone screen has no Term to count on; BEL must not panic.
	s := NewScreen(5, 10)
	funcMap[ansi.BEL](s, nil)
}

func TestAltScreenStandaloneNoop(t *testing.T) {
	// A standalone screen has no Term; the alt-screen modes must be ignored
	// without panicking.
	s := NewScreen(5, 10)
	funcMap[ansi.SM](s, Params{"?1049"})
	funcMap[ansi.RM](s, Params{"?1049"})
}
