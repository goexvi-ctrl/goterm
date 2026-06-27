package goterm

import (
	"os"
	"testing"
	"time"
)

// startEditor launches an editor on its own 24x80 terminal, skipping the test if
// the binary is not present.
func startEditor(t *testing.T, path string) *Term {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("%s not available", path)
	}
	tm := New(24, 80)
	if err := tm.Start(path); err != nil {
		t.Fatalf("Start %s: %v", path, err)
	}
	return tm
}

// editorPair drives two terminals in lockstep: identical size, identical input.
type editorPair struct{ A, B *Term }

func (p editorPair) send(in []byte) { p.A.Send(in); p.B.Send(in) }

// waitReady waits for both editors to reach a state (use a known predicate; an
// editor can go quiet mid-startup before it is ready for input, so WaitQuiet is
// not a safe readiness signal).
func (p editorPair) waitReady(pred func([]string) bool) bool {
	return p.A.WaitFor(5*time.Second, pred) && p.B.WaitFor(5*time.Second, pred)
}

// settle lets both editors finish redrawing after input.
func (p editorPair) settle() {
	p.A.WaitQuiet(200*time.Millisecond, 2*time.Second)
	p.B.WaitQuiet(200*time.Millisecond, 2*time.Second)
}

func (p editorPair) close() { p.A.Close(); p.B.Close() }

func capCellDiffs(d []CellDiff, n int) []CellDiff {
	if len(d) > n {
		return d[:n]
	}
	return d
}

// TestCompareGoviNvi drives govi and nvi with identical input and reports where
// their screens differ -- in text (DiffScreens) and in colors/attributes
// (DiffCells), the latter catching things like reverse video that Dump omits.
// It is a comparison report: it logs the differences rather than asserting the
// editors are identical (they legitimately differ while govi is in progress).
func TestCompareGoviNvi(t *testing.T) {
	nvi := startEditor(t, "/opt/homebrew/bin/nvi")
	govi := startEditor(t, "/Users/claude/bin/govi")
	pair := editorPair{A: nvi, B: govi}
	defer pair.close()

	if !pair.waitReady(func(d []string) bool { return d[1] == "~" }) {
		t.Fatal("an editor did not reach the empty-buffer state")
	}

	// An invalid ex command: both report it on the status line, where nvi uses
	// reverse video.
	pair.send([]byte(":xyzzy\r"))
	pair.settle()

	text := DiffScreens(nvi.Dump(), govi.Dump())
	cells := DiffCells(nvi.Snapshot(), govi.Snapshot())
	t.Logf("text differences (nvi | govi):\n%s", FormatDiffs(text))
	t.Logf("cell/attribute differences: %d (showing first 6) (nvi | govi):\n%s",
		len(cells), FormatCellDiffs(capCellDiffs(cells, 6)))

	pair.send([]byte(":q!\r"))
}
