package goterm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// The emulator advertises its own terminal type, "goterm": the ansi terminfo
// entry plus the alternate-screen capabilities (smcup/rmcup) the emulator
// also implements (see enterAlternate/exitAlternate). Plain TERM=ansi hid an
// entire class of bugs from the harness: ansi has no smcup/rmcup, so editors
// never emitted alternate-screen switches and nothing could catch, e.g., an
// editor that quit without restoring the primary screen.
//
// The entry is compiled with tic(1) into a per-process directory on first
// use; applications find it via TERMINFO (ncurses/nvi natively, tcell/govi
// through its infocmp-based dynamic lookup). If tic is unavailable the
// harness falls back to TERM=ansi and AltScreenSupported reports false.

const gotermTi = `goterm|headless goterm emulator (ansi plus alternate screen),
	use=ansi,
	smcup=\E[?1049h, rmcup=\E[?1049l,
`

var (
	tiOnce sync.Once
	tiDir  string // TERMINFO dir with the compiled entry; "" means fall back to ansi
)

// terminfoEnv returns the TERM/TERMINFO environment for a spawned application.
func terminfoEnv() []string {
	tiOnce.Do(compileTerminfo)
	if tiDir == "" {
		return []string{"TERM=ansi"}
	}
	return []string{"TERM=goterm", "TERMINFO=" + tiDir}
}

func compileTerminfo() {
	dir, err := os.MkdirTemp("", "goterm-ti-*")
	if err != nil {
		return
	}
	src := filepath.Join(dir, "goterm.ti")
	if err := os.WriteFile(src, []byte(gotermTi), 0o644); err != nil {
		return
	}
	if out, err := exec.Command("tic", "-x", "-o", dir, src).CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "goterm: tic failed (falling back to TERM=ansi): %v\n%s", err, out)
		return
	}
	tiDir = dir
}

// AltScreenSupported reports whether spawned applications run under the
// goterm terminal type (with smcup/rmcup) rather than the plain-ansi
// fallback. Tests asserting alternate-screen behavior skip when false.
func AltScreenSupported() bool {
	tiOnce.Do(compileTerminfo)
	return tiDir != ""
}
