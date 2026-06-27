package goterm

import (
	"fmt"
	"strconv"
	"strings"

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

	// Terminal modes (set/reset via SM/RM).  These are simple flags read by
	// the write and render paths; they default to a freshly reset terminal.
	CursorVisible bool // DECTCEM (?25): cursor is shown
	AutoWrap      bool // DECAWM (?7): wrap to the next line at the right margin
	InsertMode    bool // IRM (4): printed characters shift the line right

	term *Term // owning terminal, if any (set by New); nil for a standalone screen
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

// NewScreen returns a new screen of the given size with every cell initialized
// to a blank (space) on a black-on-white background and tab stops every 8 columns.
func NewScreen(rows, cols int) *Screen {
	s := &Screen{
		Rows:          rows,
		Cols:          cols,
		Lines:         make([]Line, rows),
		Cur:           defaultCell(),
		Tabs:          make([]bool, cols),
		CursorVisible: true, // cursor shown and autowrap on by default
		AutoWrap:      true,
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

// index moves the cursor down one line, scrolling the screen up when it is
// already on the last line.
func (s *Screen) index() {
	if s.Row >= s.Rows-1 {
		s.scrollUp(1)
	} else {
		s.Row++
	}
}

// put writes a printable rune at the cursor (using the current pen) and advances
// it.  At the right margin it performs a deferred wrap: the cursor is allowed to
// rest one column past the last, and the wrap to the next line happens when the
// next rune is written, and only when AutoWrap is set.  In InsertMode the rest
// of the line is shifted right to make room.
func (s *Screen) put(r rune) {
	if s.Col >= s.Cols {
		if s.AutoWrap {
			s.Col = 0
			s.index()
		} else {
			s.Col = s.Cols - 1
		}
	}
	if s.InsertMode {
		s.insertChars(s.Row, s.Col, 1)
	}
	cell := s.blank()
	cell.Value = r
	s.Lines[s.Row][s.Col] = cell
	if s.AutoWrap || s.Col < s.Cols-1 {
		s.Col++
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

// setAttr sets the given attribute bit in the current pen.
func (s *Screen) setAttr(attr uint) {
	s.Cur.Attributes = int(addAttr(uint(s.Cur.Attributes), attr))
}

// clearAttr clears the given attribute mask bits from the current pen.
func (s *Screen) clearAttr(mask uint) {
	s.Cur.Attributes &^= int(mask)
}

// readExtColor decodes an extended-color selector (SGR 38/48) whose selector
// code is at p[i].  It returns the resulting palette index, the number of
// additional parameters consumed, and whether a usable color was produced.
// "5;n" selects 256-color index n; "2;r;g;b" is truecolor, which a palette
// index cannot represent, so it is consumed but not applied.
func readExtColor(p Params, i int) (color, advance int, ok bool) {
	switch p.Int(i + 1) {
	case 5:
		return p.Int(i + 2), 2, true
	case 2:
		return 0, 4, false
	default:
		return 0, 0, false
	}
}

// applySGR applies a Select Graphic Rendition sequence to the current pen.
// Supported: reset (0), the common attributes and their resets, the 8 basic
// and 8 bright colors, and 256-color (38;5/48;5).  Brightness is encoded in
// the palette index (8-15), not in an attribute bit.  Truecolor (38;2/48;2)
// is parsed but not applied since cells store only a palette index.
func (s *Screen) applySGR(p Params) {
	if len(p) == 0 {
		s.Cur = defaultCell()
		return
	}
	for i := 0; i < len(p); i++ {
		switch n := p.Int(i); {
		case n == 0:
			s.Cur = defaultCell()
		case n == 1:
			s.setAttr(BoldAttr)
		case n == 2:
			s.setAttr(FaintAttr)
		case n == 3:
			s.setAttr(ItalicAttr)
		case n == 4:
			s.setAttr(UnderlineAttr)
		case n == 5:
			s.setAttr(SlowBlinkAttr)
		case n == 6:
			s.setAttr(RapidBlinkAttr)
		case n == 7:
			s.setAttr(InverseAttr)
		case n == 8:
			s.setAttr(HiddenAttr)
		case n == 9:
			s.setAttr(StrikethroughAttr)
		case n == 22:
			s.clearAttr(BoldMask | FaintMask)
		case n == 23:
			s.clearAttr(ItalicMask)
		case n == 24:
			s.clearAttr(UnderlineMask)
		case n == 25:
			s.clearAttr(SlowBlinkMask | RapidBlinkMask)
		case n == 27:
			s.clearAttr(InverseMask)
		case n == 28:
			s.clearAttr(HiddenMask)
		case n == 29:
			s.clearAttr(StrikethroughMask)
		case n >= 30 && n <= 37:
			s.Cur.Foreground = n - 30
		case n == 38:
			c, adv, ok := readExtColor(p, i)
			if ok {
				s.Cur.Foreground = c
			}
			i += adv
		case n == 39:
			s.Cur.Foreground = DefaultForeground
		case n >= 40 && n <= 47:
			s.Cur.Background = n - 40
		case n == 48:
			c, adv, ok := readExtColor(p, i)
			if ok {
				s.Cur.Background = c
			}
			i += adv
		case n == 49:
			s.Cur.Background = DefaultBackground
		case n >= 90 && n <= 97:
			s.Cur.Foreground = (n - 90) + 8
		case n >= 100 && n <= 107:
			s.Cur.Background = (n - 100) + 8
		}
	}
}

// modeParams returns the mode numbers in p and whether they are DEC private
// modes (the parameter string is prefixed with '?').  pborman/ansi does not
// split private-mode parameters on ';', so we split them here; for example
// "?7;25" yields modes [7, 25] with private true.
func modeParams(p Params) (modes []int, private bool) {
	s := p.Str(0)
	if strings.HasPrefix(s, "?") {
		private = true
		s = s[1:]
	}
	if s == "" {
		return nil, private
	}
	for _, f := range strings.Split(s, ";") {
		n, _ := strconv.Atoi(f)
		modes = append(modes, n)
	}
	return modes, private
}

// setModes sets (on=true) or resets (on=false) the terminal modes named in p.
// Unrecognized modes are ignored.
func (s *Screen) setModes(p Params, on bool) {
	modes, private := modeParams(p)
	for _, m := range modes {
		switch {
		case private && m == 7: // DECAWM
			s.AutoWrap = on
		case private && m == 25: // DECTCEM
			s.CursorVisible = on
		case private && m == 1049: // alternate screen with cursor save/restore
			s.switchAlternate(on, true)
		case private && (m == 47 || m == 1047): // alternate screen
			s.switchAlternate(on, false)
		case !private && m == 4: // IRM
			s.InsertMode = on
		}
	}
}

// respond sends bytes back to the program on the owning Term's return stream.
// A standalone screen with no Term has nowhere to reply, so it is a no-op.
func (s *Screen) respond(b []byte) {
	if s.term != nil {
		s.term.Send(b)
	}
}

// switchAlternate enters (on=true) or exits the alternate screen via the owning
// Term.  When restore is true the primary's cursor is preserved across the
// round trip (?1049 semantics).  It is a no-op for a standalone screen with no
// Term.
func (s *Screen) switchAlternate(on, restore bool) {
	if s.term == nil {
		return
	}
	if on {
		s.term.enterAlternate()
	} else {
		s.term.exitAlternate(restore)
	}
}

// clear blanks every cell using the current pen and moves the cursor home.
func (s *Screen) clear() {
	for r := 0; r < s.Rows; r++ {
		s.fill(r, 0, s.Cols-1)
	}
	s.Row, s.Col = 0, 0
}

// funcMap is a map of ansi.Names to functions that implement that escape
// sequence.  The function is provided the Screen to apply it to as well as the
// parameters Gathered.
var funcMap = map[ansi.Name]func(*Screen, Params){
	ansi.BS: func(s *Screen, p Params) { s.Col = s.ClampCol(s.Col - 1) },
	// BEL has no screen effect; it bumps the owning Term's bell counter so
	// tests can observe it.  A standalone screen (no Term) ignores it.
	ansi.BEL: func(s *Screen, p Params) {
		if s.term != nil {
			s.term.Bell++
		}
	},
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
	ansi.SU:  func(s *Screen, p Params) { s.scrollUp(p.Amt(0)) },
	ansi.SD:  func(s *Screen, p Params) { s.scrollDown(p.Amt(0)) },
	ansi.NEL: func(s *Screen, p Params) { s.Col = 0; s.index() },
	ansi.RI: func(s *Screen, p Params) {
		if s.Row <= 0 {
			s.scrollDown(1)
		} else {
			s.Row--
		}
	},
	// LF (Index) moves down one line, scrolling at the bottom.  Unlike NEL it
	// does not change the column.
	ansi.LF: func(s *Screen, p Params) { s.index() },

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

	// Select Graphic Rendition: update the current pen (colors/attributes).
	ansi.SGR: func(s *Screen, p Params) { s.applySGR(p) },

	// Set/Reset Mode.  Handles the flag modes DECTCEM (?25), DECAWM (?7), and
	// IRM (4); other modes are ignored.
	ansi.SM: func(s *Screen, p Params) { s.setModes(p, true) },
	ansi.RM: func(s *Screen, p Params) { s.setModes(p, false) },

	// Device reports.  These answer on the Term's return stream so a program
	// querying the terminal at startup does not block.
	//
	// DSR 5 reports "terminal OK" (ESC[0n); DSR 6 reports the 1-based cursor
	// position (CPR, ESC[row;colR).
	ansi.DSR: func(s *Screen, p Params) {
		switch p.Int(0) {
		case 5:
			s.respond([]byte("\x1b[0n"))
		case 6:
			s.respond([]byte(fmt.Sprintf("\x1b[%d;%dR", s.ClampRow(s.Row)+1, s.ClampCol(s.Col)+1)))
		}
	},
	// Primary Device Attributes: report a basic VT102-class terminal (ESC[?6c).
	ansi.DA: func(s *Screen, p Params) {
		if q := p.Str(0); q == "" || q == "0" {
			s.respond([]byte("\x1b[?6c"))
		}
	},
}
