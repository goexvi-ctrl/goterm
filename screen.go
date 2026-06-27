package goterm

import (
	"strconv"

	"github.com/pborman/ansi"
)

// These are the basic indicies for the colors
const (
	Black = iota
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

// These are the offsets for colors
const (
	Foreground = 30 // The normal foreground colors
	Background = 40 // The normal background colors
	Bright     = 60 // Add to make them bright
)

// ANSI attributes.
const (
	NormalAttr = uint(iota)
	BoldAttr
	FaintAttr
	ItalicAttr
	UnderlineAttr
	SlowBlinkAttr
	RapidBlinkAttr
	InverseAttr
	HiddenAttr
	StrikethroughAttr
	BrightAttr // Technically not an attribute but a color modifier
)

// Masks for attribute field
const (
	BoldMask          = (1 << (BoldAttr - 1))
	FaintMask         = (1 << (FaintAttr - 1))
	ItalicMask        = (1 << (ItalicAttr - 1))
	UnderlineMask     = (1 << (UnderlineAttr - 1))
	SlowBlinkMask     = (1 << (SlowBlinkAttr - 1))
	RapidBlinkMask    = (1 << (RapidBlinkAttr - 1))
	InverseMask       = (1 << (InverseAttr - 1))
	HiddenMask        = (1 << (HiddenAttr - 1))
	StrikethroughMask = (1 << (StrikethroughAttr - 1))
	BrightMask        = (1 << (BrightAttr - 1))
)

func addAttr(bits, attr uint) uint {
	if attr == NormalAttr {
		return 0
	}
	return bits | (1 << (attr - 1))
}

// The default foreground and background colors used for blank cells and for
// SGR "default color" (codes 39 and 49).
const (
	DefaultForeground = Black
	DefaultBackground = White
)

// A Screen represents a virtual terminal screen.
type Screen struct {
	Rows  int
	Cols  int
	Row   int
	Col   int
	Lines []Line
	Cur   Cell   // Current graphic rendition (the "pen") applied to new and erased cells
	Tabs  []bool // Tabs[c] is true when column c is a tab stop
	// TODO: Add scrolling regions
}

// A Line is a single row of Cells
type Line []Cell

// A Cell is a single position on the screen.
// Colors are stored as the color index
type Cell struct {
	Value      rune
	Foreground int
	Background int
	Attributes int  // Bit field of attributes
	Wide       bool // Wide character suffix
}

// defaultCell returns a blank cell with the default colors and no attributes.
func defaultCell() Cell {
	return Cell{Value: ' ', Foreground: DefaultForeground, Background: DefaultBackground}
}

// blank returns a blank cell using the screen's current graphic rendition.
// Erase and insert operations fill vacated positions with this cell so that
// they adopt the current background color and attributes.
func (s *Screen) blank() Cell {
	c := s.Cur
	c.Value = ' '
	c.Wide = false
	return c
}

// New returns a new screen of the given size with every cell initialized to a
// blank (space) on a black-on-white background and tab stops every 8 columns.
func New(rows, cols int) *Screen {
	s := &Screen{
		Rows:  rows,
		Cols:  cols,
		Lines: make([]Line, rows),
		Cur:   defaultCell(),
		Tabs:  make([]bool, cols),
	}
	for i := range s.Lines {
		line := make(Line, cols)
		for j := range line {
			line[j] = defaultCell()
		}
		s.Lines[i] = line
	}
	for c := range s.Tabs {
		s.Tabs[c] = c%8 == 0
	}
	return s
}

// Type Params is the list of parameters provided to an ANSI escape sequence
type Params []string

// Int returns the n'th position of p as an int or 0 if n is out of range or p[n] is not an integer.
func (p Params) Int(n int) int {
	if n < 0 || n >= len(p) {
		return 0
	}
	v, _ := strconv.Atoi(p[n])
	return v
}

// Amt returns the n'th position of p as a positive movement amount, defaulting
// to 1 when the parameter is missing or non-positive.  ANSI cursor movement
// sequences treat an omitted or zero parameter as 1.
func (p Params) Amt(n int) int {
	if v := p.Int(n); v > 0 {
		return v
	}
	return 1
}

// Str returns the n'th position of p or "" if n is out of range.
func (p Params) Str(n int) string {
	if n < 0 || n >= len(p) {
		return ""
	}
	return p[n]
}

// ClampRow returns n clamped to [0..s.Rows).
func (s *Screen) ClampRow(n int) int {
	switch {
	case n < 0:
		return 0
	case n < s.Rows:
		return n
	default:
		return s.Rows - 1
	}
}

// ClampCol returns n clamped to [0..s.Cols).
func (s *Screen) ClampCol(n int) int {
	switch {
	case n < 0:
		return 0
	case n < s.Cols:
		return n
	default:
		return s.Cols - 1
	}
}

// fill sets columns [c0, c1] (inclusive) of row r to blank cells using the
// current pen.  Indices are clamped to the screen; an empty range is a no-op.
func (s *Screen) fill(r, c0, c1 int) {
	if r < 0 || r >= s.Rows {
		return
	}
	if c0 < 0 {
		c0 = 0
	}
	if c1 > s.Cols-1 {
		c1 = s.Cols - 1
	}
	line := s.Lines[r]
	b := s.blank()
	for c := c0; c <= c1; c++ {
		line[c] = b
	}
}

// scrollUp moves the screen contents up by n rows, discarding the top n rows
// and blanking n new rows at the bottom.  The cursor is not moved.
func (s *Screen) scrollUp(n int) {
	if n <= 0 {
		return
	}
	if n > s.Rows {
		n = s.Rows
	}
	for r := 0; r < s.Rows-n; r++ {
		copy(s.Lines[r], s.Lines[r+n])
	}
	for r := s.Rows - n; r < s.Rows; r++ {
		s.fill(r, 0, s.Cols-1)
	}
}

// scrollDown moves the screen contents down by n rows, discarding the bottom n
// rows and blanking n new rows at the top.  The cursor is not moved.
func (s *Screen) scrollDown(n int) {
	if n <= 0 {
		return
	}
	if n > s.Rows {
		n = s.Rows
	}
	for r := s.Rows - 1; r >= n; r-- {
		copy(s.Lines[r], s.Lines[r-n])
	}
	for r := 0; r < n; r++ {
		s.fill(r, 0, s.Cols-1)
	}
}

// insertLines inserts n blank lines at row "at", pushing that row and those
// below it down; lines pushed past the bottom are discarded.
func (s *Screen) insertLines(at, n int) {
	if at < 0 || at >= s.Rows || n <= 0 {
		return
	}
	if n > s.Rows-at {
		n = s.Rows - at
	}
	for r := s.Rows - 1; r >= at+n; r-- {
		copy(s.Lines[r], s.Lines[r-n])
	}
	for r := at; r < at+n; r++ {
		s.fill(r, 0, s.Cols-1)
	}
}

// deleteLines deletes n lines starting at row "at", pulling the lines below up
// and blanking n lines at the bottom.
func (s *Screen) deleteLines(at, n int) {
	if at < 0 || at >= s.Rows || n <= 0 {
		return
	}
	if n > s.Rows-at {
		n = s.Rows - at
	}
	for r := at; r < s.Rows-n; r++ {
		copy(s.Lines[r], s.Lines[r+n])
	}
	for r := s.Rows - n; r < s.Rows; r++ {
		s.fill(r, 0, s.Cols-1)
	}
}

// insertChars inserts n blank cells at (r, c), shifting the rest of the line
// right; cells pushed past the right edge are discarded.
func (s *Screen) insertChars(r, c, n int) {
	if r < 0 || r >= s.Rows || c < 0 || c >= s.Cols || n <= 0 {
		return
	}
	if n > s.Cols-c {
		n = s.Cols - c
	}
	line := s.Lines[r]
	for x := s.Cols - 1; x >= c+n; x-- {
		line[x] = line[x-n]
	}
	b := s.blank()
	for x := c; x < c+n; x++ {
		line[x] = b
	}
}

// deleteChars deletes n cells at (r, c), shifting the rest of the line left and
// blanking n cells at the right edge.
func (s *Screen) deleteChars(r, c, n int) {
	if r < 0 || r >= s.Rows || c < 0 || c >= s.Cols || n <= 0 {
		return
	}
	if n > s.Cols-c {
		n = s.Cols - c
	}
	line := s.Lines[r]
	for x := c; x < s.Cols-n; x++ {
		line[x] = line[x+n]
	}
	b := s.blank()
	for x := s.Cols - n; x < s.Cols; x++ {
		line[x] = b
	}
}

// nextTab returns the first tab stop after col, or the last column if there is
// none.
func (s *Screen) nextTab(col int) int {
	for c := col + 1; c < s.Cols; c++ {
		if s.Tabs[c] {
			return c
		}
	}
	return s.Cols - 1
}

// prevTab returns the last tab stop before col, or column 0 if there is none.
func (s *Screen) prevTab(col int) int {
	for c := col - 1; c > 0; c-- {
		if s.Tabs[c] {
			return c
		}
	}
	return 0
}

// funcMap is a map of ansi.Names to functions that implement that escape
// sequence.  The function is provided the Screen to apply it to as well as the
// parameters Gathered.
var funcMap = map[ansi.Name]func(*Screen, Params){
	ansi.BS:  func(s *Screen, p Params) { s.Col = s.ClampCol(s.Col - 1) },
	ansi.CUU: func(s *Screen, p Params) { s.Row = s.ClampRow(s.Row - p.Amt(0)) },
	ansi.CUD: func(s *Screen, p Params) { s.Row = s.ClampRow(s.Row + p.Amt(0)) },
	ansi.CUF: func(s *Screen, p Params) { s.Col = s.ClampCol(s.Col + p.Amt(0)) },
	ansi.CUB: func(s *Screen, p Params) { s.Col = s.ClampCol(s.Col - p.Amt(0)) },
	ansi.CHA: func(s *Screen, p Params) { s.Col = s.ClampCol(p.Amt(0) - 1) },
	ansi.CNL: func(s *Screen, p Params) { s.Col = 0; s.Row = s.ClampRow(s.Row + p.Amt(0)) },
	ansi.CPL: func(s *Screen, p Params) { s.Col = 0; s.Row = s.ClampRow(s.Row - p.Amt(0)) },
	ansi.CR:  func(s *Screen, p Params) { s.Col = 0 },

	// Absolute cursor positioning.  CUP and HVP take a 1-based row and column;
	// VPA and HPA set a single 1-based axis.  A missing parameter defaults to 1.
	ansi.CUP: func(s *Screen, p Params) { s.Row = s.ClampRow(p.Amt(0) - 1); s.Col = s.ClampCol(p.Amt(1) - 1) },
	ansi.HVP: func(s *Screen, p Params) { s.Row = s.ClampRow(p.Amt(0) - 1); s.Col = s.ClampCol(p.Amt(1) - 1) },
	ansi.VPA: func(s *Screen, p Params) { s.Row = s.ClampRow(p.Amt(0) - 1) },
	ansi.HPA: func(s *Screen, p Params) { s.Col = s.ClampCol(p.Amt(0) - 1) },

	// Relative cursor positioning by a single axis.  HPR/VPR move forward
	// (right/down), HPB/VPB backward (left/up).  These mirror CUF/CUB/CUD/CUU.
	ansi.HPR: func(s *Screen, p Params) { s.Col = s.ClampCol(s.Col + p.Amt(0)) },
	ansi.HPB: func(s *Screen, p Params) { s.Col = s.ClampCol(s.Col - p.Amt(0)) },
	ansi.VPR: func(s *Screen, p Params) { s.Row = s.ClampRow(s.Row + p.Amt(0)) },
	ansi.VPB: func(s *Screen, p Params) { s.Row = s.ClampRow(s.Row - p.Amt(0)) },

	// Erase.  The cursor does not move.  ED/EL select a region by mode (0 =
	// cursor to end, 1 = start to cursor, 2 = all); the mode defaults to 0.
	ansi.ED: func(s *Screen, p Params) {
		switch p.Int(0) {
		case 1: // start of screen through cursor
			for r := 0; r < s.Row; r++ {
				s.fill(r, 0, s.Cols-1)
			}
			s.fill(s.Row, 0, s.Col)
		case 2, 3: // whole screen (3 also clears scrollback, which we lack)
			for r := 0; r < s.Rows; r++ {
				s.fill(r, 0, s.Cols-1)
			}
		default: // 0: cursor through end of screen
			s.fill(s.Row, s.Col, s.Cols-1)
			for r := s.Row + 1; r < s.Rows; r++ {
				s.fill(r, 0, s.Cols-1)
			}
		}
	},
	ansi.EL: func(s *Screen, p Params) {
		switch p.Int(0) {
		case 1: // start of line through cursor
			s.fill(s.Row, 0, s.Col)
		case 2: // whole line
			s.fill(s.Row, 0, s.Cols-1)
		default: // 0: cursor through end of line
			s.fill(s.Row, s.Col, s.Cols-1)
		}
	},
	// ECH erases n characters starting at the cursor (default 1).
	ansi.ECH: func(s *Screen, p Params) { s.fill(s.Row, s.Col, s.Col+p.Amt(0)-1) },

	// Scrolling and line feeds.  SU/SD scroll the contents without moving the
	// cursor.  NEL moves to the start of the next line, scrolling up at the
	// bottom; RI moves up one line, scrolling down at the top.
	ansi.SU: func(s *Screen, p Params) { s.scrollUp(p.Amt(0)) },
	ansi.SD: func(s *Screen, p Params) { s.scrollDown(p.Amt(0)) },
	ansi.NEL: func(s *Screen, p Params) {
		s.Col = 0
		if s.Row >= s.Rows-1 {
			s.scrollUp(1)
		} else {
			s.Row++
		}
	},
	ansi.RI: func(s *Screen, p Params) {
		if s.Row <= 0 {
			s.scrollDown(1)
		} else {
			s.Row--
		}
	},

	// Insert/delete lines at the cursor row.  Per DEC behavior the cursor is
	// moved to the left margin (column 0).
	ansi.IL: func(s *Screen, p Params) { s.insertLines(s.Row, p.Amt(0)); s.Col = 0 },
	ansi.DL: func(s *Screen, p Params) { s.deleteLines(s.Row, p.Amt(0)); s.Col = 0 },

	// Insert/delete characters at the cursor; the cursor does not move.
	ansi.ICH: func(s *Screen, p Params) { s.insertChars(s.Row, s.Col, p.Amt(0)) },
	ansi.DCH: func(s *Screen, p Params) { s.deleteChars(s.Row, s.Col, p.Amt(0)) },

	// Tabs.  HT advances to the next stop; CHT/CBT move forward/back n stops.
	// HTS sets a stop at the cursor; TBC clears the stop at the cursor (mode 0)
	// or all stops (mode 3).
	ansi.HT: func(s *Screen, p Params) { s.Col = s.nextTab(s.Col) },
	ansi.CHT: func(s *Screen, p Params) {
		for i := 0; i < p.Amt(0); i++ {
			s.Col = s.nextTab(s.Col)
		}
	},
	ansi.CBT: func(s *Screen, p Params) {
		for i := 0; i < p.Amt(0); i++ {
			s.Col = s.prevTab(s.Col)
		}
	},
	ansi.HTS: func(s *Screen, p Params) {
		if s.Col >= 0 && s.Col < s.Cols {
			s.Tabs[s.Col] = true
		}
	},
	ansi.TBC: func(s *Screen, p Params) {
		switch p.Int(0) {
		case 3, 5: // clear all stops
			for c := range s.Tabs {
				s.Tabs[c] = false
			}
		default: // 0: clear the stop at the cursor
			if s.Col >= 0 && s.Col < s.Cols {
				s.Tabs[s.Col] = false
			}
		}
	},
}
