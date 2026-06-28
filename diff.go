package goterm

import (
	"fmt"
	"strings"
)

// A RowDiff is a single row that differs between two screens, as returned by
// Screen.Dump.  A and B are the row's text on each screen.
type RowDiff struct {
	Row int
	A   string
	B   string
}

// String renders the difference as "row N: <A> | <B>".
func (d RowDiff) String() string {
	return fmt.Sprintf("row %d: %q | %q", d.Row, d.A, d.B)
}

// DiffScreens compares two screen dumps (each from Screen.Dump) and returns the
// rows that differ, in order.  Rows missing from the shorter screen are compared
// as "".  It returns nil when the screens are identical, so the result doubles
// as a boolean "are they the same".
func DiffScreens(a, b []string) []RowDiff {
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	var diffs []RowDiff
	for i := 0; i < n; i++ {
		var ra, rb string
		if i < len(a) {
			ra = a[i]
		}
		if i < len(b) {
			rb = b[i]
		}
		if ra != rb {
			diffs = append(diffs, RowDiff{Row: i, A: ra, B: rb})
		}
	}
	return diffs
}

// FormatDiffs renders a slice of RowDiff as one line per difference, for test
// output.  It returns "" when diffs is empty.
func FormatDiffs(diffs []RowDiff) string {
	if len(diffs) == 0 {
		return ""
	}
	lines := make([]string, len(diffs))
	for i, d := range diffs {
		lines[i] = d.String()
	}
	return strings.Join(lines, "\n")
}

// A CellDiff is a single cell that differs between two screens (from Snapshot),
// comparing the glyph, colors, and attributes that Dump's text omits.
type CellDiff struct {
	Row, Col int
	A, B     Cell
}

// String renders the difference as "cell R,C: <A> | <B>", each side describing
// the glyph and its rendition.
func (d CellDiff) String() string {
	return fmt.Sprintf("cell %d,%d: %s | %s", d.Row, d.Col, describeCell(d.A), describeCell(d.B))
}

// DiffCells compares two screen snapshots (from Snapshot) cell by cell --
// including colors and attributes, which DiffScreens ignores -- and returns the
// differing cells in row-major order.  Ragged sizes are compared against the
// zero Cell.  It returns nil when the snapshots are identical.
//
// The Erased flag is provenance, not a visible difference (an erased space and a
// written space look identical), so it is excluded from the comparison; DiffCells
// reports only differences the eye would see.
func DiffCells(a, b [][]Cell) []CellDiff {
	nrows := len(a)
	if len(b) > nrows {
		nrows = len(b)
	}
	var diffs []CellDiff
	for r := 0; r < nrows; r++ {
		var ra, rb []Cell
		if r < len(a) {
			ra = a[r]
		}
		if r < len(b) {
			rb = b[r]
		}
		ncols := len(ra)
		if len(rb) > ncols {
			ncols = len(rb)
		}
		for c := 0; c < ncols; c++ {
			var ca, cb Cell
			if c < len(ra) {
				ca = ra[c]
			}
			if c < len(rb) {
				cb = rb[c]
			}
			// Erased is invisible provenance; compare only the visible rendition.
			va, vb := ca, cb
			va.Erased, vb.Erased = false, false
			if va != vb {
				diffs = append(diffs, CellDiff{Row: r, Col: c, A: ca, B: cb})
			}
		}
	}
	return diffs
}

// FormatCellDiffs renders a slice of CellDiff as one line per difference.  It
// returns "" when diffs is empty.
func FormatCellDiffs(diffs []CellDiff) string {
	if len(diffs) == 0 {
		return ""
	}
	lines := make([]string, len(diffs))
	for i, d := range diffs {
		lines[i] = d.String()
	}
	return strings.Join(lines, "\n")
}

// describeCell renders a cell's glyph and the rendition that differs from the
// default: any set attributes and any non-default colors.
func describeCell(c Cell) string {
	var tags []string
	for _, a := range []struct {
		mask uint
		name string
	}{
		{BoldMask, "bold"}, {FaintMask, "faint"}, {ItalicMask, "italic"},
		{UnderlineMask, "underline"}, {SlowBlinkMask, "blink"}, {RapidBlinkMask, "rapidblink"},
		{InverseMask, "inverse"}, {HiddenMask, "hidden"}, {StrikethroughMask, "strike"},
	} {
		if c.Attributes&int(a.mask) != 0 {
			tags = append(tags, a.name)
		}
	}
	if c.Foreground != DefaultForeground {
		tags = append(tags, fmt.Sprintf("fg=%d", c.Foreground))
	}
	if c.Background != DefaultBackground {
		tags = append(tags, fmt.Sprintf("bg=%d", c.Background))
	}
	if c.Wide {
		tags = append(tags, "wide")
	}
	s := fmt.Sprintf("%q", c.Value)
	if len(tags) > 0 {
		s += "[" + strings.Join(tags, ",") + "]"
	}
	return s
}
