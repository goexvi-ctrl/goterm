package goterm

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestProbeExBangParity compares the ex-mode (Q) :! transcript against nvi:
// "!" expansion to the previous bang command (with an error when there is
// none), the redisplay of an expanded command, and the closing "!" line.
func TestProbeExBangParity(t *testing.T) {
	for _, p := range []string{nviPath, goviPath} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("%s not available", p)
		}
	}

	transcript := func(path string) []string {
		tm := New(24, 80)
		if err := tm.Start(path, writeLines(t, []string{"hello"})); err != nil {
			t.Fatal(err)
		}
		defer tm.Close()
		if !tm.WaitFor(5*time.Second, func(d []string) bool {
			return strings.TrimSpace(d[len(d)-1]) != ""
		}) {
			t.Fatalf("%s did not become ready", path)
		}
		tm.WaitQuiet(100*time.Millisecond, 2*time.Second)

		tm.Send([]byte("Q"))
		if !tm.WaitFor(5*time.Second, func(d []string) bool {
			return !tm.AltScreenActive()
		}) {
			t.Fatalf("%s: Q did not enter ex line mode", path)
		}
		tm.WaitQuiet(100*time.Millisecond, 2*time.Second)

		for _, line := range []string{"!!echo one", "!echo one", "!!echo two",
			"1!tr a-z A-Z", "!false"} {
			tm.Send([]byte(line + "\r"))
			tm.WaitQuiet(150*time.Millisecond, 5*time.Second)
		}

		var out []string
		for _, row := range tm.Dump() {
			if r := strings.TrimRight(row, " "); r != "" {
				out = append(out, r)
			}
		}
		tm.Send([]byte("q!\r"))
		return out
	}

	nvi := transcript(nviPath)
	govi := transcript(goviPath)
	if diffs := DiffScreens(nvi, govi); len(diffs) > 0 {
		t.Errorf("ex-mode :! transcript diverges (nvi | govi):\n%s",
			indent(FormatDiffs(diffs)))
	} else {
		t.Logf("transcripts match:\n%s", indent(strings.Join(govi, "\n")))
	}
}

// TestProbeExModeBangHang reproduces the reported hang: enter ex line mode
// with Q, then run a shell command with :!.  presentBangOutput used to paint
// the suspended tcell screen, which spun forever in the draw loop.  The test
// passes when the command output (or completion message) and a fresh ":"
// prompt appear, and the editor still responds afterward.
func TestProbeExModeBangHang(t *testing.T) {
	if _, err := os.Stat(goviPath); err != nil {
		t.Skipf("%s not available", goviPath)
	}
	tm := New(24, 80)
	if err := tm.Start(goviPath, writeLines(t, []string{"hello", "world"})); err != nil {
		t.Fatal(err)
	}
	defer tm.Close()

	if !tm.WaitFor(5*time.Second, func(d []string) bool {
		return strings.TrimSpace(d[len(d)-1]) != ""
	}) {
		t.Fatal("govi did not become ready")
	}
	tm.WaitQuiet(100*time.Millisecond, 2*time.Second)

	// Q leaves the alternate screen for the ex line REPL and prints ":".
	tm.Send([]byte("Q"))
	if !tm.WaitFor(5*time.Second, func(d []string) bool {
		return !tm.AltScreenActive()
	}) {
		t.Fatalf("Q did not enter ex line mode:\n%s", strings.Join(tm.Dump(), "\n"))
	}
	tm.WaitQuiet(100*time.Millisecond, 2*time.Second)

	// The reported sequence: !!who at the colon prompt.  Before the fix this
	// hung with the editor spinning; nothing more was ever printed.
	// Two stages so the fresh ":" prompt is not confused with the one the
	// command was typed at: first the echoed command line, then a bare ":"
	// prompt as the last nonblank row once the transcript has printed (the
	// echo itself may scroll off when the command's output is long).
	tm.Send([]byte("!!who\r"))
	if !tm.WaitFor(5*time.Second, func(d []string) bool {
		for _, row := range d {
			if strings.TrimRight(row, " ") == ":!!who" {
				return true
			}
		}
		return false
	}) {
		t.Fatalf(":!!who was not echoed:\n%s", strings.Join(tm.Dump(), "\n"))
	}
	if !tm.WaitFor(10*time.Second, func(d []string) bool {
		last := ""
		for _, row := range d {
			if r := strings.TrimRight(row, " "); r != "" {
				last = r
			}
		}
		return last == ":"
	}) {
		t.Fatalf("no fresh prompt after !!who (hang?):\n%s", strings.Join(tm.Dump(), "\n"))
	}

	// The editor must still be responsive: return to vi mode and quit.
	tm.Send([]byte("vi\r"))
	if !tm.WaitFor(5*time.Second, func(d []string) bool {
		return tm.AltScreenActive()
	}) {
		t.Fatalf("vi did not return to visual mode:\n%s", strings.Join(tm.Dump(), "\n"))
	}
	tm.Send([]byte(":q!\r"))
}
