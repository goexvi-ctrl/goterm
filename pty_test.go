package goterm

import (
	"os"
	"strings"
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

// TestPTYNvi is the real goal: drive nvi through some basic editing and confirm
// the screen matches what nvi should render.
func TestPTYNvi(t *testing.T) {
	const nvi = "/opt/homebrew/bin/nvi"
	if _, err := os.Stat(nvi); err != nil {
		t.Skip("nvi not available")
	}
	tm := New(24, 80)
	if err := tm.Start(nvi); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer tm.Close()
	dump := func() string { return strings.Join(tm.Dump(), "\n") }

	// nvi draws an empty buffer: a blank first line, tildes for the rest, and a
	// status line at the bottom.
	if !tm.WaitFor(5*time.Second, func(d []string) bool { return d[1] == "~" }) {
		t.Fatalf("nvi did not initialize; screen:\n%s", dump())
	}
	if d := tm.Dump(); d[0] != "" {
		t.Errorf("initial row 0 = %q, want empty", d[0])
	}

	// Insert a line (i...Esc), then open a second line below it (o...Esc).
	tm.Send([]byte("iHello, world\x1bosecond line\x1b"))
	ok := tm.WaitFor(5*time.Second, func(d []string) bool {
		return d[0] == "Hello, world" && d[1] == "second line"
	})
	if !ok {
		t.Errorf("edited text did not render as expected; screen:\n%s", dump())
	}
	// The lines below the text are still empty (tildes).
	if d := tm.Dump(); d[2] != "~" {
		t.Errorf("row 2 = %q, want %q", d[2], "~")
	}

	// Quit without saving.
	tm.Send([]byte(":q!\n"))
}
