package goterm

import (
	"os"
	"testing"
	"time"
)

// TestPTYCat is the plumbing smoke test: launch cat on a PTY, "type" a line,
// and confirm it appears on the screen (via the terminal's echo).
func TestPTYCat(t *testing.T) {
	if _, err := os.Stat("/bin/cat"); err != nil {
		t.Skip("/bin/cat not available")
	}
	tm := New(24, 80)
	if err := tm.Start("/bin/cat"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer tm.Close()

	tm.Send([]byte("hello\n"))
	if !tm.WaitFor(2*time.Second, func(d []string) bool { return d[0] == "hello" }) {
		t.Errorf("screen did not show %q within timeout; got %q", "hello", tm.Dump()[0])
	}
}
