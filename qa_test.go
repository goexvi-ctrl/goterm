package goterm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// qa_test.go is the 2026-07-03 final-QA battery: probes for situations no
// earlier battery exercised, weighted toward compound operations, wrapping,
// and edge cases.  Like the other mining batteries it is a REPORT: cases log
// `match` or `DIVERGE` and do not fail the run.  Findings are adjudicated in
// /Users/claude/src/nvi/govi/qa/REPORT.md.

// qaLines is the shared compound-operation fixture.
func qaLines() []string {
	return []string{
		"alpha beta gamma delta epsilon",
		"one two three four five six",
		"foo(bar, baz) qux quux",
		"  indented code line",
		"pattern here and pattern there",
		"tail end line",
	}
}

// TestQACompound probes compound operations: multiplied counts, counted
// finds as operator targets, marks and searches as motion targets, dot
// repeat with count overrides, dot after macros, counted inserts, and the
// numbered-register auto-increment on dot after a "1p.
func TestQACompound(t *testing.T) {
	f := qaLines()
	cases := []divCase{
		{"2d3w", f, "2d3w", "counts multiply: 2d3w deletes 6 words"},
		{"d2fa", f, "d2fa", "counted find as target: delete through 2nd a"},
		{"c2w", f, "c2wX \x1b", "change 2 words"},
		{"2cc", f, "2ccX\x1b", "counted line change replaces 2 lines"},
		{"3J", f, "3J", "join 3 lines"},
		{"J-eof-clamp", f, "5G9J", "J count past EOF"},
		{"J-space-rules", []string{"foo ", "   bar"}, "J", "join keeps trailing blank, eats leading"},
		{"d-mark-line", f, "3Gma1Gd'a", "delete to mark, linewise"},
		{"d-mark-char", f, "lmaww" + "d`a", "delete back to mark, charwise"},
		{"d-search-n", f, "/pattern\r" + "1Gdn", "delete with n (search repeat) as target"},
		{"d-semi", f, "fa" + "d;", "delete with ; (find repeat) as target"},
		{"d-comma", f, "3fa" + "d,", "delete with , (reversed find) as target"},
		{"dot-count-override", f, "dw3.", "dw then 3. deletes 3 words"},
		{"dot-2dd", f, "2dd.", "2dd then . repeats the 2-line delete"},
		{"dot-after-macro", f, "qaxxq@a.", ". after @a repeats the change inside the macro"},
		{"at-at", f, "qaxjq@a@@", "@@ repeats the last @ buffer"},
		{"numbered-put-dot", f, "dddd\"1p.", "\"1p then .: historic auto-increment to \"2"},
		{"3ix", f, "3ix\x1b", "counted insert repeats the text"},
		{"3A", f, "3A!\x1b", "counted append"},
		{"2o", f, "2onew\x1b", "counted open below"},
		{"2O", f, "2Oup\x1b", "counted open above"},
		{"3s", f, "3sXY\x1b", "counted s replaces 3 chars with the text"},
		{"2R", f, "2Rab\x1b", "counted replace mode"},
		{"3r-cr", f, "ll3r\r", "counted r<CR>: 3 chars become newline(s)"},
		{"x3p", f, "x3p", "charwise put with count"},
		{"reg-count-put", f, "\"ayyG\"a2p", "named-register put with count"},
		{"3shift", f, "3>>", "counted shift right"},
		{"shift-dot", f, ">>j.", ". repeats the shift on another line"},
		{"cw-esc-only", f, "cw\x1b", "cw aborted by ESC with no text"},
		{"2D", f, "w2D", "counted D"},
		{"u-dot", f, "ddu.", ". after u repeats the dd"},
		{"10tilde", []string{"ab"}, "10~", "counted ~ stops at EOL"},
		{"d0", f, "$d0", "delete to line start"},
		{"df-missing", f, "dfz", "failed find target: no change"},
		{"cw-dot-count", f, "cwX\x1bw2.", "dot with new count after cw"},
		{"d2j", f, "d2j", "operator with counted line motion"},
	}
	div := 0
	for _, c := range cases {
		if runDivCase(t, 24, 80, c) {
			div++
		}
	}
	t.Logf("qa-compound: %d/%d diverged", div, len(cases))
}

// TestQAWrap probes long-line behavior on a small (12x40) screen: motions
// and edits on soft-wrapped lines, paging with wraps in the window, lines at
// exactly the screen width, tabs in wrapped lines, and the number option
// changing the effective wrap width.
func TestQAWrap(t *testing.T) {
	long := "alpha beta gamma delta epsilon zeta eta theta iota kappa"
	wf := []string{long, "second short", "third line here", long, "fifth",
		"sixth", "seventh", "eighth", "ninth", "tenth", "eleventh", "twelfth"}
	exact := strings.Repeat("z", 40)
	cases := []divCase{
		{"j-over-wrap", wf, "j", "j crosses a wrapped line by logical line"},
		{"k-back-wrap", wf, "3Gkk", "k back across a wrapped line"},
		{"dollar-wrap", wf, "$", "$ lands on the wrapped tail row"},
		{"0-from-wrap", wf, "$0", "0 returns to the first screen row"},
		{"bar-wrap", wf, "50|", "| motion into the wrapped region"},
		{"x-on-wrap", wf, "45|x", "x inside the wrapped tail"},
		{"dd-wrapped", wf, "dd", "delete a wrapped line redraws correctly"},
		{"J-creates-wrap", []string{"first part of line", "second part joined on"},
			"J", "join produces a line that wraps"},
		{"H-wraps", wf, "GH", "H with wrapped lines on screen"},
		{"M-wraps", wf, "M", "M with wrapped lines on screen"},
		{"L-wraps", wf, "L", "L with wrapped lines on screen"},
		{"G-to-wrapped", wf, "4G", "goto a wrapped line"},
		{"ctrl-f-wraps", wf, "\x06", "^F pages over wrapped lines"},
		{"ctrl-b-wraps", wf, "G\x02", "^B pages back over wrapped lines"},
		{"insert-grow-wrap", wf, "A extra words pushing past forty columns\x1b",
			"append grows the line across the wrap"},
		{"exact-width", []string{exact, "next line"}, "$",
			"line exactly at screen width (deferred wrap)"},
		{"exact-width-j", []string{exact, "next line"}, "j",
			"j over an exactly-full line"},
		{"tab-wrap", []string{"head\tmiddle\ttail\tmore\tend", "plain"}, "$",
			"tabs in a wrapped line"},
		{"number-wrap", wf, ":set nu\rj$", "number option shifts the wrap column"},
		{"col-memory-wrap", wf, "$jj", "column memory across wrapped lines"},
		{"wm-insert", []string{"start"}, ":set wm=10\roThe quick brown fox jumps over the lazy dog\x1b",
			"wrapmargin auto-breaks during insert"},
		{"wm-off-guard", []string{"start"}, "oThe quick brown fox jumps over the lazy dog\x1b",
			"guard: without wm the inserted line stays one line"},
		{"wraplen-insert", []string{"start"}, ":set wl=20\roThe quick brown fox jumps over the lazy dog\x1b",
			"wraplen auto-breaks at column 20"},
		{"ws-wrap-fwd", []string{"needle here", "middle", "last line"}, "G/needle\rx",
			"wrapscan: / wraps past EOF to line 1"},
		{"ws-wrap-back", []string{"first line", "middle", "needle here"}, "?needle\rx",
			"wrapscan: ? wraps past BOF to the last line"},
	}
	div := 0
	for _, c := range cases {
		if runDivCase(t, 12, 40, c) {
			div++
		}
	}
	t.Logf("qa-wrap: %d/%d diverged", div, len(cases))
}

// TestQAEdge probes boundary conditions: the empty buffer, one-character
// files, counts far past EOF, marks on deleted lines, undo with nothing to
// undo, and literal control-character input.
func TestQAEdge(t *testing.T) {
	var empty []string
	one := []string{"x"}
	five := []string{"line one", "line two", "line three", "line four", "line five"}
	cases := []divCase{
		{"empty-x", empty, "x", "x in an empty buffer"},
		{"empty-dd", empty, "dd", "dd in an empty buffer"},
		{"empty-p", empty, "p", "p with nothing yanked"},
		{"empty-J", empty, "J", "J in an empty buffer"},
		{"empty-dollar", empty, "$", "$ in an empty buffer"},
		{"empty-G", empty, "G", "G in an empty buffer"},
		{"empty-pct", empty, "%", "% with no match target"},
		{"empty-o", empty, "ozz\x1b", "o in an empty buffer"},
		{"empty-search", empty, "/x\r", "search in an empty buffer"},
		{"one-char-x", one, "x", "delete the only character"},
		{"one-char-xx", one, "xx", "x again on the now-empty line"},
		{"one-line-ddu", one, "ddu", "dd then undo on a 1-line file"},
		{"100G", five, "100G", "goto far past EOF"},
		{"100dd", five, "100dd", "dd count far past EOF"},
		{"100w", five, "100w", "w count far past EOF"},
		{"100x", []string{"abc"}, "100x", "x count past EOL"},
		{"caret-blank", []string{"", "   ", "abc"}, "j^", "^ on a blanks-only line"},
		{"mark-deleted", five, "3Gmagg" + "3dd'a", "jump to a mark whose line was deleted"},
		{"u-nochange", five, "u", "undo with no changes"},
		{"U-after-move", five, "xjU", "U after leaving the line"},
		{"ctrl-v-esc", []string{"ab"}, "i\x16\x1b\x1b", "^V<ESC> inserts a literal escape"},
		{"put-on-blank", []string{"", "abc"}, "jyyggp", "put a yanked line onto a blank line"},
		{"delete-all-insert", []string{"a", "b"}, "dGiX\x1b", "insert into the emptied buffer"},
		{"ctrl-e-bottom", five, "G\x05\x05\x05", "^E cannot scroll past EOF"},
		{"ctrl-y-top", five, "\x19", "^Y at the top of the file"},
		{"fx-missing", five, "fz;", "; repeat of a failed find"},
	}
	div := 0
	for _, c := range cases {
		if runDivCase(t, 24, 80, c) {
			div++
		}
	}
	t.Logf("qa-edge: %d/%d diverged", div, len(cases))
}

// TestQAExEdge probes ex-command corners: zero-width and special-replacement
// substitutions, empty-pattern reuse, address arithmetic (offsets, semicolon,
// backward search), :g with :m0, forced join, trailing counts, and :e
// variants.
func TestQAExEdge(t *testing.T) {
	el := []string{"foo bar foo baz", "second line here", "third here", "fourth line"}
	cases := []divCase{
		{"s-zero-width", []string{"abc"}, ":s/x*/-/g\r", "zero-width matches: -a-b-c-"},
		{"s-amp-repl", el, ":s/foo/[&]/\r", "& in the replacement"},
		{"s-tilde-repl", el, ":s/foo/X/\r:2s/line/~/\r", "~ replacement reuses the last replacement"},
		{"s-case-u", el, ":s/foo/\\u&/\r", "\\u upcases the next character"},
		{"s-case-U", el, ":s/foo bar/\\U&\\E!/\r", "\\U...\\E upcases a span"},
		{"s-backref", []string{"one two"}, ":s/\\(one\\) \\(two\\)/\\2 \\1/\r", "backreference swap"},
		{"s-empty-pat", el, "/foo\r:s//X/\r", ":s// reuses the search pattern"},
		{"slash-slash", el, "/foo\r//\r", "// repeats the last search"},
		{"amp-key", el, ":s/foo/X/\rj&", "& key repeats the substitution"},
		{"amp-addr", el, ":s/foo/X/\r:2&\r", ":& with an address"},
		{"put-0", el, "yy:0put\r", ":0put puts before line 1"},
		{"r-0-cmd", el, ":0r !echo TOP\r", ":0r !cmd inserts at the top"},
		{"addr-plus", el, ":.,+2d\r", "relative range .,+2"},
		{"addr-minus", el, ":$-1d\r", "address arithmetic $-1"},
		{"addr-semi", el, ":1;/here/d\r", "semicolon address: . moves before the search"},
		{"addr-search-off", el, ":/here/+1d\r", "search address with offset"},
		{"addr-back-search", el, "G:?foo?d\r", "backward search address"},
		{"g-move-0", el, ":g/here/m0\r", ":g with :m0 (renumber while iterating)"},
		{"v-subst", el, ":v/foo/s/e/E/\r", ":v with a substitution"},
		{"j-bang", []string{"aa", "bb"}, ":1,2j!\r", ":j! joins without inserting a space"},
		{"d-count-arg", el, ":2d 2\r", ":d with a trailing count"},
		{"e-plus5", nil, "", "placeholder"}, // replaced below by runArgs cases
	}
	cases = cases[:len(cases)-1]
	div := 0
	for _, c := range cases {
		if runDivCase(t, 24, 80, c) {
			div++
		}
	}

	// :s with the c (confirm) flag is interactive: drive it step by step.
	if runSeqCase(t, 24, 80, seqCase{"s-confirm", el,
		[]string{":%s/foo/X/c\r", "y", "n", "q"},
		":s///c confirm flow: y then n then quit"}) {
		div++
	}

	// :e variants need an on-disk fixture path in the keys.
	dir := shortDir(t)
	f1 := tmpFileAt(t, dir, "one.txt", "file one line A\nfile one line B\n")
	tgt := tmpFileAt(t, dir, "tgt.txt", "aaa\nbbb\nccc target line\nddd\neee\n")
	if runArgs(t, "e-plus-num", ":e +3 file starts on line 3",
		":e +3 "+tgt+"\r", 24, 80, f1) {
		div++
	}
	if runArgs(t, "e-plus-search", ":e +/target file starts on the match",
		":e +/target "+tgt+"\r", 24, 80, f1) {
		div++
	}
	// :e! reload discards changes; per-editor copies since a shared file would
	// hit the lock, and no write happens so copies are safe.
	if runSepCase(t, "e-bang-reload", ":e! reloads and discards the edit",
		"OJUNK\x1b:e!\r", 24, 80,
		[]prFixture{{"buf.txt", "kept line one\nkept line two\n"}}) {
		div++
	}
	t.Logf("qa-exedge: %d/%d diverged", div, len(cases)+4)
}

// qaDiskCase runs ONE editor on its own copy of a fixture, sends keys (which
// must end by exiting the editor), waits for exit, and returns the resulting
// bytes of the named files in the fixture dir.
func qaDiskRun(t *testing.T, bin string, fx []prFixture, keys func(dir string) string) map[string][]byte {
	t.Helper()
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("%s not available", bin)
	}
	dir := shortDir(t)
	var first string
	for i, f := range fx {
		p := tmpFileAt(t, dir, f.name, f.body)
		if i == 0 {
			first = p
		}
	}
	tm := New(24, 80)
	if err := tm.Start(bin, first); err != nil {
		t.Fatal(err)
	}
	defer tm.Close()
	tm.WaitFor(5*time.Second, func(d []string) bool {
		return strings.TrimSpace(d[len(d)-1]) != ""
	})
	tm.WaitQuiet(200*time.Millisecond, 2*time.Second)
	tm.Send([]byte(keys(dir)))
	// Editor should exit; give it a moment, then snapshot the directory.
	tm.WaitQuiet(300*time.Millisecond, 3*time.Second)
	out := map[string][]byte{}
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err == nil {
			out[e.Name()] = b
		}
	}
	return out
}

// runQADisk compares the on-disk results of the same session in both editors.
func runQADisk(t *testing.T, name, desc string, fx []prFixture, keys func(dir string) string) bool {
	t.Helper()
	nvi := qaDiskRun(t, nviPath, fx, keys)
	govi := qaDiskRun(t, goviPath, fx, keys)
	diverged := false
	seen := map[string]bool{}
	for n := range nvi {
		seen[n] = true
	}
	for n := range govi {
		seen[n] = true
	}
	for n := range seen {
		nb, nok := nvi[n]
		gb, gok := govi[n]
		if nok != gok {
			t.Logf("[%-12s] DIVERGE  %s: file %q nvi-present=%v govi-present=%v", name, desc, n, nok, gok)
			diverged = true
			continue
		}
		if string(nb) != string(gb) {
			t.Logf("[%-12s] DIVERGE  %s: file %q\n    nvi:  %q\n    govi: %q", name, desc, n, nb, gb)
			diverged = true
		}
	}
	if !diverged {
		t.Logf("[%-12s] match   %s (%d files identical)", name, desc, len(seen))
	}
	return diverged
}

// TestQADisk compares what actually lands on disk: write-back of a file
// without a trailing newline, :w >> append, writing a brand-new file, and
// buffer content written after control-character input.
func TestQADisk(t *testing.T) {
	div := 0
	if runQADisk(t, "no-final-nl", "write-back of a file lacking the final newline",
		[]prFixture{{"buf.txt", "abc\ndef"}},
		func(dir string) string { return "GAx\x1b:wq\r" }) {
		div++
	}
	if runQADisk(t, "w-append", ":w >> appends to another file",
		[]prFixture{{"buf.txt", "a\nb\nc\n"}, {"other.txt", "existing\n"}},
		func(dir string) string { return ":1w >> " + dir + "/other.txt\r:q!\r" }) {
		div++
	}
	if runQADisk(t, "w-newfile", ":w newfile writes the whole buffer",
		[]prFixture{{"buf.txt", "x\ny\n"}},
		func(dir string) string { return ":w " + dir + "/new.txt\r:q!\r" }) {
		div++
	}
	if runQADisk(t, "ctrl-char-write", "literal control chars survive a write",
		[]prFixture{{"buf.txt", "plain\n"}},
		func(dir string) string { return "o\x16\x07bell\x1b:wq\r" }) {
		div++
	}
	if runQADisk(t, "empty-buf-write", "writing an emptied buffer",
		[]prFixture{{"buf.txt", "a\nb\n"}},
		func(dir string) string { return "dG:wq\r" }) {
		div++
	}
	t.Logf("qa-disk: %d/5 diverged", div)
}
