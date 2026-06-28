package goterm

import (
	"os"
	"strings"
	"testing"
	"time"
)

// This file does AGGRESSIVE, multi-step comparison of govi against nvi: realistic
// sessions that mix ex (:) commands, vi commands, external filters (!), writes,
// and undo/redo.  Unlike divergence_test.go (one write, one settle per case),
// each step here is sent as its OWN write with a settle between, so commands that
// block -- file writes, external filter programs, anything that redraws in
// stages -- complete before the next input.  The motivating bug: open a file,
// edit, :w, then `!Ggoimports`, then `u` -> govi reported undo would not work.

// seqCase is a multi-step scenario.  steps are sent one at a time, settling
// between, and the final body + cursor are compared (the status line, which
// carries differing info/error wording, is excluded like in runDivCase).
type seqCase struct {
	name  string
	file  []string
	steps []string
	desc  string
}

// runSeqCase starts both editors on the file, plays the steps with a settle
// between each (long enough for external filters), and reports divergence of the
// final body + cursor.  It logs both final status lines too, since undo/filter
// errors surface there.  Returns true on divergence.
func runSeqCase(t *testing.T, rows, cols int, c seqCase) bool {
	t.Helper()
	for _, p := range []string{nviPath, goviPath} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("%s not available", p)
		}
	}

	start := func(path string) *Term {
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
	nvi := start(nviPath)
	govi := start(goviPath)
	pair := editorPair{A: nvi, B: govi}
	defer pair.close()

	ready := func(d []string) bool {
		if len(c.file) > 0 {
			return strings.TrimSpace(d[len(d)-1]) != ""
		}
		return len(d) > 1 && d[1] == "~"
	}
	if !pair.waitReady(ready) {
		t.Fatalf("[%s] an editor did not become ready", c.name)
	}
	settle := func() {
		nvi.WaitQuiet(250*time.Millisecond, 5*time.Second)
		govi.WaitQuiet(250*time.Millisecond, 5*time.Second)
	}
	settle()

	for i, s := range c.steps {
		pair.send([]byte(s))
		settle()
		// Per-step body divergence is useful for locating WHERE a sequence breaks.
		if d := DiffScreens(bodyOf(nvi.Dump()), bodyOf(govi.Dump())); len(d) > 0 {
			t.Logf("[%-16s] step %d (%q) already diverges in body", c.name, i, s)
		}
	}

	nb, gb := bodyOf(nvi.Dump()), bodyOf(govi.Dump())
	bodyDiffs := DiffScreens(nb, gb)
	nr, nc := nvi.Cursor()
	gr, gc := govi.Cursor()
	curMatch := nr == gr && nc == gc

	if len(bodyDiffs) == 0 && curMatch {
		t.Logf("[%-16s] match   (cursor %d,%d)  %s", c.name, nr, nc, c.desc)
		pair.send([]byte("\x1b:q!\r"))
		return false
	}

	t.Logf("[%-16s] DIVERGE  %s", c.name, c.desc)
	if !curMatch {
		t.Logf("    cursor: nvi=%d,%d govi=%d,%d", nr, nc, gr, gc)
	}
	if len(bodyDiffs) > 0 {
		t.Logf("    body (nvi | govi):\n%s", indent(FormatDiffs(bodyDiffs)))
	}
	t.Logf("    status nvi:  %q", nvi.Dump()[len(nvi.Dump())-1])
	t.Logf("    status govi: %q", govi.Dump()[len(govi.Dump())-1])
	pair.send([]byte("\x1b:q!\r"))
	return true
}

// goSnippet is a tiny Go file that uses fmt without importing it, so goimports
// (or gofmt) visibly rewrites the buffer when run as a filter.
func goSnippet() []string {
	return []string{
		"package main",
		"",
		"func main() {",
		"\tfmt.Println(\"hello\")",
		"}",
	}
}

// TestDivergeSequences plays realistic multi-command sessions.
func TestDivergeSequences(t *testing.T) {
	cases := []seqCase{
		// The reported bug class: edit, write, external filter, then undo.
		{"write-filter-undo", goSnippet(),
			[]string{"ggO// header\x1b", ":w\r", "1G!Ggoimports\r", "u"},
			"edit, :w, filter whole file through goimports, then undo"},
		// Same shape with a filter that is always present.
		{"write-sort-undo", sampleLines(),
			[]string{"ggO// header\x1b", ":w\r", "1G!Gsort\r", "u"},
			"edit, :w, filter through sort, then undo"},
		// Undo across a write: nvi can undo edits made before :w.
		{"edit-write-edit-undo", sampleLines(),
			[]string{"ihello \x1b", ":w\r", "Aworld\x1b", "u", "u"},
			"insert, write, append, undo twice (across the write)"},
		// Filter current line only, then undo, then redo.
		{"filter-line-undo-redo", sampleLines(),
			[]string{"!!tr a-z A-Z\r", "u", "\x12"},
			"uppercase line via tr filter, undo, redo (^R)"},
		// ex goto + vi delete + undo + ex range put.
		{"ex-vi-mix", sampleLines(),
			[]string{":3\r", "dd", "u", ":1\r", "yy", ":4\r", "p"},
			"ex goto, dd, undo, ex goto, yank, ex goto, put"},
		// Mark, ex range delete using the mark, then undo.
		{"mark-range-undo", sampleLines(),
			[]string{"jma", "G", ":'a,$d\r", "u"},
			"set mark a, ex-delete 'a,$ then undo"},
		// Counted op + dot repeat + multiple undo.
		{"count-dot-undo", sampleLines(),
			[]string{"2dd", "2.", "u", "u"},
			"delete 2 lines, repeat, undo twice"},
		// Search, change, repeat-with-n, then undo the lot.
		{"search-change-undo", sampleLines(),
			[]string{"/beta\r", "cwBETA\x1b", "u"},
			"search, change word, undo"},
		// Global substitute then undo (ex command undo).
		{"global-subst-undo", sampleLines(),
			[]string{":%s/a/A/g\r", "u"},
			"substitute all then undo"},
		// Repeated joins then repeated undo back to original.
		{"join-undo", sampleLines(),
			[]string{"J", "J", "u", "u"},
			"join twice, undo twice"},
	}
	div := 0
	for _, c := range cases {
		if runSeqCase(t, 24, 80, c) {
			div++
		}
	}
	t.Logf("sequences: %d/%d diverged", div, len(cases))
}

// bracedLines is a small file with brackets and varied words for %, d%, text
// motions, and global/substitute probing.
func bracedLines() []string {
	return []string{
		"func main() {",
		"\tx := (1 + 2)",
		"\ty := [3, 4]",
		"\treturn",
		"}",
		"alpha alpha alpha",
	}
}

// TestDivergeAdvanced hunts for NEW divergences in the un-mined territory:
// counted insert + dot, replace mode, autoindent, global (:g) commands, numbered
// registers, macros that contain ex commands, & substitute-repeat, bracket
// delete, and layout-affecting :set.  These are the sequences unit tests miss.
func TestDivergeAdvanced(t *testing.T) {
	rc := func(rows, cols int, c seqCase) bool { return runSeqCase(t, rows, cols, c) }
	type adv struct {
		rows, cols int
		c          seqCase
	}
	cases := []adv{
		// Counted insert, then dot-repeat it.
		{24, 80, seqCase{"count-insert-dot", sampleLines(),
			[]string{"3ix\x1b", "0", "."}, "3ix<Esc> then . repeats the triple insert"}},
		// Replace (overtype) mode then undo.
		{24, 80, seqCase{"replace-mode-undo", sampleLines(),
			[]string{"RWXYZ\x1b", "u"}, "R overtype then undo"}},
		// Replace a char with a newline (should SPLIT the line).  No trailing undo:
		// undo would restore both to the original and hide the divergence.
		{24, 80, seqCase{"r-newline", sampleLines(),
			[]string{"wr\r"}, "replace 'b' of beta with a newline -> split into two lines"}},
		// Autoindent: open lines under an indented line, indent should carry.
		{24, 80, seqCase{"autoindent-open", bracedLines(),
			[]string{":set ai\r", "joINDENTED\x1bofoo\x1b"}, "set ai, open two lines under tab-indented line"}},
		// Global delete every line containing a vowel pattern.
		{24, 80, seqCase{"global-delete", sampleLines(),
			[]string{":g/a/d\r", "u"}, "delete all lines matching /a/, then undo"}},
		// Global substitute via :g .. s.
		{24, 80, seqCase{"global-subst", sampleLines(),
			[]string{":g/e/s//E/\r", "u"}, "on lines matching /e/, substitute e->E, undo"}},
		// Numbered registers after successive deletes.
		{24, 80, seqCase{"numbered-regs", sampleLines(),
			[]string{"dd", "dd", "G", "\"1p", "\"2p"}, "two dd then put numbered regs 1 and 2"}},
		// Transpose two characters with xp; repeat.
		{24, 80, seqCase{"xp-transpose", sampleLines(),
			[]string{"xp", "."}, "xp transpose then repeat"}},
		// Counted join then undo each.
		{24, 80, seqCase{"counted-join", sampleLines(),
			[]string{"3J", "u"}, "3J join three lines, undo"}},
		// f then ; and , to repeat the find both ways.
		{24, 80, seqCase{"find-repeat", sampleLines(),
			[]string{"fa", ";", ";", ","}, "find a, repeat forward twice, reverse once"}},
		// Delete to matching bracket.
		{24, 80, seqCase{"delete-to-paren", bracedLines(),
			[]string{"f(", "d%", "u"}, "delete from ( to matching ), undo"}},
		// Macro that contains an ex command, replayed with a count.
		{24, 80, seqCase{"macro-ex", sampleLines(),
			[]string{"qa:s/a/Z/\rjq", "2@a"}, "record macro doing :s and j, replay twice"}},
		// & repeats the last substitute on the current line.
		{24, 80, seqCase{"amp-subst-repeat", sampleLines(),
			[]string{":s/a/Z/\r", "j", "&"}, "substitute, move down, & repeats it"}},
		// dgg: delete from current line to top of file.
		{24, 80, seqCase{"dgg", sampleLines(),
			[]string{"G", "dgg", "u"}, "delete to top of file, undo"}},
		// ~ with a count toggles several characters.
		{24, 80, seqCase{"count-tilde", sampleLines(),
			[]string{"5~", "u"}, "toggle case of 5 chars, undo"}},
		// Layout: wrapmargin should wrap a long inserted line (narrow screen).
		{8, 30, seqCase{"wrapmargin", []string{""},
			[]string{":set wm=5\r", "ithe quick brown fox jumped over the lazy dog\x1b"},
			"set wrapmargin on a narrow screen, insert a long line"}},
		// Layout: list mode shows $ at EOL and tabs as ^I.
		{12, 40, seqCase{"list-mode", bracedLines(),
			[]string{":set list\r"}, "set list: tabs and line-ends rendered"}},
		// Open many lines past the screen with autoindent, forcing scroll + indent.
		{12, 40, seqCase{"ai-scroll", bracedLines(),
			[]string{":set ai\r", "Goa\rb\rc\rd\re\rf\rg\rh\ri\rj\rk\x1b"},
			"autoindent open many lines, scrolling the viewport"}},
	}
	div := 0
	for _, a := range cases {
		if rc(a.rows, a.cols, a.c) {
			div++
		}
	}
	t.Logf("advanced: %d/%d diverged", div, len(cases))
}

// sentenceLines: multiple sentences per line and across lines, for ) and (
// motions (sentence = end punctuation . ! ? then space/EOL).
func sentenceLines() []string {
	return []string{
		"One fish. Two fish! Red fish? Blue fish.",
		"Next line starts here. And ends here.",
	}
}

// paragraphLines: blank-line-separated paragraphs for } and { motions.
func paragraphLines() []string {
	return []string{
		"para one line a",
		"para one line b",
		"",
		"para two line a",
		"para two line b",
		"",
		"para three only",
	}
}

// TestDivergeMore mines more classic-vi territory nvi actually supports:
// sentence/paragraph motions, ex search-address ranges, insert-mode editing keys
// (^W ^U ^T ^D), replace-mode backspace restore, shiftwidth/tabstop, WORD
// motions, and counted puts.  (Visual mode v/V is intentionally NOT here: nvi
// rejects it -- "v isn't a vi command" -- so there is no reference behavior.)
func TestDivergeMore(t *testing.T) {
	type adv struct {
		rows, cols int
		c          seqCase
	}
	cases := []adv{
		// Sentence motions and an operator over one.
		{24, 80, seqCase{"sentence-fwd", sentenceLines(),
			[]string{")", ")"}, "two sentences forward"}},
		{24, 80, seqCase{"sentence-back", sentenceLines(),
			[]string{"$", "(", "("}, "from EOL, two sentences back"}},
		{24, 80, seqCase{"delete-sentence", sentenceLines(),
			[]string{"d)", "u"}, "delete to next sentence, undo"}},
		// Paragraph motions and an operator over one.
		{24, 80, seqCase{"para-fwd", paragraphLines(),
			[]string{"}", "}"}, "two paragraphs forward"}},
		{24, 80, seqCase{"para-back", paragraphLines(),
			[]string{"G", "{", "{"}, "from end, two paragraphs back"}},
		{24, 80, seqCase{"delete-para", paragraphLines(),
			[]string{"d}", "u"}, "delete to next paragraph, undo"}},
		// ex search-address ranges.  No trailing undo: undo restores both to the
		// identical original and hides that govi's search-address delete is a no-op.
		{24, 80, seqCase{"ex-search-addr", sampleLines(),
			[]string{":/quick/d\r"}, "ex delete the line matching /quick/"}},
		{24, 80, seqCase{"ex-search-range", sampleLines(),
			[]string{":/alpha/,/punct/d\r"}, "ex delete /alpha/,/punct/ range"}},
		{24, 80, seqCase{"ex-rel-search", sampleLines(),
			[]string{":.,/fox/d\r"}, "ex delete from current line to /fox/"}},
		// Insert-mode editing keys.
		{24, 80, seqCase{"insert-ctrlw", sampleLines(),
			[]string{"A more words\x17\x17\x1b"}, "append, then ^W ^W delete two words"}},
		{24, 80, seqCase{"insert-ctrlu", sampleLines(),
			[]string{"ostuff here\x15\x1b"}, "open line, type, ^U erases the inserted text"}},
		// Autoindent with ^T (indent) in insert mode.  ^T works; ^D (dedent) is
		// left out -- nvi's mid-line ^D behavior is arcane (it can store a literal
		// ^D), so it needs a dedicated autoindent-context test, not this battery.
		{12, 40, seqCase{"insert-ctrlt", sampleLines(),
			[]string{":set ai sw=4\r", "o\x14foo\x1b"},
			"set ai, open line, ^T indent then text"}},
		// Replace mode then backspace restores the original characters.
		{24, 80, seqCase{"replace-backspace", sampleLines(),
			[]string{"RXXXX\x7f\x7f\x1b"}, "R overtype 4, backspace 2 (should restore originals)"}},
		// Shiftwidth affects >>.
		{24, 80, seqCase{"shiftwidth", sampleLines(),
			[]string{":set sw=3\r", ">>", "u"}, "set shiftwidth=3, shift line, undo"}},
		// Tabstop affects rendering of a tab-indented file.
		{12, 40, seqCase{"tabstop", bracedLines(),
			[]string{":set ts=4\r"}, "set tabstop=4: tab-indented lines render narrower"}},
		// WORD (whitespace-delimited) motions on punctuation.
		{24, 80, seqCase{"WORD-motion", []string{"foo.bar baz, qux; end"},
			[]string{"W", "W", "E", "B"}, "W W E B over punctuation"}},
		// Counted put.
		{24, 80, seqCase{"counted-put", sampleLines(),
			[]string{"yy", "3p", "u"}, "yank line, put 3 times, undo"}},
		// Put a multi-line yank.
		{24, 80, seqCase{"multiline-put", sampleLines(),
			[]string{"2yy", "G", "p"}, "yank 2 lines, put at end"}},
	}
	div := 0
	for _, a := range cases {
		if runSeqCase(t, a.rows, a.cols, a.c) {
			div++
		}
	}
	t.Logf("more: %d/%d diverged", div, len(cases))
}
