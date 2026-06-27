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

// A Screen represents a virtual terminal screen.
type Screen struct {
	Rows  int
	Cols  int
	Row   int
	Col   int
	Lines []Line
	// TODO: Add scrolling regions
}

// A Line is a single row of Cells
type Line []*Cell

// A Cell is a single position on the screen.
// Colors are stored as the color index
type Cell struct {
	Value      rune
	Foreground int
	Background int
	Attributes int  // Bit field of attributes
	Wide       bool // Wide character suffix
}

// New returns a new screen of the given size with every cell initialized to a
// blank (space) on a black-on-white background.
func New(rows, cols int) *Screen {
	s := &Screen{
		Rows:  rows,
		Cols:  cols,
		Lines: make([]Line, rows),
	}
	for i := range s.Lines {
		line := make([]*Cell, cols)
		for j := range line {
			line[j] = &Cell{
				Value:      ' ',
				Foreground: Black,
				Background: White,
			}
		}
		s.Lines[i] = line
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
}
