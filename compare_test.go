package goterm

import (
	"fmt"
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

// makeNumberedFile writes n lines "001".."NNN" to a temp file: distinct, sortable
// content so the scrolled-to position is unambiguous.
func makeNumberedFile(t *testing.T, n int) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "cmp-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= n; i++ {
		fmt.Fprintf(f, "%03d\n", i)
	}
	f.Close()
	return f.Name()
}

func startEditorFile(t *testing.T, path, file string, rows, cols int) *Term {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("%s not available", path)
	}
	tm := New(rows, cols)
	if err := tm.Start(path, file); err != nil {
		t.Fatalf("Start %s: %v", path, err)
	}
	return tm
}

// TestCompareScroll compares scroll/motion commands between govi and nvi on a
// multi-screen numbered file, reporting where the view (top line) or cursor
// diverges.  Finding: ^F (forward one screen) diverges -- nvi pages the viewport
// forward (window-2 lines, cursor to the top of the new page), while govi leaves
// the view in place and only moves the cursor.  ^B/^D/^U then compound it.
func TestCompareScroll(t *testing.T) {
	// Both editors open the SAME fixture: disable the advisory file lock
	// (govi implements nvi's flock, #45) or the lock-race loser sits at a
	// "Press any key" prompt that eats the first sent byte ("1G" -> "G").
	// Which editor loses alternated run to run once the goterm terminfo
	// gained xenl/REP and shifted startup timing (2026-07-03 QA review).
	t.Setenv("EXINIT", "set nolock")
	file := makeNumberedFile(t, 60)
	nvi := startEditorFile(t, "/opt/homebrew/bin/nvi", file, 12, 40)
	defer nvi.Close()
	govi := startEditorFile(t, "/Users/claude/bin/govi", file, 12, 40)
	defer govi.Close()
	pair := editorPair{A: nvi, B: govi}

	if !pair.waitReady(func(d []string) bool { return d[len(d)-1] != "" }) {
		t.Fatal("editors did not finish loading the file")
	}
	// Let startup output finish before the first send: the status line can be
	// populated a beat before nvi reads input, and a key sent in that window
	// is swallowed ("1G" decays to "G"). Same order as runDivCase.
	pair.settle()

	body := func(d []string) []string { return d[:len(d)-1] } // drop the status line
	report := func(label string) {
		pair.settle()
		ar, ac := nvi.Cursor()
		br, bc := govi.Cursor()
		bodyMatch := len(DiffScreens(body(nvi.Dump()), body(govi.Dump()))) == 0
		curMatch := ar == br && ac == bc
		verdict := "match"
		if !bodyMatch || !curMatch {
			verdict = "DIVERGE"
		}
		t.Logf("[%-6s] %s  nvi(top=%q cur=%d,%d) govi(top=%q cur=%d,%d)",
			label, verdict, nvi.Dump()[0], ar, ac, govi.Dump()[0], br, bc)
	}

	for _, s := range []struct {
		label, keys string
	}{
		{"1G", "1G"}, {"j", "j"}, {"G", "G"}, {"1G", "1G"},
		{"^F", "\x06"}, {"^B", "\x02"}, {"^D", "\x04"}, {"^U", "\x15"},
	} {
		pair.send([]byte(s.keys))
		report(s.label)
	}

	// Document the ^F divergence concretely from a known start (top of file).
	pair.send([]byte("1G"))
	pair.settle()
	pair.send([]byte("\x06"))
	pair.settle()
	t.Logf("after 1G then ^F: nvi view starts at %q (cursor %s), govi view starts at %q (cursor %s)",
		nvi.Dump()[0], cursorStr(nvi), govi.Dump()[0], cursorStr(govi))

	pair.send([]byte(":q!\r"))
}

func cursorStr(t *Term) string {
	r, c := t.Cursor()
	return fmt.Sprintf("%d,%d", r, c)
}

// TestGoviClearsWithErase guards the forked tcell fix: clearing a line must emit
// a real erase (cells come through Erased=true) rather than overwriting the line
// with spaces (Erased=false) -- the upstream-tcell breakage that, e.g., makes a
// terminal cut-and-paste grab trailing spaces.  The Cell.Erased flag is what lets
// us tell the two apart; plain text (Dump) cannot.  nvi is checked too as the
// reference for correct behavior.  Skips if either editor binary is absent.
func TestGoviClearsWithErase(t *testing.T) {
	for _, p := range []string{nviPath, goviPath} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("%s not available", p)
		}
	}
	const cleared = 16 // width of the line of X's that gets deleted
	file := writeLines(t, []string{"XXXXXXXXXXXXXXXX"})

	open := func(path string) *Term {
		tm := New(6, 24)
		if err := tm.Start(path, file); err != nil {
			t.Fatalf("start %s: %v", path, err)
		}
		return tm
	}
	nvi := open(nviPath)
	govi := open(goviPath)
	pair := editorPair{A: nvi, B: govi}
	defer pair.close()
	if !pair.waitReady(func(d []string) bool { return d[0] != "" }) {
		t.Fatal("editors did not load the file")
	}
	pair.settle()

	// The X's are written (Erased=false) on load; delete them with 0D.
	pair.send([]byte("0D"))
	pair.settle()

	check := func(name string, tm *Term) {
		row := tm.Snapshot()[0]
		for c := 0; c < cleared; c++ {
			if row[c].Value != " " {
				t.Errorf("%s: col %d not blank after 0D: %q", name, c, row[c].Value)
			}
			if !row[c].Erased {
				t.Errorf("%s: col %d cleared with a written space, not an erase (tcell regression)", name, c)
			}
		}
	}
	check("nvi", nvi)
	check("govi", govi)
}
