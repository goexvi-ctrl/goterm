package goterm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCaptureUsage is a throwaway helper (not a divergence test): it dumps the
// full :viusage and :exusage output of nvi and govi into untracked files under
// usage-ref/ for offline reference. Run explicitly:
//   go test -run TestCaptureUsage -v .
func TestCaptureUsage(t *testing.T) {
	if os.Getenv("CAPTURE_USAGE") == "" {
		t.Skip("set CAPTURE_USAGE=1 to regenerate usage-ref/*.txt")
	}
	const rows, cols = 240, 120
	bins := []struct{ name, bin string }{{"nvi", nviPath}, {"govi", goviPath}}
	for _, b := range bins {
		if _, err := os.Stat(b.bin); err != nil {
			t.Skipf("%s not available", b.bin)
		}
	}
	cmds := []struct{ label, keys string }{
		{"viusage", ":viusage\r"},
		{"exusage", ":exusage\r"},
	}
	outDir := "usage-ref"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// clean strips the tilde fill column, the +=+= pager bar, and trailing blank
	// rows, leaving the usage text. The last row (status/echo) is dropped.
	clean := func(d []string) string {
		if len(d) > 0 {
			d = d[:len(d)-1]
		}
		out := make([]string, 0, len(d))
		for _, r := range d {
			r = strings.TrimRight(r, " ")
			if r == "~" {
				r = ""
			}
			if strings.HasPrefix(r, "+=+=") {
				continue
			}
			out = append(out, r)
		}
		// trim leading and trailing empties
		for len(out) > 0 && out[0] == "" {
			out = out[1:]
		}
		for len(out) > 0 && out[len(out)-1] == "" {
			out = out[:len(out)-1]
		}
		return strings.Join(out, "\n") + "\n"
	}

	for _, b := range bins {
		for _, c := range cmds {
			tm := New(rows, cols)
			if err := tm.Start(b.bin); err != nil {
				t.Fatalf("start %s: %v", b.bin, err)
			}
			if !tm.WaitFor(5*time.Second, func(d []string) bool { return len(d) > 1 && d[1] == "~" }) {
				t.Fatalf("%s did not reach empty-buffer state", b.name)
			}
			tm.WaitQuiet(300*time.Millisecond, 3*time.Second)
			tm.Send([]byte(c.keys))
			tm.WaitQuiet(400*time.Millisecond, 5*time.Second)
			text := clean(tm.Dump())
			path := filepath.Join(outDir, b.name+"."+c.label+".txt")
			if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Logf("wrote %s (%d lines)", path, strings.Count(text, "\n"))
			tm.Close()
		}
	}
}
