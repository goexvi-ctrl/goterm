package goterm

import "testing"

// exmine_test.go is the standing battery for the 2026-06-28 ex-mode-focused mining
// wave (catalog GOTERM_DIVERGENCES.md, findings #17-33). Each OPEN case reports
// DIVERGE now and should flip to `match` when govi is fixed; the matched cases are
// regression guards for ex behavior that already agrees with nvi. These use the
// burst divCase model (runDivCase); cursor-only/display/stepwise findings (#25,
// #28, #29, #33) live only in the catalog -- see the notes there.

// exMineLines: foo/hello/aaa content with repeats and varied case for :s and :g.
func exMineLines() []string {
	return []string{
		"foo bar foo baz",
		"hello WORLD hello",
		"aaa bbb aaa ccc",
		"one two three four",
		"  leading space line",
	}
}

// colorLines: a clear matching predicate (lines with "red") for :g / :v.
func colorLines() []string {
	return []string{
		"apple red", "banana yellow", "cherry red",
		"date brown", "elderberry blue", "fig green",
	}
}

// TestDivergeExMine exercises ex-command divergences found in the ex-mode wave.
func TestDivergeExMine(t *testing.T) {
	el := exMineLines()
	cl := colorLines()
	cases := []divCase{
		// --- regression guards: these already MATCH nvi ---
		{"ex-d-range", el, ":2,4d\r", "ranged delete (guard)"},
		{"ex-v-del", cl, ":v/red/d\r", "vglobal delete non-matching (guard)"},
		{"ex-g-subst", cl, ":g/red/s//RED/\r", "global subst reuse (guard)"},
		{"ex-addr-search", cl, ":/cherry/d\r", "search address (guard)"},
		{"ex-amp", el, ":s/o/0/\rj:&\r", ":& repeat subst (guard)"},
		// --- OPEN divergences (#17-27) ---
		{"ex-print", el, ":1,3p\r", "#17 :p prints nothing in govi"},
		{"ex-s-count", el, ":1s/o/0/g 3\r", "#18 :s trailing count ignored"},
		{"ex-s-bare", el, ":s/foo/X/\r:s\r", "#19 bare :s does not repeat"},
		{"ex-s-tilde", el, ":s/o/0/\rj:~\r", "#20 :~ repeat-subst is a no-op"},
		{"ex-s-nflag", el, ":%s/foo//n\r", "#21 unsupported :s flag substitutes anyway"},
		{"ex-shift-stack", el, ":1>>\r", "#22 :>> stacked shift only one level"},
		{"ex-mark-k", el, ":2k a\rG:'a,$d\r", "#23 ex :k/:mark does not set a mark"},
		{"ex-join-cursor", el, ":1,2j\r", "#24 :j leaves cursor at col 0"},
		{"ex-g-bang", cl, ":g!/red/d\r", "#26 :g! deletes matching not non-matching"},
		{"ex-g-copy", cl, ":g/red/t$\r", "#27 :g mistracks line numbers when inserting"},
	}
	mined := 0
	for _, c := range cases {
		if runDivCase(t, 12, 40, c) {
			mined++
		}
	}
	t.Logf("ex-mine: %d/%d scenarios diverged (10 of these are open findings #17-27)", mined, len(cases))
}

// sentenceBlankLines: two text lines separated by a blank, for sentence-motion
// boundary tests (#32).
func sentenceBlankLines() []string {
	return []string{"One sentence.  Two sentence.  Three here.", "", "Next para."}
}

// TestDivergeViMine exercises vi-mode divergences found in the same wave.
func TestDivergeViMine(t *testing.T) {
	el := exMineLines()
	cases := []divCase{
		// --- regression guards ---
		{"search-plain", el, "/aaa\r", "plain search (guard)"},
		{"sent-inline", sentenceBlankLines(), "0)", "in-line two-space sentence fwd (guard)"},
		// --- OPEN divergences ---
		{"search-offset", el, "/aaa/+1\r", "#30 search line offset is a no-op"},
		{"search-chain", el, "/foo/;/bar/\r", "#30 search ; chain is a no-op"},
		{"insert-ctrl", []string{"x"}, "i\x01\x1b", "#31 insert swallows unhandled ctrl char"},
		{"sentence-blank", sentenceBlankLines(), "3G(", "#32 ( skips a blank-line boundary"},
		// #34 was a bogus finding: nvi `#` is the INCREMENT command (#+/#-), which
		// govi implements and matches; bare `#` is an incomplete prefix. Guards:
		{"incr-plus", []string{"count 5 here"}, "f5#+", "#34 nvi # increment (#+); govi matches"},
		{"incr-minus", []string{"count 5 here"}, "f5#-", "#34 increment #- guard"},
		// #35 (operator + /pat search motion) is now FIXED; kept as a guard.
		{"op-search", el, "d/baz\r", "#35 d/pat search-motion (was buffer corruption; fixed)"},
	}
	mined := 0
	for _, c := range cases {
		if runDivCase(t, 12, 40, c) {
			mined++
		}
	}
	t.Logf("vi-mine: %d/%d scenarios diverged (open: only #32 remains; #30/#31/#35 fixed, #34 bogus)", mined, len(cases))
}

// TestDivergeRoster covers #37: ex commands nvi lists in :exusage that are no-ops
// in govi (found by driving the whole usage roster). Guards confirm the adjacent
// commands that DO work, so a fix is attributable.
func TestDivergeRoster(t *testing.T) {
	el := exMineLines()
	exec := []string{"d", "alpha beta", "second line", "third line"}
	cases := []divCase{
		// guards: these ex commands already match nvi.
		{"ex-nu-guard", el, ":1,3nu\r", "guard: :number prints (works)"},
		{"ex-p-guard", el, ":1,3p\r", "guard: :print works"},
		// #37 FIXED 2026-06-28 -- now regression guards (were no-ops in govi).
		{"ex-at-exec", exec, "\"ayyj:@a\r", "#37 FIXED: :@ executes buffer as ex commands"},
		{"ex-undo", el, "dd:undo\r", "#37 FIXED: :undo undoes last change"},
		{"ex-hash-num", el, ":1,3#\r", "#37 FIXED: :# is a synonym for :number"},
		// #37 still open (deferred): informational display tangled with message pagination.
		{"ex-z", numberedLines(40), ":10z\r", "#37 :z screenful display unimplemented"},
		{"ex-display", el, "\"ayy:display buffers\r", "#54 :display buffers format differs (implemented; nvi lists the default buffer + mode headers)"},
	}
	mined := 0
	for _, c := range cases {
		if runDivCase(t, 12, 40, c) {
			mined++
		}
	}
	t.Logf("roster: %d/%d scenarios diverged (open: only #37 :z/:display remain)", mined, len(cases))
}
