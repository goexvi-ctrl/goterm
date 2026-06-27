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
