package goterm

import (
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
)

// ptySession holds the state of an application running on the terminal's PTY.
type ptySession struct {
	cmd  *exec.Cmd
	ptmx *os.File
	stop chan struct{}
}

// Start launches name (with args) on a new pseudo-terminal sized to the terminal
// and begins driving the screen from it.  Two goroutines run for the life of the
// session: one pumps the application's output through Write (rendering it), and
// one forwards the terminal's return stream (Out) -- DSR/DA responses and any
// keystrokes sent via Send -- to the application's input.
//
// Read the screen with Dump or WaitFor while the application runs, deliver
// keystrokes with Send, and stop it with Close.
func (t *Term) Start(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	// Advertise TERM=ansi: this emulator implements the ansi terminfo, so the
	// application emits the sequences we support.
	cmd.Env = append(os.Environ(), "TERM=ansi")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: uint16(t.Primary.Rows),
		Cols: uint16(t.Primary.Cols),
	})
	if err != nil {
		return err
	}
	stop := make(chan struct{})
	t.pty = &ptySession{cmd: cmd, ptmx: ptmx, stop: stop}
	t.mu.Lock()
	t.lastWrite = time.Now() // so WaitQuiet does not read as settled before output arrives
	t.mu.Unlock()

	go t.pumpOutput(ptmx)
	go t.forwardInput(ptmx, stop)
	return nil
}

// pumpOutput reads the application's output and feeds it to the screen until the
// PTY is closed.
func (t *Term) pumpOutput(ptmx *os.File) {
	buf := make([]byte, 4096)
	for {
		n, err := ptmx.Read(buf)
		if n > 0 {
			t.Write(buf[:n])
		}
		if err != nil {
			return
		}
	}
}

// forwardInput writes the terminal's return stream to the application's input --
// the one serial line back to the program -- until stopped.
func (t *Term) forwardInput(ptmx *os.File, stop chan struct{}) {
	for {
		select {
		case b := <-t.Out:
			ptmx.Write(b)
		case <-stop:
			return
		}
	}
}

// Close terminates the running application and releases the pseudo-terminal.  It
// is safe to call more than once.
func (t *Term) Close() error {
	p := t.pty
	if p == nil {
		return nil
	}
	t.pty = nil
	close(p.stop)
	p.ptmx.Close()
	if p.cmd.Process != nil {
		p.cmd.Process.Kill()
		p.cmd.Wait()
	}
	return nil
}

// Dump returns the current screen contents (see Screen.Dump) under the lock, so
// it is safe to call while an application is producing output.
func (t *Term) Dump() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Current.Dump()
}

// WaitFor polls the screen until pred is satisfied or timeout elapses, returning
// whether it was satisfied.  It is how to synchronize with an application that
// renders asynchronously: send input, then wait for the screen to reflect it.
func (t *Term) WaitFor(timeout time.Duration, pred func(screen []string) bool) bool {
	deadline := time.Now().Add(timeout)
	for {
		if pred(t.Dump()) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// WaitQuiet waits until the screen has been idle -- no Write -- for at least
// idle, returning true once it settles, or false if timeout elapses first.
//
// Use it after sending input that triggers a redraw, when you don't know the
// exact result to wait for: it lets the application finish drawing.  Choose idle
// comfortably larger than the gaps between chunks of a single redraw but smaller
// than the time you're willing to wait.  (For a known target state, WaitFor is
// more precise; WaitQuiet can settle early if the application is slow to begin
// responding.)
func (t *Term) WaitQuiet(idle, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		t.mu.Lock()
		quietFor := time.Since(t.lastWrite)
		t.mu.Unlock()
		if quietFor >= idle {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(5 * time.Millisecond)
	}
}
