package goterm

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// This file mines behavioral divergences between govi (/Users/claude/bin/govi)
// and nvi (/opt/homebrew/bin/nvi).  Each scenario starts BOTH editors fresh on
// an identical file and terminal, sends an identical key sequence, then compares
// the body (all rows except the status line) and the cursor position.  Starting
// fresh per scenario avoids state bleed, so a divergence is attributable to the
// one sequence under test.
//
// The goal is a report, not a pass/fail gate: govi is a work in progress and is
// expected to differ in places.  Confirmed, reproducible divergences are
// curated into GOTERM_DIVERGENCES.md in the govi source tree
// (/Users/claude/src/nvi/govi), the single divergence catalog.

const (
	goviPath = "/Users/claude/bin/govi"
	nviPath  = "/opt/homebrew/bin/nvi"
)

// writeLines writes lines (newline-terminated) to a temp file and returns its
// path.
func writeLines(t *testing.T, lines []string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "buf.txt")
	body := ""
	if len(lines) > 0 {
		body = strings.Join(lines, "\n") + "\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// divCase is one comparison scenario.
type divCase struct {
	name string
	file []string // file contents; empty => empty buffer started with no file arg
	keys string   // bytes sent after both editors are ready
	desc string   // what the sequence is meant to do
}

// bodyOf returns the screen rows above the status line.
func bodyOf(d []string) []string {
	if len(d) == 0 {
		return d
	}
	return d[:len(d)-1]
}

// runDivCase starts both editors on the case's file, sends its keys, and reports
// whether the body text or cursor diverged, logging both screens when they do.
// It returns true on divergence.
func runDivCase(t *testing.T, rows, cols int, c divCase) bool {
	t.Helper()
	for _, p := range []string{nviPath, goviPath} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("%s not available", p)
		}
	}

	startOne := func(path string) *Term {
		tm := New(rows, cols)
		var err error
		if len(c.file) > 0 {
			err = tm.Start(path, writeLines(t, c.file))
		} else {
			err = tm.Start(path)
		}
		if err != nil {
			t.Fatalf("Start %s: %v", path, err)
		}
		return tm
	}
	nvi := startOne(nviPath)
	govi := startOne(goviPath)
	pair := editorPair{A: nvi, B: govi}
	defer pair.close()

	// Ready when the status line (last row) is populated for a file load, or the
	// tilde column has appeared for an empty buffer.
	ready := func(d []string) bool {
		if len(c.file) > 0 {
			return strings.TrimSpace(d[len(d)-1]) != ""
		}
		return len(d) > 1 && d[1] == "~"
	}
	if !pair.waitReady(ready) {
		t.Fatalf("[%s] an editor did not become ready", c.name)
	}
	pair.settle()

	if c.keys != "" {
		pair.send([]byte(c.keys))
		pair.settle()
	}

	nb, gb := bodyOf(nvi.Dump()), bodyOf(govi.Dump())
	bodyDiffs := DiffScreens(nb, gb)
	nr, nc := nvi.Cursor()
	gr, gc := govi.Cursor()
	curMatch := nr == gr && nc == gc

	if len(bodyDiffs) == 0 && curMatch {
		t.Logf("[%-12s] match   (cursor %d,%d)  %s", c.name, nr, nc, c.desc)
		pair.send([]byte("\x1b:q!\r"))
		return false
	}

	t.Logf("[%-12s] DIVERGE  %s", c.name, c.desc)
	if !curMatch {
		t.Logf("    cursor: nvi=%d,%d govi=%d,%d", nr, nc, gr, gc)
	}
	if len(bodyDiffs) > 0 {
		t.Logf("    body (nvi | govi):\n%s", indent(FormatDiffs(bodyDiffs)))
	}
	pair.send([]byte("\x1b:q!\r"))
	return true
}

func indent(s string) string {
	if s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = "        " + lines[i]
	}
	return strings.Join(lines, "\n")
}

// sampleLines is a small mixed-content buffer for editing/motion tests: short
// and long words, punctuation, leading whitespace, and blank lines.
func sampleLines() []string {
	return []string{
		"alpha beta gamma",
		"the quick brown fox",
		"  indented line here",
		"punct: foo, bar; baz.",
		"",
		"last line of file",
	}
}

// TestDivergeEditing exercises single-keystroke and operator editing commands.
func TestDivergeEditing(t *testing.T) {
	cases := []divCase{
		{"x", sampleLines(), "x", "delete char under cursor"},
		{"3x", sampleLines(), "3x", "delete 3 chars"},
		{"X", sampleLines(), "llX", "delete char before cursor"},
		{"dd", sampleLines(), "dd", "delete line"},
		{"2dd", sampleLines(), "2dd", "delete 2 lines"},
		{"D", sampleLines(), "wD", "delete to end of line"},
		{"dw", sampleLines(), "dw", "delete word"},
		{"d$", sampleLines(), "lld$", "delete to EOL"},
		{"dG", sampleLines(), "jjdG", "delete to end of file"},
		{"J", sampleLines(), "J", "join lines"},
		{"r", sampleLines(), "rZ", "replace char"},
		{"~", sampleLines(), "~", "toggle case"},
		{"cw", sampleLines(), "cwXYZ\x1b", "change word"},
		{"yyp", sampleLines(), "yyp", "yank line and put"},
		{"ddp", sampleLines(), "ddp", "delete line and put below"},
	}
	mined := 0
	for _, c := range cases {
		if runDivCase(t, 12, 40, c) {
			mined++
		}
	}
	t.Logf("editing: %d/%d scenarios diverged", mined, len(cases))
}

// TestDivergeMotion exercises cursor motions; the buffer content is unchanged so
// only the cursor position can diverge.
func TestDivergeMotion(t *testing.T) {
	cases := []divCase{
		{"w", sampleLines(), "w", "word forward"},
		{"3w", sampleLines(), "3w", "3 words forward"},
		{"b", sampleLines(), "wwb", "word back"},
		{"e", sampleLines(), "e", "end of word"},
		{"0", sampleLines(), "ww0", "to column 0"},
		{"$", sampleLines(), "$", "to end of line"},
		{"^", sampleLines(), "jj^", "to first non-blank"},
		{"fx", sampleLines(), "fa", "find char forward"},
		{"tx", sampleLines(), "tg", "till char forward"},
		{"G", sampleLines(), "G", "to last line"},
		{"gg", sampleLines(), "Ggg", "gg to first line"},
		{"H", sampleLines(), "GH", "to top of screen"},
		{"L", sampleLines(), "L", "to bottom of screen"},
		{"M", sampleLines(), "M", "to middle of screen"},
		{"percent", sampleLines(), "50%", "to 50% of file"},
	}
	mined := 0
	for _, c := range cases {
		if runDivCase(t, 12, 40, c) {
			mined++
		}
	}
	t.Logf("motion: %d/%d scenarios diverged", mined, len(cases))
}

// TestDivergeSearch exercises search and the related n/N/*/# commands.
func TestDivergeSearch(t *testing.T) {
	cases := []divCase{
		{"slash", sampleLines(), "/fox\r", "search forward"},
		{"slash-n", sampleLines(), "/line\rn", "search then next"},
		{"slash-N", sampleLines(), "/line\rN", "search then prev (wraps)"},
		{"question", sampleLines(), "G?alpha\r", "search backward"},
		{"star", sampleLines(), "*", "search word under cursor"},
		{"no-match", sampleLines(), "/zzznope\r", "search with no match"},
	}
	mined := 0
	for _, c := range cases {
		if runDivCase(t, 12, 40, c) {
			mined++
		}
	}
	t.Logf("search: %d/%d scenarios diverged", mined, len(cases))
}

// TestDivergeEx exercises ex (colon) commands.
func TestDivergeEx(t *testing.T) {
	cases := []divCase{
		{"d", sampleLines(), ":2d\r", "delete line 2"},
		{"range-d", sampleLines(), ":2,3d\r", "delete lines 2-3"},
		{"subst", sampleLines(), ":s/alpha/ALPHA/\r", "substitute on line"},
		{"global-subst", sampleLines(), ":%s/line/LINE/g\r", "substitute all"},
		{"move", sampleLines(), ":1m3\r", "move line 1 after 3"},
		{"copy", sampleLines(), ":1t3\r", "copy line 1 after 3"},
		{"goto", sampleLines(), ":4\r", "go to line 4"},
		{"set-nu", sampleLines(), ":set number\r", "show line numbers"},
	}
	mined := 0
	for _, c := range cases {
		if runDivCase(t, 12, 40, c) {
			mined++
		}
	}
	t.Logf("ex: %d/%d scenarios diverged", mined, len(cases))
}

// numberedLines returns n lines "001".."NNN": distinct, sortable content so the
// scrolled-to position is unambiguous.
func numberedLines(n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmtLine(i + 1)
	}
	return lines
}

func fmtLine(i int) string {
	s := strconv.Itoa(i)
	for len(s) < 3 {
		s = "0" + s
	}
	return s
}

// TestDivergePaging exercises the scrolling/paging commands on a 60-line file at
// 12x40.  These move the viewport, so both the body (top visible line) and the
// cursor matter.  ^F is the original known divergence.
func TestDivergePaging(t *testing.T) {
	f := numberedLines(60)
	cases := []divCase{
		{"^F", f, "\x06", "forward one screen"},
		{"^B", f, "G\x02", "back one screen (from end)"},
		{"^D", f, "\x04", "down half screen"},
		{"^U", f, "G\x15", "up half screen (from end)"},
		{"^E", f, "\x05", "scroll down one line"},
		{"^Y", f, "5G\x19", "scroll up one line"},
		{"^F^F", f, "\x06\x06", "forward two screens"},
		{"goto-off", f, "20G", "goto line off-screen (viewport placement)"},
	}
	mined := 0
	for _, c := range cases {
		if runDivCase(t, 12, 40, c) {
			mined++
		}
	}
	t.Logf("paging: %d/%d scenarios diverged", mined, len(cases))
}

// TestDivergeRegisters exercises marks, named registers, dot-repeat, and macros.
func TestDivergeRegisters(t *testing.T) {
	cases := []divCase{
		{"mark", sampleLines(), "majj`a", "set mark a, move, jump back"},
		{"mark-line", sampleLines(), "majj'a", "mark then line-jump back"},
		{"named-yank", sampleLines(), "\"ayyj\"ap", "yank to reg a, put reg a"},
		{"dot-x", sampleLines(), "x..", "delete char, repeat twice"},
		{"dot-dd", sampleLines(), "dd.", "delete line, repeat"},
		{"dot-cw", sampleLines(), "cwX\x1bw.", "change word, repeat on next"},
		{"macro", sampleLines(), "qaxjq@a", "record macro, replay"},
		{"append-reg", sampleLines(), "\"ayyj\"Ayygg\"ap", "append to register"},
	}
	mined := 0
	for _, c := range cases {
		if runDivCase(t, 12, 40, c) {
			mined++
		}
	}
	t.Logf("registers: %d/%d scenarios diverged", mined, len(cases))
}

// TestDivergeStructure exercises shift operators and bracket matching.
func TestDivergeStructure(t *testing.T) {
	brackets := []string{
		"func f() {",
		"    if (a) {",
		"        x = 1",
		"    }",
		"}",
	}
	cases := []divCase{
		{"shift-right", sampleLines(), ">>", "shift line right"},
		{"shift-2", sampleLines(), "2>>", "shift 2 lines right"},
		{"shift-left", brackets, "j<<", "shift line left"},
		{"shift-G", sampleLines(), ">G", "shift to end of file"},
		{"match-paren", brackets, "f(%", "match to closing paren"},
		{"match-brace", brackets, "$%", "match to closing brace"},
		{"indent-keep", brackets, "jj==", "reindent line"},
	}
	mined := 0
	for _, c := range cases {
		if runDivCase(t, 12, 40, c) {
			mined++
		}
	}
	t.Logf("structure: %d/%d scenarios diverged", mined, len(cases))
}
