package goterm

import "testing"

// TestSpecialKeysTerminfo probes the terminal special keys the goterm
// terminfo entry advertises (kich1/kdch1/kpp/knp, xterm sequences). nvi
// seeds command maps from terminfo at startup (cl/cl_term.c: kich1->i,
// kdch1->x, kpp->^B, knp->^F); govi consumes the decoded keys the same way
// (CORNERS.md Part C #10, fixed 2026-07-04). All cases must match.
func TestSpecialKeysTerminfo(t *testing.T) {
	if !AltScreenSupported() {
		t.Skip("goterm terminfo entry unavailable (tic missing)")
	}
	cases := []divCase{
		{"key-delete", sampleLines(), "\x1b[3~",
			"Delete key deletes the character under the cursor"},
		{"key-insert", sampleLines(), "\x1b[2~zap\x1b",
			"Insert key enters insert mode (kich1 -> i)"},
		{"key-pagedown", numberedLines(60), "\x1b[6~x",
			"PageDown pages forward (knp -> ^F); x edits the paged-to line"},
		{"key-pageup", numberedLines(60), "G\x1b[5~x",
			"PageUp pages back from EOF (kpp -> ^B); x edits the paged-to line"},
	}
	div := 0
	for _, c := range cases {
		if runDivCase(t, 24, 80, c) {
			div++
		}
	}
	if div > 0 {
		t.Errorf("special-keys: %d/%d diverged (all must match since #10 was fixed)", div, len(cases))
	}
}
