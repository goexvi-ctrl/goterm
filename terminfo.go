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
// The entry also declares xenl: the emulator implements vt100-style DEFERRED
// wrap (writing the last column leaves the cursor pending, see TestWriteDeferredWrap),
// but ansi's terminfo has am without xenl, which promises IMMEDIATE wrap.
// Under that lie, ncurses' cursor model desynced from the emulator whenever a
// buffer line exactly filled the screen width, scrambling the rendered rows
// (found by the 2026-07 QA review's exact-width probes).
//
// The entry is compiled with tic(1) into a per-process directory on first
// use; applications find it via TERMINFO (ncurses/nvi natively, tcell/govi
// through its infocmp-based dynamic lookup). If tic is unavailable the
// harness falls back to TERM=ansi and AltScreenSupported reports false.

// The special-key capabilities (kich1/kdch1/kpp/knp) use the xterm sequences
// and exist so key-mapping behavior is probeable at all: nvi seeds command
// maps from terminfo at startup (cl/cl_term.c: kich1->i, kpp->^B, knp->^F,
// kdch1->x) and tcell decodes input by the same entry. ansi's own kich1 is
// the nonstandard \E[L; it is overridden here so probes can send the familiar
// \E[2~. Without these caps the harness was structurally blind to the
// Insert/PageUp/PageDown gap (govi qa/CORNERS.md Part C #10).
const gotermTi = `goterm|headless goterm emulator (ansi plus alternate screen),
	xenl,
	use=ansi,
	smcup=\E[?1049h, rmcup=\E[?1049l,
	kich1=\E[2~, kdch1=\E[3~, kpp=\E[5~, knp=\E[6~,
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
