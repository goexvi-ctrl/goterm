package goterm

// Verification harness for the NVI_CORRECTNESS_FIXES.md review (see
// nvi/govi/NVI_CORRECTNESS_REVIEW.md). It drives THREE editors on identical
// input and reports each result:
//   old-nvi  /opt/homebrew/bin/nvi          -- nvi 1.81.6, BUGGY for post-1.81.6 fixes
//   fix-nvi  nvi/build.unix/vi (the oracle) -- FIXED nvi; the correct reference
//   govi     /Users/claude/bin/govi
// A verdict of "govi != fixed-nvi" flags a possible bug; comparing only against
// old-nvi would be misleading because it lacks the very fixes under review.
//
// Use this to re-verify a govi fix for #8/#9/#10, #13, #15, #32, or #33: a fixed
// case flips to "govi == fixed-nvi (OK)". Rebuild /Users/claude/bin/govi first.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const oraclePath = "/Users/claude/src/nvi/build.unix/vi" // FIXED nvi oracle

type threeCase struct {
	issue string
	file  []string
	keys  string
	desc  string
}

func nvifixesEditors() map[string]string {
	return map[string]string{"old-nvi": nviPath, "fix-nvi": oraclePath, "govi": goviPath}
}

// run3 starts the three editors on the case, sends keys, returns body+cursor.
func run3(t *testing.T, rows, cols int, c threeCase) (map[string][]string, map[string][2]int) {
	t.Helper()
	paths := nvifixesEditors()
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("%s not available", p)
		}
	}
	bodies := map[string][]string{}
	cur := map[string][2]int{}
	for name, path := range paths {
		tm := New(rows, cols)
		var err error
		if len(c.file) > 0 {
			err = tm.Start(path, writeLines(t, c.file))
		} else {
			err = tm.Start(path)
		}
		if err != nil {
			t.Fatalf("start %s: %v", name, err)
		}
		ready := func(d []string) bool {
			if len(c.file) > 0 {
				return strings.TrimSpace(d[len(d)-1]) != ""
			}
			return len(d) > 1 && d[1] == "~"
		}
		if !tm.WaitFor(5*time.Second, ready) {
			tm.Close()
			t.Fatalf("[%s] %s did not become ready", c.issue, name)
		}
		tm.WaitQuiet(200*time.Millisecond, 2*time.Second)
		if c.keys != "" {
			tm.Send([]byte(c.keys))
			tm.WaitQuiet(200*time.Millisecond, 3*time.Second)
		}
		d := tm.Dump()
		bodies[name] = append([]string{}, d[:len(d)-1]...)
		r, cc := tm.Cursor()
		cur[name] = [2]int{r, cc}
		tm.Send([]byte("\x1b:q!\r"))
		tm.Close()
	}
	return bodies, cur
}

func equalBody(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimRight(a[i], " ") != strings.TrimRight(b[i], " ") {
			return false
		}
	}
	return true
}

func bodyView(b []string) string {
	var rs []string
	for i := 0; i < 5 && i < len(b); i++ {
		rs = append(rs, strings.TrimRight(b[i], " "))
	}
	return strings.Join(rs, " / ")
}

func report3(t *testing.T, c threeCase, bodies map[string][]string, cur map[string][2]int) {
	t.Helper()
	matchFix := equalBody(bodies["govi"], bodies["fix-nvi"]) && cur["govi"] == cur["fix-nvi"]
	verdict := "govi != fixed-nvi  (POSSIBLE BUG)"
	if matchFix {
		verdict = "govi == fixed-nvi  (OK)"
	}
	t.Logf("[%s] %s -- %s\n   old-nvi cur=%v  %q\n   fix-nvi cur=%v  %q\n   govi    cur=%v  %q",
		c.issue, verdict, c.desc,
		cur["old-nvi"], bodyView(bodies["old-nvi"]),
		cur["fix-nvi"], bodyView(bodies["fix-nvi"]),
		cur["govi"], bodyView(bodies["govi"]))
}

// TestNviFixes3Way is the main battery: the confirmed-present items plus the
// positive controls (items govi gets right).
func TestNviFixes3Way(t *testing.T) {
	cases := []threeCase{
		// PRESENT
		{"#9 ^A nonword", []string{"foo ^ bar", "baz ^ qux"}, "f^\x01", "^A on a lone ^ -> govi retargets to a word"},
		{"#8 ^A idem", []string{"foo ^ bar", "baz ^ qux"}, "f^\x01\x01", "^A x2 on a lone ^"},
		{"#10 ^A kw", []string{"^foo bar", "x ^foo y"}, "\x01", "^A on ^foo"},
		{"#13 J.", []string{"Hello.", "World"}, "J", "J after '.' -> two spaces in nvi"},
		{"#13 :j.", []string{"Hello.", "World"}, ":1,2j\r", ":j after '.'"},
		{"#33 r!echo%", []string{"x"}, ":r !echo %\r", "% should expand to the file name"},
		// POSITIVE CONTROLS (govi correct)
		{"#11 ^A wend", []string{"a ab abc", "a ab abc"}, "\x01\x01", "^A honors word-end"},
		{"#12 append", []string{"one", "two", "three"}, "\"ayyj\"AyyG\"ap", "named-register append"},
		{"#14 :2j", []string{"aaa", "bbb", "ccc", "ddd"}, ":2j\r", "single-address join"},
		// #30/#1 are NOT PRESENT (see review): #30 body is base-10 (correct); its
		// "POSSIBLE BUG" verdict is only the benign at-rest tab-cursor delta. #1
		// does not abort (correct); the body diff is just that govi inserts a
		// literal '^' rather than implementing the ^^D autoindent-erase form.
		{"#30 ts=010", []string{"\tend"}, ":set ts=010\r", "leading-zero option = base 10 (cursor-only diff)"},
		{"#31 abbrev", nil, ":ab xx hello\rixx \x1b", "abbreviation expands"},
		{"#1 ^^D", nil, ":set ai\ri\t\r^\x04\x1b", "no abort on ^^D (body diff = unimplemented ^^D, not a crash)"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.issue, func(t *testing.T) {
			b, cur := run3(t, 12, 40, c)
			report3(t, c, b, cur)
		})
	}
}

// TestNviFix32 reproduces #32: a second @<buffer> fails. Root cause: m.reg leaks
// after a register-prefixed command, so the macro's own delete clobbers the
// macro register. TestNviFixRegLeak isolates the leak without a macro.
func TestNviFix32(t *testing.T) {
	c := threeCase{"#32 @a x3", []string{"x", "AAAA", "BBBB", "CCCC", "DDDD"}, "\"ayyj@a@a@a", "repeated @a"}
	b, cur := run3(t, 12, 40, c)
	report3(t, c, b, cur)
}

func TestNviFixRegLeak(t *testing.T) {
	// "ayy (reg a="yanked"); G to last line; x deletes a char (into reg a if
	// leaked); "aP puts reg a. nvi keeps reg a; govi clobbers it.
	c := threeCase{"#32 reg-leak", []string{"yanked", "AAAA", "BBBB"}, "\"ayyGx\"aP", "delete must not clobber reg a"}
	b, cur := run3(t, 12, 40, c)
	report3(t, c, b, cur)
}

// TestNviFixTags checks #15 (taglength) and #16 (anchored tag pattern).
func TestNviFixTags(t *testing.T) {
	paths := nvifixesEditors()
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("%s not available", p)
		}
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "code.txt")
	srcBody := "package x\nxtargetx is not it\ntarget\nfunc counter() {}\n"
	if err := os.WriteFile(src, []byte(srcBody), 0o644); err != nil {
		t.Fatal(err)
	}
	tagsPath := filepath.Join(dir, "tags")
	tagsBody := "anchored\t" + src + "\t/^target$/\ncounter\t" + src + "\t/^func counter/\n"
	if err := os.WriteFile(tagsPath, []byte(tagsBody), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(label, keys string) {
		cur := map[string][2]int{}
		line := map[string]string{}
		for name, path := range paths {
			tm := New(12, 40)
			if err := tm.Start(path, src); err != nil {
				t.Fatalf("start %s: %v", name, err)
			}
			if !tm.WaitFor(5*time.Second, func(d []string) bool { return strings.TrimSpace(d[len(d)-1]) != "" }) {
				tm.Close()
				t.Fatalf("%s did not load", name)
			}
			tm.WaitQuiet(200*time.Millisecond, 2*time.Second)
			tm.Send([]byte(":set tags=" + tagsPath + "\r"))
			tm.WaitQuiet(150*time.Millisecond, 2*time.Second)
			tm.Send([]byte(keys))
			tm.WaitQuiet(200*time.Millisecond, 3*time.Second)
			r, c := tm.Cursor()
			cur[name] = [2]int{r, c}
			d := tm.Dump()
			cr := r
			if cr < 0 || cr >= len(d) {
				cr = 0
			}
			line[name] = strings.TrimRight(d[cr], " ") + "  |stat:" + strings.TrimRight(d[len(d)-1], " ")
			tm.Send([]byte("\x1b:q!\r"))
			tm.Close()
		}
		verdict := "govi != fixed-nvi (POSSIBLE BUG)"
		if cur["govi"] == cur["fix-nvi"] {
			verdict = "govi == fixed-nvi (OK)"
		}
		t.Logf("[%s] %s\n   fix-nvi cur=%v %q\n   govi    cur=%v %q",
			label, verdict, cur["fix-nvi"], line["fix-nvi"], cur["govi"], line["govi"])
	}
	run("#16 anchored", ":tag anchored\r")           // OK: anchors honored
	run("#15 taglength", ":set taglength=4\r:tag countXXXX\r") // PRESENT: ignored
}
