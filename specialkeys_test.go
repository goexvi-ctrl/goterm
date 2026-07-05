package goterm

import "testing"

// TestSpecialKeysTerminfo probes the terminal special keys the goterm
// terminfo entry now advertises (kich1/kdch1/kpp/knp, xterm sequences).
// nvi seeds command maps from terminfo at startup (cl/cl_term.c: kich1->i,
// kdch1->x, kpp->^B, knp->^F); govi decodes the keys but the engine consumes
// only Delete (govi qa/CORNERS.md Part C #10), so the insert/page cases
// expect DIVERGE until that gap is closed -- they exist to make the gap
// visible to the harness at all (it previously advertised no such caps, so
// both editors were blind to the keys and every probe vacuously matched).
func TestSpecialKeysTerminfo(t *testing.T) {
	if !AltScreenSupported() {
		t.Skip("goterm terminfo entry unavailable (tic missing)")
	}
	cases := []divCase{
		{"key-delete", sampleLines(), "\x1b[3~",
			"Delete key deletes the character under the cursor in both"},
		{"key-insert", sampleLines(), "\x1b[2~zap\x1b",
			"expect DIVERGE: nvi maps Insert->i and inserts zap; govi drops the key (#10)"},
		{"key-pagedown", numberedLines(60), "\x1b[6~x",
			"expect DIVERGE: nvi maps PageDown->^F then x edits a paged-to line; govi drops it (#10)"},
		{"key-pageup", numberedLines(60), "G\x1b[5~x",
			"expect DIVERGE: nvi maps PageUp->^B from EOF; govi drops it (#10)"},
	}
	div := 0
	for _, c := range cases {
		if runDivCase(t, 24, 80, c) {
			div++
		}
	}
	if div == 0 {
		t.Log("special-keys: all matched -- #10 may have been fixed; update qa/CORNERS.md")
	}
	t.Logf("special-keys: %d/%d diverged", div, len(cases))
}
