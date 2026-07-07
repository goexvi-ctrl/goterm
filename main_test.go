package goterm

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// recoverParent is the single directory under which every test run's throwaway
// recovery directory is created. Keeping them all under one parent means a run
// that dies before its cleanup can be swept wholesale with
// `rm -rf /var/tmp/goterm_testing`.
const recoverParent = "/var/tmp/goterm_testing"

// TestMain isolates crash-recovery files for the whole goterm test binary.
//
// The harness drives real vi implementations (the nvi oracle and govi) on a
// PTY, and any edit makes them drop a recover.* file into their recovery
// directory. Left at the default that directory is the shared system location
// (/var/tmp/vi.recover), so a test run -- especially one that kills an editor
// mid-edit -- would litter it with stale files. Both editors honor
// GOTERM_ORACLE_PRESERVE as the recovery directory (govi via its recdir option
// default in engine/options.go, the nvi oracle via common/options.c), so we
// point it at a fresh /var/tmp/goterm_testing/vi.recover.XXXXXX and remove that
// directory when the suite ends. Start (pty.go) hands the parent environment to
// every spawned editor, so setting it here reaches both.
func TestMain(m *testing.M) {
	if err := os.MkdirAll(recoverParent, 0o700); err != nil {
		panic(err)
	}
	dir, err := os.MkdirTemp(recoverParent, "vi.recover.")
	if err != nil {
		panic(err)
	}
	os.Setenv("GOTERM_ORACLE_PRESERVE", dir)

	code := m.Run()

	os.RemoveAll(dir)
	// Best effort: remove the shared parent too. This succeeds only when it is
	// empty, so a concurrent run's directory is never disturbed.
	os.Remove(recoverParent)
	os.Exit(code)
}

// TestRecoveryDirIsolated checks that both editors honor GOTERM_ORACLE_PRESERVE:
// after an edit and a forced recovery snapshot (:preserve) each writes its
// recover.* file into the throwaway directory TestMain set up, and therefore not
// into the shared /var/tmp/vi.recover. govi picks it up via its recdir option
// default (engine/options.go); the nvi oracle via common/options.c. This guards
// against a regression -- or a stale oracle binary that predates the options.c
// change -- reintroducing pollution of the shared directory.
func TestRecoveryDirIsolated(t *testing.T) {
	recdir := os.Getenv("GOTERM_ORACLE_PRESERVE")
	if recdir == "" {
		t.Fatal("GOTERM_ORACLE_PRESERVE not set; TestMain should have set it")
	}

	// preservesToRecdir drives the editor to make a change and force a recovery
	// snapshot (:preserve), then reports whether a recover.* file landed in
	// recdir. Any pre-existing recover.* files are cleared first so the result is
	// unambiguous for the editor under test.
	preservesToRecdir := func(t *testing.T, path string) bool {
		t.Helper()
		if old, _ := filepath.Glob(filepath.Join(recdir, "recover.*")); len(old) > 0 {
			for _, f := range old {
				os.Remove(f)
			}
		}
		file := makeNumberedFile(t, 3)
		tm := New(24, 80)
		if err := tm.Start(path, file); err != nil {
			t.Fatalf("Start %s: %v", path, err)
		}
		defer tm.Close()
		if !tm.WaitFor(5*time.Second, func(d []string) bool { return d[0] != "" }) {
			t.Fatalf("%s did not open the file", path)
		}
		// Make a change (so there is something to preserve), leave insert mode,
		// then force a recovery snapshot with :preserve.
		tm.Send([]byte("ox\x1b"))
		tm.WaitQuiet(200*time.Millisecond, 2*time.Second)
		tm.Send([]byte(":preserve\r"))
		tm.WaitQuiet(300*time.Millisecond, 2*time.Second)

		got, _ := filepath.Glob(filepath.Join(recdir, "recover.*"))
		return len(got) > 0
	}

	editors := []struct{ name, path string }{
		{"govi", "/Users/claude/bin/govi"},
		{"nvi", "/Users/claude/src/nvi/build.unix/vi"},
	}
	for _, e := range editors {
		t.Run(e.name, func(t *testing.T) {
			if _, err := os.Stat(e.path); err != nil {
				t.Skipf("%s not available", e.path)
			}
			if !preservesToRecdir(t, e.path) {
				t.Errorf("%s did not write its recovery file into the isolated recdir %s "+
					"(a stale binary predating the GOTERM_ORACLE_PRESERVE change would pollute /var/tmp/vi.recover)",
					e.name, recdir)
			}
		})
	}
}
