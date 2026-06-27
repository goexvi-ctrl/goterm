package goterm

import (
	"testing"

	"github.com/pborman/ansi"
)

func TestColorConstants(t *testing.T) {
	if Black != 0 {
		t.Errorf("Black = %d, want 0", Black)
	}
	if Red != 1 {
		t.Errorf("Red = %d, want 1", Red)
	}
	if Green != 2 {
		t.Errorf("Green = %d, want 2", Green)
	}
	if Yellow != 3 {
		t.Errorf("Yellow = %d, want 3", Yellow)
	}
	if Blue != 4 {
		t.Errorf("Blue = %d, want 4", Blue)
	}
	if Magenta != 5 {
		t.Errorf("Magenta = %d, want 5", Magenta)
	}
	if Cyan != 6 {
		t.Errorf("Cyan = %d, want 6", Cyan)
	}
	if White != 7 {
		t.Errorf("White = %d, want 7", White)
	}
}

func TestOffsetConstants(t *testing.T) {
	if Foreground != 30 {
		t.Errorf("Foreground = %d, want 30", Foreground)
	}
	if Background != 40 {
		t.Errorf("Background = %d, want 40", Background)
	}
	if Bright != 60 {
		t.Errorf("Bright = %d, want 60", Bright)
	}
}

func TestAttrConstants(t *testing.T) {
	tests := []struct {
		name string
		got  uint
		want uint
	}{
		{"NormalAttr", NormalAttr, 0},
		{"BoldAttr", BoldAttr, 1},
		{"FaintAttr", FaintAttr, 2},
		{"ItalicAttr", ItalicAttr, 3},
		{"UnderlineAttr", UnderlineAttr, 4},
		{"SlowBlinkAttr", SlowBlinkAttr, 5},
		{"RapidBlinkAttr", RapidBlinkAttr, 6},
		{"InverseAttr", InverseAttr, 7},
		{"HiddenAttr", HiddenAttr, 8},
		{"StrikethroughAttr", StrikethroughAttr, 9},
		{"BrightAttr", BrightAttr, 10},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}

func TestMaskConstants(t *testing.T) {
	tests := []struct {
		name string
		mask uint
		attr uint
	}{
		{"BoldMask", BoldMask, BoldAttr},
		{"FaintMask", FaintMask, FaintAttr},
		{"ItalicMask", ItalicMask, ItalicAttr},
		{"UnderlineMask", UnderlineMask, UnderlineAttr},
		{"SlowBlinkMask", SlowBlinkMask, SlowBlinkAttr},
		{"RapidBlinkMask", RapidBlinkMask, RapidBlinkAttr},
		{"InverseMask", InverseMask, InverseAttr},
		{"HiddenMask", HiddenMask, HiddenAttr},
		{"StrikethroughMask", StrikethroughMask, StrikethroughAttr},
		{"BrightMask", BrightMask, BrightAttr},
	}
	for _, tt := range tests {
		want := uint(1) << (tt.attr - 1)
		if tt.mask != want {
			t.Errorf("%s = %d, want %d", tt.name, tt.mask, want)
		}
	}
}

func TestNew(t *testing.T) {
	s := New(24, 80)
	if s == nil {
		t.Fatal("New returned nil")
	}
	if s.Rows != 24 {
		t.Errorf("Rows = %d, want 24", s.Rows)
	}
	if s.Cols != 80 {
		t.Errorf("Cols = %d, want 80", s.Cols)
	}
	if s.Row != 0 {
		t.Errorf("Row = %d, want 0", s.Row)
	}
	if s.Col != 0 {
		t.Errorf("Col = %d, want 0", s.Col)
	}
	if len(s.Lines) != 24 {
		t.Fatalf("len(Lines) = %d, want 24", len(s.Lines))
	}
	for i, line := range s.Lines {
		if len(line) != 80 {
			t.Errorf("len(Lines[%d]) = %d, want 80", i, len(line))
		}
		for j, cell := range line {
			if cell.Value != ' ' {
				t.Errorf("Lines[%d][%d].Value = %q, want ' '", i, j, cell.Value)
			}
			if cell.Foreground != Black {
				t.Errorf("Lines[%d][%d].Foreground = %d, want Black", i, j, cell.Foreground)
			}
			if cell.Background != White {
				t.Errorf("Lines[%d][%d].Background = %d, want White", i, j, cell.Background)
			}
		}
	}
}

func TestNewPenAndTabs(t *testing.T) {
	s := New(5, 24)
	if s.Cur != defaultCell() {
		t.Errorf("Cur = %+v, want default cell %+v", s.Cur, defaultCell())
	}
	if len(s.Tabs) != 24 {
		t.Fatalf("len(Tabs) = %d, want 24", len(s.Tabs))
	}
	for c, stop := range s.Tabs {
		want := c%8 == 0
		if stop != want {
			t.Errorf("Tabs[%d] = %v, want %v", c, stop, want)
		}
	}
}

func TestBlankUsesPen(t *testing.T) {
	s := New(5, 5)
	s.Cur.Foreground = Red
	s.Cur.Background = Blue
	s.Cur.Attributes = int(BoldMask)
	b := s.blank()
	if b.Value != ' ' {
		t.Errorf("blank Value = %q, want ' '", b.Value)
	}
	if b.Foreground != Red || b.Background != Blue {
		t.Errorf("blank colors = %d/%d, want %d/%d", b.Foreground, b.Background, Red, Blue)
	}
	if b.Attributes != int(BoldMask) {
		t.Errorf("blank Attributes = %d, want %d", b.Attributes, int(BoldMask))
	}
}

func TestNewSmall(t *testing.T) {
	s := New(1, 1)
	if s.Rows != 1 || s.Cols != 1 {
		t.Errorf("New(1,1): Rows=%d Cols=%d, want 1 1", s.Rows, s.Cols)
	}
	if len(s.Lines) != 1 || len(s.Lines[0]) != 1 {
		t.Errorf("New(1,1): Lines shape wrong")
	}
}

func TestAddAttrNormal(t *testing.T) {
	if got := addAttr(0xFF, NormalAttr); got != 0 {
		t.Errorf("addAttr(0xFF, NormalAttr) = %d, want 0", got)
	}
}

func TestAddAttrSetsbit(t *testing.T) {
	attrs := []uint{BoldAttr, FaintAttr, ItalicAttr, UnderlineAttr,
		SlowBlinkAttr, RapidBlinkAttr, InverseAttr, HiddenAttr,
		StrikethroughAttr, BrightAttr}
	for _, attr := range attrs {
		got := addAttr(0, attr)
		want := uint(1) << (attr - 1)
		if got != want {
			t.Errorf("addAttr(0, %d) = %b, want %b", attr, got, want)
		}
	}
}

func TestAddAttrAccumulates(t *testing.T) {
	bits := addAttr(0, BoldAttr)
	bits = addAttr(bits, ItalicAttr)
	if bits&(1<<(BoldAttr-1)) == 0 {
		t.Error("Bold bit lost after adding Italic")
	}
	if bits&(1<<(ItalicAttr-1)) == 0 {
		t.Error("Italic bit not set")
	}
}

func TestAddAttrIdempotent(t *testing.T) {
	once := addAttr(0, BoldAttr)
	twice := addAttr(once, BoldAttr)
	if once != twice {
		t.Errorf("addAttr idempotent: %b != %b", once, twice)
	}
}

func TestParamsInt(t *testing.T) {
	p := Params{"3", "42", "bad", ""}
	tests := []struct {
		n    int
		want int
	}{
		{0, 3},
		{1, 42},
		{2, 0}, // non-integer returns 0
		{3, 0}, // empty string returns 0
		{-1, 0},
		{4, 0}, // out of range
	}
	for _, tt := range tests {
		if got := p.Int(tt.n); got != tt.want {
			t.Errorf("Params.Int(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

func TestParamsStr(t *testing.T) {
	p := Params{"hello", "world"}
	tests := []struct {
		n    int
		want string
	}{
		{0, "hello"},
		{1, "world"},
		{-1, ""},
		{2, ""},
	}
	for _, tt := range tests {
		if got := p.Str(tt.n); got != tt.want {
			t.Errorf("Params.Str(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestParamsAmt(t *testing.T) {
	p := Params{"3", "0", "bad", ""}
	tests := []struct {
		n    int
		want int
	}{
		{0, 3},  // explicit value
		{1, 1},  // zero defaults to 1
		{2, 1},  // non-integer defaults to 1
		{3, 1},  // empty defaults to 1
		{-1, 1}, // out of range defaults to 1
		{4, 1},  // out of range defaults to 1
	}
	for _, tt := range tests {
		if got := p.Amt(tt.n); got != tt.want {
			t.Errorf("Params.Amt(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

func TestFuncMap(t *testing.T) {
	tests := []struct {
		name              string
		fn                ansi.Name
		params            Params
		startRow, startCol int
		wantRow, wantCol  int
	}{
		// Backspace moves left, stops at column 0.
		{"BS", ansi.BS, nil, 5, 5, 5, 4},
		{"BS at left edge", ansi.BS, nil, 5, 0, 5, 0},

		// Cursor up decreases the row; default amount is 1.
		{"CUU default", ansi.CUU, nil, 5, 3, 4, 3},
		{"CUU n", ansi.CUU, Params{"3"}, 5, 3, 2, 3},
		{"CUU clamps to top", ansi.CUU, Params{"99"}, 5, 3, 0, 3},

		// Cursor down increases the row; default amount is 1.
		{"CUD default", ansi.CUD, nil, 5, 3, 6, 3},
		{"CUD n", ansi.CUD, Params{"3"}, 5, 3, 8, 3},
		// Row clamps against Rows (10), not Cols (20).
		{"CUD clamps to bottom", ansi.CUD, Params{"99"}, 5, 3, 9, 3},

		// Cursor forward/back move the column.
		{"CUF default", ansi.CUF, nil, 5, 3, 5, 4},
		{"CUF n", ansi.CUF, Params{"4"}, 5, 3, 5, 7},
		{"CUF clamps to right", ansi.CUF, Params{"99"}, 5, 3, 5, 19},
		{"CUB default", ansi.CUB, nil, 5, 3, 5, 2},
		{"CUB clamps to left", ansi.CUB, Params{"99"}, 5, 3, 5, 0},

		// Cursor horizontal absolute is 1-based: column 1 -> index 0.
		{"CHA default", ansi.CHA, nil, 5, 7, 5, 0},
		{"CHA col 1", ansi.CHA, Params{"1"}, 5, 7, 5, 0},
		{"CHA col 5", ansi.CHA, Params{"5"}, 5, 7, 5, 4},
		{"CHA clamps", ansi.CHA, Params{"99"}, 5, 7, 5, 19},

		// Next/previous line move the row and reset the column.
		{"CNL default", ansi.CNL, nil, 5, 7, 6, 0},
		{"CNL n", ansi.CNL, Params{"2"}, 5, 7, 7, 0},
		{"CNL clamps", ansi.CNL, Params{"99"}, 5, 7, 9, 0},
		{"CPL default", ansi.CPL, nil, 5, 7, 4, 0},
		{"CPL clamps", ansi.CPL, Params{"99"}, 5, 7, 0, 0},

		// Carriage return resets the column only.
		{"CR", ansi.CR, nil, 5, 7, 5, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := funcMap[tt.fn]
			if !ok {
				t.Fatalf("funcMap has no entry for %v", tt.fn)
			}
			s := New(10, 20)
			s.Row, s.Col = tt.startRow, tt.startCol
			fn(s, tt.params)
			if s.Row != tt.wantRow || s.Col != tt.wantCol {
				t.Errorf("%s: got Row=%d Col=%d, want Row=%d Col=%d",
					tt.name, s.Row, s.Col, tt.wantRow, tt.wantCol)
			}
		})
	}
}

func TestCursorPositioning(t *testing.T) {
	tests := []struct {
		name              string
		fn                ansi.Name
		params            Params
		wantRow, wantCol  int
	}{
		// CUP/HVP are 1-based row;col, default 1;1 -> (0,0).
		{"CUP home (no params)", ansi.CUP, nil, 0, 0},
		{"CUP 1;1", ansi.CUP, Params{"1", "1"}, 0, 0},
		{"CUP 5;10", ansi.CUP, Params{"5", "10"}, 4, 9},
		{"CUP row only", ansi.CUP, Params{"5"}, 4, 0},
		{"CUP clamps", ansi.CUP, Params{"99", "99"}, 9, 19},
		{"HVP 3;4", ansi.HVP, Params{"3", "4"}, 2, 3},
		{"HVP home", ansi.HVP, nil, 0, 0},
		// VPA sets the row only, HPA the column only.
		{"VPA 7", ansi.VPA, Params{"7"}, 6, 5},
		{"VPA default", ansi.VPA, nil, 0, 5},
		{"VPA clamps", ansi.VPA, Params{"99"}, 9, 5},
		{"HPA 7", ansi.HPA, Params{"7"}, 5, 6},
		{"HPA default", ansi.HPA, nil, 5, 0},
		{"HPA clamps", ansi.HPA, Params{"99"}, 5, 19},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(10, 20)
			s.Row, s.Col = 5, 5 // known starting point
			funcMap[tt.fn](s, tt.params)
			if s.Row != tt.wantRow || s.Col != tt.wantCol {
				t.Errorf("%s: got Row=%d Col=%d, want Row=%d Col=%d",
					tt.name, s.Row, s.Col, tt.wantRow, tt.wantCol)
			}
		})
	}
}

func TestCursorRelative(t *testing.T) {
	tests := []struct {
		name             string
		fn               ansi.Name
		params           Params
		wantRow, wantCol int
	}{
		{"HPR default", ansi.HPR, nil, 5, 6},
		{"HPR 3", ansi.HPR, Params{"3"}, 5, 8},
		{"HPR clamps", ansi.HPR, Params{"99"}, 5, 19},
		{"HPB default", ansi.HPB, nil, 5, 4},
		{"HPB clamps", ansi.HPB, Params{"99"}, 5, 0},
		{"VPR default", ansi.VPR, nil, 6, 5},
		{"VPR clamps", ansi.VPR, Params{"99"}, 9, 5},
		{"VPB default", ansi.VPB, nil, 4, 5},
		{"VPB clamps", ansi.VPB, Params{"99"}, 0, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(10, 20)
			s.Row, s.Col = 5, 5
			funcMap[tt.fn](s, tt.params)
			if s.Row != tt.wantRow || s.Col != tt.wantCol {
				t.Errorf("%s: got Row=%d Col=%d, want Row=%d Col=%d",
					tt.name, s.Row, s.Col, tt.wantRow, tt.wantCol)
			}
		})
	}
}

// markScreen sets every cell's Value to a non-blank marker so erase/scroll
// operations can be observed by which cells become blank (space) again.
func markScreen(s *Screen, r rune) {
	for _, line := range s.Lines {
		for c := range line {
			line[c].Value = r
		}
	}
}

// rowPattern renders a row as '.' for blank cells and 'X' for non-blank ones.
func rowPattern(line Line) string {
	b := make([]byte, len(line))
	for i, c := range line {
		if c.Value == ' ' {
			b[i] = '.'
		} else {
			b[i] = 'X'
		}
	}
	return string(b)
}

func countBlanks(s *Screen) int {
	n := 0
	for _, line := range s.Lines {
		for _, c := range line {
			if c.Value == ' ' {
				n++
			}
		}
	}
	return n
}

func TestEraseLine(t *testing.T) {
	tests := []struct {
		name    string
		params  Params
		pattern string
	}{
		{"EL default (cursor to end)", nil, "XXXXX..............."},
		{"EL 0", Params{"0"}, "XXXXX..............."},
		{"EL 1 (start to cursor)", Params{"1"}, "......XXXXXXXXXXXXXX"},
		{"EL 2 (whole line)", Params{"2"}, "...................."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(10, 20)
			markScreen(s, 'X')
			s.Row, s.Col = 2, 5
			funcMap[ansi.EL](s, tt.params)
			if got := rowPattern(s.Lines[2]); got != tt.pattern {
				t.Errorf("row 2 = %q, want %q", got, tt.pattern)
			}
			// Other rows must be untouched.
			if got := rowPattern(s.Lines[0]); got != "XXXXXXXXXXXXXXXXXXXX" {
				t.Errorf("row 0 modified: %q", got)
			}
		})
	}
}

func TestEraseChar(t *testing.T) {
	tests := []struct {
		name    string
		params  Params
		pattern string
	}{
		{"ECH default (1)", nil, "XXXXX.XXXXXXXXXXXXXX"},
		{"ECH 3", Params{"3"}, "XXXXX...XXXXXXXXXXXX"},
		{"ECH clamps past edge", Params{"99"}, "XXXXX..............."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(10, 20)
			markScreen(s, 'X')
			s.Row, s.Col = 2, 5
			funcMap[ansi.ECH](s, tt.params)
			if got := rowPattern(s.Lines[2]); got != tt.pattern {
				t.Errorf("row 2 = %q, want %q", got, tt.pattern)
			}
			if s.Row != 2 || s.Col != 5 {
				t.Errorf("cursor moved to %d,%d", s.Row, s.Col)
			}
		})
	}
}

func TestEraseDisplay(t *testing.T) {
	// Cursor at row 2, col 5 on a 10x20 screen.
	t.Run("mode 0 (cursor to end)", func(t *testing.T) {
		s := New(10, 20)
		markScreen(s, 'X')
		s.Row, s.Col = 2, 5
		funcMap[ansi.ED](s, nil)
		if got := countBlanks(s); got != 155 {
			t.Errorf("blanks = %d, want 155", got)
		}
		if got := rowPattern(s.Lines[2]); got != "XXXXX..............." {
			t.Errorf("row 2 = %q", got)
		}
		if got := rowPattern(s.Lines[1]); got != "XXXXXXXXXXXXXXXXXXXX" {
			t.Errorf("row 1 should be untouched: %q", got)
		}
		if got := rowPattern(s.Lines[3]); got != "...................." {
			t.Errorf("row 3 should be cleared: %q", got)
		}
	})
	t.Run("mode 1 (start to cursor)", func(t *testing.T) {
		s := New(10, 20)
		markScreen(s, 'X')
		s.Row, s.Col = 2, 5
		funcMap[ansi.ED](s, Params{"1"})
		if got := countBlanks(s); got != 46 {
			t.Errorf("blanks = %d, want 46", got)
		}
		if got := rowPattern(s.Lines[2]); got != "......XXXXXXXXXXXXXX" {
			t.Errorf("row 2 = %q", got)
		}
		if got := rowPattern(s.Lines[3]); got != "XXXXXXXXXXXXXXXXXXXX" {
			t.Errorf("row 3 should be untouched: %q", got)
		}
	})
	t.Run("mode 2 (whole screen)", func(t *testing.T) {
		s := New(10, 20)
		markScreen(s, 'X')
		s.Row, s.Col = 2, 5
		funcMap[ansi.ED](s, Params{"2"})
		if got := countBlanks(s); got != 200 {
			t.Errorf("blanks = %d, want 200", got)
		}
	})
}

// labelRows sets every cell in row r to the digit rune for r (rows 0-9), so
// scrolling can be tracked by reading any column of a row.
func labelRows(s *Screen) {
	for r, line := range s.Lines {
		for c := range line {
			line[c].Value = rune('0' + r)
		}
	}
}

// rowLabel returns the marker in column 0 of row r (' ' for a blank row).
func rowLabel(s *Screen, r int) rune { return s.Lines[r][0].Value }

func TestScrollUp(t *testing.T) {
	s := New(10, 5)
	labelRows(s)
	s.Row, s.Col = 4, 2
	funcMap[ansi.SU](s, Params{"2"})
	// Rows shift up by 2; bottom two rows blank; cursor unchanged.
	for r := 0; r < 8; r++ {
		if got := rowLabel(s, r); got != rune('0'+r+2) {
			t.Errorf("row %d label = %q, want %q", r, got, rune('0'+r+2))
		}
	}
	for r := 8; r < 10; r++ {
		if got := rowLabel(s, r); got != ' ' {
			t.Errorf("row %d label = %q, want blank", r, got)
		}
	}
	if s.Row != 4 || s.Col != 2 {
		t.Errorf("cursor moved to %d,%d", s.Row, s.Col)
	}
}

func TestScrollDown(t *testing.T) {
	s := New(10, 5)
	labelRows(s)
	funcMap[ansi.SD](s, Params{"2"})
	for r := 0; r < 2; r++ {
		if got := rowLabel(s, r); got != ' ' {
			t.Errorf("row %d label = %q, want blank", r, got)
		}
	}
	for r := 2; r < 10; r++ {
		if got := rowLabel(s, r); got != rune('0'+r-2) {
			t.Errorf("row %d label = %q, want %q", r, got, rune('0'+r-2))
		}
	}
}

func TestNextLine(t *testing.T) {
	t.Run("middle moves down to column 0", func(t *testing.T) {
		s := New(10, 5)
		labelRows(s)
		s.Row, s.Col = 2, 3
		funcMap[ansi.NEL](s, nil)
		if s.Row != 3 || s.Col != 0 {
			t.Errorf("cursor = %d,%d, want 3,0", s.Row, s.Col)
		}
		if rowLabel(s, 0) != '0' {
			t.Error("contents should not scroll in the middle")
		}
	})
	t.Run("bottom scrolls up", func(t *testing.T) {
		s := New(10, 5)
		labelRows(s)
		s.Row, s.Col = 9, 3
		funcMap[ansi.NEL](s, nil)
		if s.Row != 9 || s.Col != 0 {
			t.Errorf("cursor = %d,%d, want 9,0", s.Row, s.Col)
		}
		if got := rowLabel(s, 0); got != '1' {
			t.Errorf("row 0 after scroll = %q, want '1'", got)
		}
		if got := rowLabel(s, 9); got != ' ' {
			t.Errorf("row 9 after scroll = %q, want blank", got)
		}
	})
}

func TestReverseIndex(t *testing.T) {
	t.Run("middle moves up, column unchanged", func(t *testing.T) {
		s := New(10, 5)
		labelRows(s)
		s.Row, s.Col = 2, 3
		funcMap[ansi.RI](s, nil)
		if s.Row != 1 || s.Col != 3 {
			t.Errorf("cursor = %d,%d, want 1,3", s.Row, s.Col)
		}
	})
	t.Run("top scrolls down", func(t *testing.T) {
		s := New(10, 5)
		labelRows(s)
		s.Row, s.Col = 0, 3
		funcMap[ansi.RI](s, nil)
		if s.Row != 0 || s.Col != 3 {
			t.Errorf("cursor = %d,%d, want 0,3", s.Row, s.Col)
		}
		if got := rowLabel(s, 0); got != ' ' {
			t.Errorf("row 0 after scroll = %q, want blank", got)
		}
		if got := rowLabel(s, 1); got != '0' {
			t.Errorf("row 1 after scroll = %q, want '0'", got)
		}
	})
}

func TestInsertLines(t *testing.T) {
	s := New(10, 5)
	labelRows(s)
	s.Row, s.Col = 3, 4
	funcMap[ansi.IL](s, Params{"2"})
	// Rows 0-2 unchanged; rows 3-4 blank; old rows 3-7 now at 5-9.
	for r := 0; r < 3; r++ {
		if got := rowLabel(s, r); got != rune('0'+r) {
			t.Errorf("row %d label = %q, want %q", r, got, rune('0'+r))
		}
	}
	for r := 3; r < 5; r++ {
		if got := rowLabel(s, r); got != ' ' {
			t.Errorf("row %d label = %q, want blank", r, got)
		}
	}
	for r := 5; r < 10; r++ {
		if got := rowLabel(s, r); got != rune('0'+r-2) {
			t.Errorf("row %d label = %q, want %q", r, got, rune('0'+r-2))
		}
	}
	if s.Col != 0 {
		t.Errorf("Col = %d, want 0 (left margin)", s.Col)
	}
}

func TestDeleteLines(t *testing.T) {
	s := New(10, 5)
	labelRows(s)
	s.Row, s.Col = 3, 4
	funcMap[ansi.DL](s, Params{"2"})
	// Rows 0-2 unchanged; old rows 5-9 now at 3-7; rows 8-9 blank.
	for r := 0; r < 3; r++ {
		if got := rowLabel(s, r); got != rune('0'+r) {
			t.Errorf("row %d label = %q, want %q", r, got, rune('0'+r))
		}
	}
	for r := 3; r < 8; r++ {
		if got := rowLabel(s, r); got != rune('0'+r+2) {
			t.Errorf("row %d label = %q, want %q", r, got, rune('0'+r+2))
		}
	}
	for r := 8; r < 10; r++ {
		if got := rowLabel(s, r); got != ' ' {
			t.Errorf("row %d label = %q, want blank", r, got)
		}
	}
	if s.Col != 0 {
		t.Errorf("Col = %d, want 0 (left margin)", s.Col)
	}
}

// labelCols sets row r's cells to 'a', 'b', 'c', ... so horizontal shifts can
// be read back as a string.
func labelCols(s *Screen, r int) {
	for c := range s.Lines[r] {
		s.Lines[r][c].Value = rune('a' + c)
	}
}

func rowString(line Line) string {
	b := make([]rune, len(line))
	for i, c := range line {
		b[i] = c.Value
	}
	return string(b)
}

func TestInsertChars(t *testing.T) {
	tests := []struct {
		name   string
		params Params
		want   string
	}{
		{"ICH default (1)", nil, "abc defghi"},
		{"ICH 2", Params{"2"}, "abc  defgh"},
		{"ICH clamps to right edge", Params{"99"}, "abc       "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(3, 10)
			labelCols(s, 1)
			s.Row, s.Col = 1, 3
			funcMap[ansi.ICH](s, tt.params)
			if got := rowString(s.Lines[1]); got != tt.want {
				t.Errorf("row = %q, want %q", got, tt.want)
			}
			if s.Row != 1 || s.Col != 3 {
				t.Errorf("cursor moved to %d,%d", s.Row, s.Col)
			}
		})
	}
}

func TestDeleteChars(t *testing.T) {
	tests := []struct {
		name   string
		params Params
		want   string
	}{
		{"DCH default (1)", nil, "abcefghij "},
		{"DCH 2", Params{"2"}, "abcfghij  "},
		{"DCH clamps to right edge", Params{"99"}, "abc       "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(3, 10)
			labelCols(s, 1)
			s.Row, s.Col = 1, 3
			funcMap[ansi.DCH](s, tt.params)
			if got := rowString(s.Lines[1]); got != tt.want {
				t.Errorf("row = %q, want %q", got, tt.want)
			}
			if s.Row != 1 || s.Col != 3 {
				t.Errorf("cursor moved to %d,%d", s.Row, s.Col)
			}
		})
	}
}

func TestClampRow(t *testing.T) {
	s := New(10, 20)
	tests := []struct {
		n    int
		want int
	}{
		{0, 0},
		{5, 5},
		{9, 9},
		{10, 9},  // >= Rows clamps to Rows-1
		{100, 9},
		{-1, 0},
		{-99, 0},
	}
	for _, tt := range tests {
		if got := s.ClampRow(tt.n); got != tt.want {
			t.Errorf("ClampRow(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

func TestClampCol(t *testing.T) {
	s := New(10, 20)
	tests := []struct {
		n    int
		want int
	}{
		{0, 0},
		{10, 10},
		{19, 19},
		{20, 19},  // >= Cols clamps to Cols-1
		{100, 19},
		{-1, 0},
		{-99, 0},
	}
	for _, tt := range tests {
		if got := s.ClampCol(tt.n); got != tt.want {
			t.Errorf("ClampCol(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}
