package goterm

// termstate_test.go pins terminal-state behavior under the goterm terminal
// type (see terminfo.go): a full-screen vi session runs on the alternate
// screen buffer and restores the primary on quit; an ex session (-e) is a
// scrolling line interface and never touches it. Motivated by a real-terminal
// bug this harness could not see under TERM=ansi: govi quitting from ex mode
// re-entered the alternate screen after the tty was already restored, leaving
// the terminal raw on a cleared screen (fixed in govi commit 71b50db).

import (
	"strings"
	"testing"
	"time"
)

// waitCond polls a Term-state predicate; use it for state WaitFor cannot see
// (which screen buffer is active), with the same readiness caveats.
func waitCond(timeout time.Duration, pred func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if pred() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func TestParityTermState(t *testing.T) {
	if !AltScreenSupported() {
		t.Skip("tic unavailable; sessions run under plain TERM=ansi")
	}
	editors := []struct {
		name, path string
	}{{"nvi", nviPath}, {"govi", goviPath}}

	for _, ed := range editors {
		t.Run(ed.name+"-vi", func(t *testing.T) {
			file := writeLines(t, []string{"alpha", "beta"})
			tm := startArgs(t, ed.path, 12, 60, file)
			defer tm.Close()
			if !waitCond(3*time.Second, tm.AltScreenActive) {
				t.Fatal("vi session did not enter the alternate screen")
			}
			tm.Send([]byte(":q!\r"))
			if !waitCond(3*time.Second, func() bool { return !tm.AltScreenActive() }) {
				t.Fatal("alternate screen still active after :q!")
			}
		})
		t.Run(ed.name+"-ex", func(t *testing.T) {
			file := writeLines(t, []string{"alpha", "beta"})
			tm := startArgs(t, ed.path, 12, 60, "-e", file)
			defer tm.Close()
			ok := tm.WaitFor(3*time.Second, func(screen []string) bool {
				return strings.Contains(strings.Join(screen, "\n"), ":")
			})
			if !ok {
				t.Fatalf("no ex prompt; screen:\n%s", strings.Join(tm.Dump(), "\n"))
			}
			if tm.AltScreenActive() {
				t.Fatal("ex session entered the alternate screen")
			}
			tm.Send([]byte("q\n"))
			tm.WaitQuiet(150*time.Millisecond, 2*time.Second)
			if tm.AltScreenActive() {
				t.Fatal("alternate screen active after ex quit")
			}
		})
	}
}
