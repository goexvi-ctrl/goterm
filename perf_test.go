package goterm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
)

// perf_test.go is a gated, realistic editing-session benchmark comparing govi and
// nvi: it drives each editor through one scripted session on a largish file (mixed
// substitutions, global commands, navigation, paging, inserts/deletes, shifts,
// undo) on a real PTY, draining output fast so the editor is never throttled by
// the parent, and reports the child process's real / user / sys time. Run it with:
//   PERF=1 go test -run TestPerfSession -v -timeout 600s .
// Not part of the divergence suite; it asserts nothing, it measures.

// genPerfFile writes n lines of varied-but-repetitive text (recurring tokens so
// substitutions and global commands actually match) and returns the path.
func genPerfFile(t *testing.T, dir string, n int) string {
	t.Helper()
	path := filepath.Join(dir, "big.txt")
	var b strings.Builder
	b.Grow(n * 80)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b,
			"func process_%05d(the quick brown fox) { return foo + bar + baz; } // comment alpha beta %05d\n",
			i, i)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// perfScript builds one extensive editing session as a keystroke stream. Each
// round pairs every mutation with its inverse so the file stays ~stable across
// rounds (so every round does comparable work and the two editors do not drift),
// and avoids any command that prints (which would trip nvi's "press any key"
// pager and desync the input). It ends by writing the file and quitting.
func perfScript(rounds int) []byte {
	var s strings.Builder
	for r := 0; r < rounds; r++ {
		s.WriteString("\x1b")                 // ensure command mode
		s.WriteString("gg")                   // top of file
		s.WriteString(":%s/foo/FOO/g\r")      // global substitute (heavy)
		s.WriteString(":%s/FOO/foo/g\r")      // ... and revert
		s.WriteString(":g/comment/s/alpha/ALPHA/\r") // global + substitute
		s.WriteString(":g/comment/s/ALPHA/alpha/\r") // ... and revert
		s.WriteString("/quick\r")             // search
		s.WriteString("nnnnn")                // repeat search (navigate)
		s.WriteString("5GddP")                // delete a line and put it back
		s.WriteString("100GoINSERTED\x1bdd")  // open+insert a line, then delete it
		s.WriteString("Gyyp\x1bdd")           // yank/put at EOF, then delete
		s.WriteString("gg\x06\x06\x06\x06")   // ^F page forward x4
		s.WriteString("\x02\x02")             // ^B page back x2
		s.WriteString("\x04\x04\x15")         // ^D ^D ^U half-page scroll
		s.WriteString(":1,800>\r:1,800<\r")   // shift 800 lines right then left
		s.WriteString("ggcwfunc\x1b")         // change a word (insert), back to cmd
		s.WriteString("uu")                   // undo twice
	}
	s.WriteString("\x1b\x1b:wq!\r") // write the file and quit
	return []byte(s.String())
}

type perfResult struct {
	real, session, user, sys time.Duration
	timedOut                 bool
}

// perfEditor identifies an editor binary and any args to run it cleanly.
type perfEditor struct {
	name string
	bin  string
	args []string
}

// perfEditors returns the editors to benchmark that are present. vim is run with a
// clean config (no vimrc/viminfo/swap, nocompatible) so it is not penalized by
// plugins or helped by a tuned rc -- closest to the others' out-of-the-box state.
func perfEditors(t *testing.T) []perfEditor {
	cands := []perfEditor{
		{"nvi", nviPath, nil},
		{"vim", "/usr/bin/vim", []string{"-u", "NONE", "-N", "-n", "-i", "NONE"}},
		{"govi", goviPath, nil},
	}
	var out []perfEditor
	for _, e := range cands {
		if _, err := os.Stat(e.bin); err == nil {
			out = append(out, e)
		}
	}
	return out
}

// runPerfSession runs one session of bin (with args) on a fresh copy of base and
// returns its timings. Output is drained fast so the editor renders at full speed
// without parent backpressure; CPU times come from the child's rusage.
func runPerfSession(t *testing.T, bin string, args []string, base string, script []byte) perfResult {
	t.Helper()
	dir := t.TempDir()
	f := filepath.Join(dir, "edit.txt")
	data, err := os.ReadFile(base)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f, data, 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, append(append([]string{}, args...), f)...)
	cmd.Env = append(os.Environ(), "TERM=ansi")
	tStart := time.Now()
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatal(err)
	}

	// Drain output fast; signal once the editor has produced its first screen.
	var firstOnce sync.Once
	firstCh := make(chan struct{})
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				firstOnce.Do(func() { close(firstCh) })
			}
			if err != nil {
				return
			}
		}
	}()
	select {
	case <-firstCh:
	case <-time.After(5 * time.Second):
		t.Fatalf("%s: no initial output (startup)", bin)
	}

	tInput := time.Now()
	// Write the whole script; this may block as the PTY buffer fills, resuming as
	// the editor consumes -- safe because output is drained concurrently.
	go func() {
		ptmx.Write(script)
	}()

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	timeout := 120 * time.Second
	if v := os.Getenv("PERF_TIMEOUT"); v != "" {
		var secs int
		fmt.Sscan(v, &secs)
		timeout = time.Duration(secs) * time.Second
	}
	timedOut := false
	select {
	case <-done:
	case <-time.After(timeout):
		timedOut = true
		cmd.Process.Kill()
		<-done
	}
	tEnd := time.Now()
	_ = ptmx.Close()

	ps := cmd.ProcessState
	r := perfResult{real: tEnd.Sub(tStart), session: tEnd.Sub(tInput), timedOut: timedOut}
	if ps != nil {
		r.user = ps.UserTime()
		r.sys = ps.SystemTime()
	}
	return r
}

// buildScript repeats round() the given number of times and appends the quit.
func buildScript(rounds int, round func(*strings.Builder)) []byte {
	var s strings.Builder
	for i := 0; i < rounds; i++ {
		s.WriteString("\x1b")
		round(&s)
	}
	s.WriteString("\x1b\x1b:wq!\r")
	return []byte(s.String())
}

// TestPerfBreakdown attributes the govi/nvi gap to operation classes by running an
// isolated script per class and reporting CPU for each.
//   PERF=1 go test -run TestPerfBreakdown -v -timeout 600s .
func TestPerfBreakdown(t *testing.T) {
	if os.Getenv("PERF") == "" {
		t.Skip("set PERF=1 to run the breakdown benchmark")
	}
	for _, bin := range []string{nviPath, goviPath} {
		if _, err := os.Stat(bin); err != nil {
			t.Skipf("%s not available", bin)
		}
	}
	lines := 10000
	if v := os.Getenv("PERF_LINES"); v != "" {
		fmt.Sscan(v, &lines)
	}
	base := genPerfFile(t, t.TempDir(), lines)

	modes := []struct {
		name   string
		rounds int
		round  func(*strings.Builder)
	}{
		{"subst %s/g", 30, func(s *strings.Builder) {
			s.WriteString("gg:%s/foo/FOO/g\r:%s/FOO/foo/g\r")
		}},
		{"global :g", 30, func(s *strings.Builder) {
			s.WriteString("gg:g/comment/s/alpha/ALPHA/\r:g/comment/s/ALPHA/alpha/\r")
		}},
		{"shift > <", 30, func(s *strings.Builder) {
			s.WriteString(":1,8000>\r:1,8000<\r")
		}},
		{"search n", 30, func(s *strings.Builder) {
			s.WriteString("gg/quick\r")
			s.WriteString(strings.Repeat("n", 50))
		}},
		{"paging ^F^B", 8, func(s *strings.Builder) {
			s.WriteString("gg")
			s.WriteString(strings.Repeat("\x06", 60)) // ^F
			s.WriteString(strings.Repeat("\x02", 60)) // ^B
		}},
		// NB: an insert/Esc-heavy "edit churn" mode is intentionally omitted -- under
		// machine-speed input nvi/vim stall on their per-Esc key-disambiguation timer
		// (a human pauses after Esc), which is a measurement artifact, not CPU cost.
	}

	fmtCPU := func(r perfResult) string {
		if r.timedOut {
			return ">timeout"
		}
		return (r.user + r.sys).Round(time.Millisecond).String()
	}
	editors := perfEditors(t)
	names := make([]string, len(editors))
	for i, e := range editors {
		names[i] = e.name
	}
	t.Logf("file: %d lines   editors: %s   (cpu = user+sys)", lines, strings.Join(names, ", "))
	for _, m := range modes {
		script := buildScript(m.rounds, m.round)
		res := map[string]perfResult{}
		for _, e := range editors {
			res[e.name] = runPerfSession(t, e.bin, e.args, base, script)
		}
		var cols []string
		for _, e := range editors {
			cols = append(cols, fmt.Sprintf("%s=%s", e.name, fmtCPU(res[e.name])))
		}
		// govi-relative ratios when available.
		ratio := ""
		g := res["govi"]
		if !g.timedOut && g.user+g.sys > 0 {
			var rs []string
			for _, base := range []string{"nvi", "vim"} {
				if b, ok := res[base]; ok && !b.timedOut && b.user+b.sys > 0 {
					rs = append(rs, fmt.Sprintf("govi/%s=%.1fx", base, float64(g.user+g.sys)/float64(b.user+b.sys)))
				}
			}
			ratio = "   " + strings.Join(rs, " ")
		}
		t.Logf("%-12s %3d rounds | %-42s|%s", m.name, m.rounds, strings.Join(cols, "  "), ratio)
	}
}

// TestPerfProfile samples govi's stacks (macOS `sample`) during a redraw-heavy
// session to show which functions the CPU is in.
//   PERF=1 go test -run TestPerfProfile -v -timeout 120s .
func TestPerfProfile(t *testing.T) {
	if os.Getenv("PERF") == "" {
		t.Skip("set PERF=1 to run the profile")
	}
	if _, err := exec.LookPath("sample"); err != nil {
		t.Skip("macOS `sample` not available")
	}
	if _, err := os.Stat(goviPath); err != nil {
		t.Skip("govi not available")
	}
	lines := 10000
	dir := t.TempDir()
	base := genPerfFile(t, dir, lines)
	f := filepath.Join(dir, "edit.txt")
	data, _ := os.ReadFile(base)
	os.WriteFile(f, data, 0o644)

	cmd := exec.Command(goviPath, f)
	cmd.Env = append(os.Environ(), "TERM=ansi")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		buf := make([]byte, 32*1024)
		for {
			if _, err := ptmx.Read(buf); err != nil {
				return
			}
		}
	}()
	time.Sleep(500 * time.Millisecond) // let it load + draw

	// A long redraw-heavy stream: search + paging, enough to outlast the sample.
	script := buildScript(200, func(s *strings.Builder) {
		s.WriteString("gg/quick\r")
		s.WriteString(strings.Repeat("n", 40))
		s.WriteString("gg")
		s.WriteString(strings.Repeat("\x06", 40))
		s.WriteString(strings.Repeat("\x02", 40))
	})
	go func() { ptmx.Write(script) }()
	time.Sleep(800 * time.Millisecond) // get into steady state

	out, _ := exec.Command("sample", fmt.Sprint(cmd.Process.Pid), "6", "-mayDie").CombinedOutput()
	cmd.Process.Kill()
	cmd.Wait()
	ptmx.Close()

	if p := os.Getenv("PERF_SAMPLE_OUT"); p != "" {
		os.WriteFile(p, out, 0o644)
		t.Logf("full sample written to %s", p)
	}

	// Log the "Sort by top of stack" leaf summary (clearest "where CPU is"), else
	// the call-graph head.
	text := string(out)
	if i := strings.Index(text, "Sort by top of stack"); i >= 0 {
		tail := text[i:]
		lines := strings.Split(tail, "\n")
		if len(lines) > 45 {
			lines = lines[:45]
		}
		t.Logf("govi sample -- leaf functions by self time:\n%s", strings.Join(lines, "\n"))
	} else {
		head := strings.Split(text, "\n")
		if len(head) > 80 {
			head = head[:80]
		}
		t.Logf("govi sample (call graph head):\n%s", strings.Join(head, "\n"))
	}
}

func TestPerfSession(t *testing.T) {
	if os.Getenv("PERF") == "" {
		t.Skip("set PERF=1 to run the editing-session benchmark")
	}
	for _, bin := range []string{nviPath, goviPath} {
		if _, err := os.Stat(bin); err != nil {
			t.Skipf("%s not available", bin)
		}
	}
	lines := 10000
	rounds := 60
	if v := os.Getenv("PERF_LINES"); v != "" {
		fmt.Sscan(v, &lines)
	}
	if v := os.Getenv("PERF_ROUNDS"); v != "" {
		fmt.Sscan(v, &rounds)
	}
	base := genPerfFile(t, t.TempDir(), lines)
	fi, _ := os.Stat(base)
	script := perfScript(rounds)
	t.Logf("file: %d lines (%d KiB)   script: %d rounds (%d keystroke bytes)",
		lines, fi.Size()/1024, rounds, len(script))
	t.Logf("%-6s %-4s %10s %10s %10s %10s %8s", "editor", "run", "real", "session", "user", "sys", "cpu%")

	editors := perfEditors(t)
	const runs = 2
	agg := map[string][]perfResult{}
	for _, e := range editors {
		for run := 1; run <= runs; run++ {
			r := runPerfSession(t, e.bin, e.args, base, script)
			cpu := float64(r.user+r.sys) / float64(r.session) * 100
			t.Logf("%-6s %-4d %10s %10s %10s %10s %7.0f%%",
				e.name, run, r.real.Round(time.Millisecond), r.session.Round(time.Millisecond),
				r.user.Round(time.Millisecond), r.sys.Round(time.Millisecond), cpu)
			agg[e.name] = append(agg[e.name], r)
		}
	}
	// Best-of-N (min) summary per editor -- least noisy.
	t.Logf("--- best-of-%d (min) ---", runs)
	for _, e := range editors {
		rs := agg[e.name]
		best := rs[0]
		for _, r := range rs[1:] {
			if r.user+r.sys < best.user+best.sys {
				best = r
			}
		}
		t.Logf("%-6s  real=%s session=%s user=%s sys=%s cpu=%s",
			e.name, best.real.Round(time.Millisecond), best.session.Round(time.Millisecond),
			best.user.Round(time.Millisecond), best.sys.Round(time.Millisecond),
			(best.user + best.sys).Round(time.Millisecond))
	}
}
