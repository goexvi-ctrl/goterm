package goterm

import (
	"os"
	"strings"
	"testing"
)

// Probe battery for the cedit implementation (govi/qa/CEDIT.md P1-P14):
// nvi's colon command-line history and edit window, compared side by side
// against homebrew nvi 1.81.6, which supports the feature.
//
// Both editors open the SAME fixture file (short-pathed, EXINIT nolock, like
// runArgs) so the parent screen's split status row -- a compared body row --
// shows an identical path. The comedit window sits at the bottom, so its own
// modeline is the dropped last terminal row EXCEPT in the full-screen cases,
// where nvi shows its random temp-file path (an accepted divergence, see
// cedSkipLastRow).

// cedSET types :set cedit=<literal-next><escape> on the colon line.
const cedSET = ":set cedit=\x16\x1b\r"

// cedLines is the fixture buffer.
func cedLines() []string {
	return []string{
		"alpha beta gamma",
		"the quick brown fox",
		"last line of file",
	}
}

// runCedit compares body+cursor (the last terminal row dropped, as in
// runArgs). When full is true the whole screen including the last row is
// compared -- used for the in-window error messages, which land on the
// comedit window's status row.
func runCedit(t *testing.T, name, desc, keys string, full bool) bool {
	t.Helper()
	oldExinit, hadExinit := os.LookupEnv("EXINIT")
	os.Setenv("EXINIT", "set nolock")
	defer func() {
		if hadExinit {
			os.Setenv("EXINIT", oldExinit)
		} else {
			os.Unsetenv("EXINIT")
		}
	}()
	d := shortDir(t)
	f := tmpFileAt(t, d, "buf.txt", strings.Join(cedLines(), "\n")+"\n")
	nvi := startArgs(t, nviPath, 24, 80, f)
	govi := startArgs(t, goviPath, 24, 80, f)
	pair := editorPair{A: nvi, B: govi}
	defer pair.close()
	if !pair.waitReady(func(dmp []string) bool { return strings.TrimSpace(dmp[len(dmp)-1]) != "" }) {
		t.Fatalf("[%s] an editor did not become ready", name)
	}
	pair.settle()
	pair.send([]byte(keys))
	pair.settle()
	nd, gd := nvi.Dump(), govi.Dump()
	if !full {
		nd, gd = bodyOf(nd), bodyOf(gd)
	}
	diffs := DiffScreens(nd, gd)
	nr, nc := nvi.Cursor()
	gr, gc := govi.Cursor()
	defer pair.send([]byte("\x1b:q!\r:q!\r"))
	if len(diffs) == 0 && nr == gr && nc == gc {
		t.Logf("[%-14s] match   (cursor %d,%d)  %s", name, nr, nc, desc)
		return false
	}
	t.Logf("[%-14s] DIVERGE  %s", name, desc)
	if nr != gr || nc != gc {
		t.Logf("    cursor: nvi=%d,%d govi=%d,%d", nr, nc, gr, gc)
	}
	if len(diffs) > 0 {
		t.Logf("    screen (nvi | govi):\n%s", indent(FormatDiffs(diffs)))
	}
	return true
}

func TestCeditAudit(t *testing.T) {
	type pc struct {
		name, desc, keys string
		full             bool
	}
	cases := []pc{
		// P1 + P14: window opens below, small (cap at 24 rows), shows the
		// history with the ':' prefixes, cursor on the last line.
		{"P1-open", "trigger opens the history window below",
			cedSET + ":set nu\r:set nonu\r:\x1b", false},
		// P2: consecutive duplicates are logged once.
		{"P2-dedupe", "consecutive duplicate commands log once",
			cedSET + ":set nu\r:set nu\r:set nonu\r:\x1b", false},
		// P3: text typed before the trigger is logged, not executed.
		{"P3-partial", "partial colon text is logged unexecuted",
			cedSET + ":xyzzy\x1b", false},
		// P4: <CR> executes the current line in the parent and closes.
		{"P4-cr-exec", "<CR> runs the line in the parent and closes",
			cedSET + ":set nu\r:set nonu\r:\x1bkk\r", false},
		// P5: edits persist in the history; the edited line executes.
		{"P5-edit-exec", "edited history line executes and persists",
			cedSET + ":set nonu\r:\x1bcc:set nu\x1b\r:\x1b", false},
		// P6: <CR> on an empty line reports and stays open (message row).
		{"P6-empty-line", "<CR> on an empty line: No ex command to execute",
			cedSET + ":set nu\r:\x1bo\x1b\r", true},
		// P7: ^W is refused with the fixed message.
		{"P7-ctrl-w", "^W refused inside the window",
			cedSET + ":set nu\r:\x1b\x17", true},
		// P8: :q closes silently despite the dirtied history buffer.
		{"P8-quit", ":q closes the window silently",
			cedSET + ":set nu\r:\x1bx:q\r", false},
		// P9: failing commands are logged too (log precedes exec).
		{"P9-fail-log", "failing command still logged",
			cedSET + ":bogus\r:\x1b", false},
		// P10: is a bare ':' logged? (source says yes: the TEXT buffer holds
		// the prompt character)
		{"P10-bare-colon", "bare :<CR> logging",
			cedSET + ":\r:set nu\r:\x1b", false},
		// P11: the window's own modeline (nvi shows its random temp path, so
		// this is expected to DIVERGE; kept for the record).
		{"P11-modeline", "expect DIVERGE: window modeline naming",
			cedSET + ":set nu\r:\x1b", true},
		// P12: cedit == filec == tab: window on an empty line, completion
		// after text.
		{"P12a-tab-empty", "tab as cedit fires on an empty colon line",
			":set cedit=\x16\t\r:set nu\r:\t", false},
		{"P12b-tab-text", "tab after text falls through to completion",
			":set cedit=\x16\t\r:set nu\r:e nosuchpfx\t", false},
		// P13: with cedit unset, ESC at the colon prompt still cancels.
		{"P13-esc-unset", "ESC cancels the colon line when cedit is unset",
			":\x1b", false},
		// Extra: trigger again from inside the open window.
		{"PX-reopen", "trigger inside the window",
			cedSET + ":set nu\r:\x1b:\x1b", false},
	}
	div := 0
	for _, c := range cases {
		if runCedit(t, c.name, c.desc, c.keys, c.full) {
			div++
		}
	}
	t.Logf("cedit-audit: %d/%d diverged", div, len(cases))
}
