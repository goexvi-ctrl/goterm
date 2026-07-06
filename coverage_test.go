package goterm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// coverage_test.go is a SYSTEMATIC sweep: it drives every vi/ex command and mode
// in docs/VI_EX_COMMANDS.md that is NOT an explicit non-objective, so we can say
// each has been exercised against the nvi oracle through the harness. It is a
// report (like the mining tests): a case logs `match` or `DIVERGE`, it does not
// fail. Commands that cannot be driven by this PTY model (signals, interactive
// shell, recovery, the editor-exiting quit family) are listed in
// TestCoverageManifest below with the reason. Explicit non-objectives (split
// screens, scripting, cscope, :open, the dead-option set) are intentionally
// absent.

// covLines is a small fixture with a blank line (paragraph boundary), an indented
// line (first-nonblank motions), punctuation, and repeated words (search).
func covLines() []string {
	return []string{
		"alpha beta gamma",
		"the quick brown fox",
		"  indented line here",
		"punct: foo, bar; baz.",
		"",
		"last line of file",
	}
}

// runCov runs one divCase and returns whether it diverged (delegates to the
// shared runner so behavior matches the other batteries).
func runCov(t *testing.T, c divCase) bool { return runDivCase(t, 24, 80, c) }

// TestCoverageViMotion covers the movement keys not already pinned by
// TestDivergeMotion: space $ ^ _ h j k l F T | + - } [[ ]] ^M ^N ^P ^H ^J.
func TestCoverageViMotion(t *testing.T) {
	s := covLines()
	cases := []divCase{
		{"space", s, "  ", "space moves right"},
		{"dollar", s, "$", "$ to end of line"},
		{"caret", s, "2G^", "^ to first nonblank"},
		{"underscore", s, "2G_", "_ first nonblank (line)"},
		{"h", s, "$h", "h left"},
		{"j", s, "j", "j down"},
		{"k", s, "Gk", "k up"},
		{"l", s, "l", "l right"},
		{"F", s, "$Fa", "F backward find char"},
		{"T", s, "$Ta", "T backward till char"},
		{"bar", s, "8|", "| to column 8"},
		{"plus", s, "+", "+ down to first nonblank"},
		{"minus", s, "G-", "- up to first nonblank"},
		{"brace-fwd", s, "}", "} forward paragraph (blank line)"},
		{"brace-back", s, "G{", "{ backward paragraph"},
		{"sec-back", s, "G[[", "[[ backward section (to BOF)"},
		{"sec-fwd", s, "]]", "]] forward section (to EOF)"},
		{"cr", s, "\r", "^M / CR down to first nonblank"},
		{"ctrl-n", s, "\x0e", "^N down"},
		{"ctrl-p", s, "j\x10", "^P up"},
		{"ctrl-h", s, "ll\x08", "^H left"},
		{"ctrl-j", s, "\n", "^J down"},
	}
	mined := 0
	for _, c := range cases {
		if runCov(t, c) {
			mined++
		}
	}
	t.Logf("cov-motion: %d/%d diverged", mined, len(cases))
}

// TestCoverageViEdit covers editing/operator keys not already pinned by
// TestDivergeEditing: C c$ s S Y yw i I a A o O R P U ~ ^A ^G, plus z screen
// positioning.
func TestCoverageViEdit(t *testing.T) {
	s := covLines()
	el := exMineLines() // repeated "foo" for ^A word search
	cases := []divCase{
		{"C", s, "wCzzz\x1b", "C change to end of line"},
		{"c-dollar", s, "wc$END\x1b", "c$ change to EOL"},
		{"s", s, "sZ\x1b", "s substitute char"},
		{"S", s, "SNEW\x1b", "S substitute line"},
		{"Y-p", s, "Yp", "Y yank line, p put"},
		{"yw-P", s, "ywwP", "yw yank word, P put before"},
		{"i", s, "iX\x1b", "i insert"},
		{"I", s, "2GIX\x1b", "I insert at first nonblank"},
		{"a", s, "aX\x1b", "a append"},
		{"A", s, "AX\x1b", "A append at EOL"},
		{"o", s, "oNEWLINE\x1b", "o open below"},
		{"O", s, "ONEWLINE\x1b", "O open above"},
		{"R", s, "RXYZ\x1b", "R replace mode"},
		{"P", s, "yyP", "P put line before"},
		{"U", s, "xxxU", "U restore line"},
		{"tilde", s, "3~", "~ toggle case x3"},
		{"ctrl-a", el, "\x01", "^A search word under cursor fwd"},
		{"ctrl-g", s, "\x07", "^G file info (status only; body/cursor unchanged)"},
	}
	mined := 0
	for _, c := range cases {
		if runCov(t, c) {
			mined++
		}
	}
	// z screen positioning needs a multi-screen file at a small height.
	zcases := []divCase{
		{"z-cr", numberedLines(40), "20Gz\r", "z<CR> current line to top"},
		{"z-dot", numberedLines(40), "20Gz.", "z. current line to center"},
		{"z-dash", numberedLines(40), "20Gz-", "z- current line to bottom"},
	}
	for _, c := range zcases {
		if runDivCase(t, 12, 40, c) {
			mined++
		}
	}
	t.Logf("cov-edit: %d/%d diverged", mined, len(cases)+len(zcases))
}

// TestCoverageViInput covers text-input-mode commands: ^@ replay, ^V quote,
// ^W word-erase, ^X hex (govi extension), autoindent carry, ^T/^D shift,
// abbreviations.
func TestCoverageViInput(t *testing.T) {
	s := covLines()
	cases := []divCase{
		{"replay", s, "ofoo\x1bo\x00\x1b", "^@ replay previous insert"},
		{"quote", s, "i\x16\x07\x1b", "^V quote literal ^G"},
		{"werase", s, "ifoo bar\x17\x1b", "^W erase word"},
		{"hex", s, "i\x18263a\x1b", "^X hex insert (govi extension; nvi differs)"},
		{"autoindent", s, ":set ai\rI    deep\rmore\x1b", "autoindent carry"},
		{"ctrl-t", s, ":set ai sw=4\ria\x14b\x1b", "^T shift in input"},
		{"abbrev", s, ":ab fb foobar\rofb \x1b", "abbreviation expands on word break"},
	}
	mined := 0
	for _, c := range cases {
		if runCov(t, c) {
			mined++
		}
	}
	t.Logf("cov-input: %d/%d diverged", mined, len(cases))
}

// TestCoverageEx covers ex commands not already pinned elsewhere: : < > = pu l
// ya co * a i c ab una map unmap args f r!cmd.
func TestCoverageEx(t *testing.T) {
	s := covLines()
	cases := []divCase{
		{"ex-lt", s, ":2,3>\r:2,3<\r", ":< shift left"},
		{"ex-gt", s, ":2,3>\r", ":> shift right"},
		{"ex-eq", s, ":3=\r", ":= line number (status)"},
		{"ex-put", s, "yy:3pu\r", ":pu put register after line"},
		{"ex-list", s, ":1,3l\r", ":l list"},
		{"ex-yank", s, ":1ya b\rG:pu b\r", ":ya yank to buffer + :pu"},
		{"ex-copy", s, ":1co3\r", ":co copy (synonym of :t)"},
		{"ex-at", []string{"d", "x y z"}, "\"ayyj:@a\r", ":@ execute buffer as ex cmds (#37)"},
		// NOTE: the :* form is intentionally NOT tested for parity: nvi's C_STAR
		// table entry carries no address flag, so a bare :* does not default the
		// address to the current line and an address-sensitive buffer command
		// (e.g. "d") fails with "address of 0". That is an arcane nvi quirk, like
		// the 2G0d( underflow; govi runs the buffer (matching the useful :@ form).
		{"ex-append", s, ":2a\rINSERTED\r.\r", ":a append input after line"},
		{"ex-insert", s, ":2i\rINSERTED\r.\r", ":i insert input before line"},
		{"ex-change", s, ":2c\rCHANGED\r.\r", ":c change line to input"},
		{"ex-abbrev", s, ":ab zz zebra\rozz \x1b", ":ab abbreviation"},
		{"ex-unabbrev", s, ":ab zz zebra\r:una zz\rozz \x1b", "#38 FIXED: :una resolves to :unabbreviate"},
		{"ex-unabbr-full", s, ":ab zz zebra\r:unabbreviate zz\rozz \x1b", "guard: full :unabbreviate works"},
		{"ex-map", s, ":map X 3x\rX", ":map then use mapping"},
		{"ex-unmap", s, ":map X 3x\r:unmap X\rX", ":unmap removes mapping"},
		{"ex-args", s, ":args\r", ":args list (status/overlay)"},
		{"ex-read-cmd", s, ":1r !echo INSERTED\r", ":r !cmd read command output"},
		{"ex-file", s, ":f\r", ":f file info (status)"},
		{"ex-amp-flag", s, ":1,3s/a/A/g\r", ":s with g flag (guard)"},
	}
	mined := 0
	for _, c := range cases {
		if runCov(t, c) {
			mined++
		}
	}
	t.Logf("cov-ex: %d/%d diverged", mined, len(cases))
}

// --- special-setup runners (multiple file args, tags, source) ---

// startArgs starts an editor with the given on-disk file args.
func startArgs(t *testing.T, path string, rows, cols int, args ...string) *Term {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("%s not available", path)
	}
	tm := New(rows, cols)
	if err := tm.Start(path, args...); err != nil {
		t.Fatalf("Start %s: %v", path, err)
	}
	return tm
}

// runArgs runs one scenario where both editors open the same explicit file args,
// reporting body+cursor divergence. Used for multi-file, tags, and source cases.
func runArgs(t *testing.T, name, desc, keys string, rows, cols int, args ...string) bool {
	t.Helper()
	// Both editors open the SAME fixture files. Since govi implements nvi's
	// advisory flock (#45), whichever editor starts second comes up read-only
	// behind the other's lock, stuck at a "Press any key" prompt that eats the
	// first byte of the keys. Locking itself is compared by TestCoverageLock
	// and the #45 assertions; everywhere else it is harness noise, so turn it
	// off for both editors only while this comparison's pair is starting.
	// (Not t.Setenv: that would stay set for the rest of the calling test.)
	oldExinit, hadExinit := os.LookupEnv("EXINIT")
	os.Setenv("EXINIT", "set nolock")
	defer func() {
		if hadExinit {
			os.Setenv("EXINIT", oldExinit)
		} else {
			os.Unsetenv("EXINIT")
		}
	}()
	nvi := startArgs(t, nviPath, rows, cols, args...)
	govi := startArgs(t, goviPath, rows, cols, args...)
	pair := editorPair{A: nvi, B: govi}
	defer pair.close()
	if !pair.waitReady(func(d []string) bool { return strings.TrimSpace(d[len(d)-1]) != "" }) {
		t.Fatalf("[%s] an editor did not become ready", name)
	}
	pair.settle()
	pair.send([]byte(keys))
	pair.settle()
	nb, gb := bodyOf(nvi.Dump()), bodyOf(govi.Dump())
	bodyDiffs := DiffScreens(nb, gb)
	nr, nc := nvi.Cursor()
	gr, gc := govi.Cursor()
	curMatch := nr == gr && nc == gc
	defer pair.send([]byte("\x1b:q!\r"))
	if len(bodyDiffs) == 0 && curMatch {
		t.Logf("[%-12s] match   (cursor %d,%d)  %s", name, nr, nc, desc)
		return false
	}
	t.Logf("[%-12s] DIVERGE  %s", name, desc)
	if !curMatch {
		t.Logf("    cursor: nvi=%d,%d govi=%d,%d", nr, nc, gr, gc)
	}
	if len(bodyDiffs) > 0 {
		t.Logf("    body (nvi | govi):\n%s", indent(FormatDiffs(bodyDiffs)))
	}
	return true
}

// tmpFile writes body to a uniquely-named temp file and returns its path.
func tmpFile(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestCoverageMultiFile covers the argument-list / alternate-file commands:
// :n :N/:prev :rew ^^ :e.
func TestCoverageMultiFile(t *testing.T) {
	f1 := tmpFile(t, "one.txt", "file one line A\nfile one line B\n")
	f2 := tmpFile(t, "two.txt", "file two line A\nfile two line B\n")
	mined := 0
	type mc struct{ name, desc, keys string }
	cases := []mc{
		{"next", ":n shows second file", ":n\r"},
		{"prev", ":n then :prev returns", ":n\r:prev\r"},
		{"rewind", ":n then :rew rewinds to first", ":n\r:rew\r"},
		{"alt", ":n then ^^ to alternate (first)", ":n\r\x1e"},
		{"edit", ":e second file", ":e " + f2 + "\r"},
		// The '.' dot buffer must survive a file switch (nvi v_init.c: "User
		// can replay the last input"). dw in file one, :n! past the unsaved
		// change, then '.' repeats dw in file two. :n! (not :n) because the dw
		// leaves file one modified and a plain :n would refuse the switch.
		{"dot-next", "dw, :n!, . repeats dw in next file", "dw:n!\r."},
	}
	for _, c := range cases {
		if runArgs(t, c.name, c.desc, c.keys, 24, 80, f1, f2) {
			mined++
		}
	}
	t.Logf("cov-multifile: %d/%d diverged", mined, len(cases))
}

// TestCoverageTags covers tag navigation: :tag, ^] (push), ^T (pop). A small
// ctags-format tags file points at symbols in a source file; tags= is set so the
// lookup does not depend on cwd.
func TestCoverageTags(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	body := "alpha here\nbeta here\ngamma here\ndelta here\n"
	if err := os.WriteFile(src, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	// ctags format: {tag}\t{file}\t{excmd}
	tags := filepath.Join(dir, "tags")
	tbody := "beta\t" + src + "\t/^beta here$/\n" +
		"delta\t" + src + "\t/^delta here$/\n"
	if err := os.WriteFile(tags, []byte(tbody), 0o644); err != nil {
		t.Fatal(err)
	}
	setT := ":set tags=" + tags + "\r"
	mined := 0
	type tc struct{ name, desc, keys string }
	cases := []tc{
		{"tag-cmd", ":tag jumps to symbol", setT + ":tag delta\r"},
		{"tag-push", "^] pushes tag for word under cursor", setT + ":tag beta\r0\x1d"},
		{"tag-pop", "^T pops back", setT + ":tag beta\r:tag delta\r\x14"},
	}
	for _, c := range cases {
		if runArgs(t, c.name, c.desc, c.keys, 24, 80, src) {
			mined++
		}
	}
	t.Logf("cov-tags: %d/%d diverged", mined, len(cases))
}

// TestCoverageSource covers :so[urce] -- reading ex commands from a file. Both
// the plain form and the leading-colon form (#39, FIXED 2026-06-28) now match
// nvi; kept as regression guards.
func TestCoverageSource(t *testing.T) {
	src := func() string { return tmpFile(t, "buf.txt", "alpha\nbeta\ngamma\n") }
	plain := tmpFile(t, "plain.ex", "%s/a/A/g\n2d\n")
	colon := tmpFile(t, "colon.ex", ":%s/a/A/g\n:2d\n")
	mined := 0
	if runArgs(t, "so-plain", ":so runs ex cmds (guard)", ":so "+plain+"\r", 24, 80, src()) {
		mined++
	}
	if runArgs(t, "so-colon", "#39 FIXED: :so tolerates leading colon", ":so "+colon+"\r", 24, 80, src()) {
		mined++
	}
	t.Logf("cov-source: %d/2 diverged (#39 fixed; guards)", mined)
}

// TestCoverageRecentFixes guards the small-item fixes #40-#42 (2026-06-28), all
// verified matching nvi.
func TestCoverageRecentFixes(t *testing.T) {
	// #40 vi z+/z^ screen types, #41 secure gating -- lockstep guards.
	cases := []divCase{
		{"z-plus", numberedLines(40), "20Gz+", "#40 z+ scrolls a screen forward"},
		{"z-caret", numberedLines(40), "20Gz^", "#40 z^ scrolls a screen back"},
	}
	for _, c := range cases {
		runDivCase(t, 12, 40, c)
	}
	runDivCase(t, 24, 80, divCase{"secure-filter", covLines(), ":set secure\r:%!sort\r", "#41 secure blocks ! filter"})
	// #41 correction: :source is NOT secured in nvi -- it must still run.
	src := tmpFile(t, "sbuf.txt", "alpha\nbeta\ngamma\n")
	cmds := tmpFile(t, "s.ex", "%s/a/A/g\n")
	runArgs(t, "secure-source", "#41 :source still runs under secure", ":set secure\r:so "+cmds+"\r", 24, 80, src)

	// #42 overwrite guard -- disk check (not a screen diff): :w of an existing,
	// non-buffer file must REFUSE without ! and obey writeany. Compared vs nvi.
	overwrite := func(bin string, keysTail func(string) string) string {
		d := t.TempDir()
		a := filepath.Join(d, "a.txt")
		b := filepath.Join(d, "b.txt")
		os.WriteFile(a, []byte("AAA\n"), 0o644)
		os.WriteFile(b, []byte("BBB\n"), 0o644)
		tm := New(24, 80)
		if err := tm.Start(bin, a); err != nil {
			t.Fatal(err)
		}
		tm.WaitFor(5*time.Second, func(dd []string) bool { return len(dd) > 0 && dd[len(dd)-1] != "" })
		tm.WaitQuiet(300*time.Millisecond, 3*time.Second)
		tm.Send([]byte(keysTail(b)))
		tm.WaitQuiet(400*time.Millisecond, 4*time.Second)
		tm.Send([]byte("\x1b:q!\r"))
		tm.Close()
		bc, _ := os.ReadFile(b)
		return string(bc)
	}
	for _, ed := range []struct{ n, bin string }{{"nvi", nviPath}, {"govi", goviPath}} {
		if _, err := os.Stat(ed.bin); err != nil {
			t.Skipf("%s missing", ed.bin)
		}
		noforce := overwrite(ed.bin, func(b string) string { return ":w " + b + "\r" })
		force := overwrite(ed.bin, func(b string) string { return ":w! " + b + "\r" })
		wa := overwrite(ed.bin, func(b string) string { return ":set wa\r:w " + b + "\r" })
		if noforce != "BBB\n" {
			t.Errorf("[%s] :w existing file should REFUSE (data loss!); b=%q", ed.n, noforce)
		}
		if force != "AAA\n" {
			t.Errorf("[%s] :w! should overwrite; b=%q", ed.n, force)
		}
		if wa != "AAA\n" {
			t.Errorf("[%s] :set wa + :w should overwrite; b=%q", ed.n, wa)
		}
		t.Logf("[%-4s] #42 overwrite guard: noforce=%q force=%q wa=%q", ed.n, noforce, force, wa)
	}

	// #45 readonly write-guard -- :set ro then edit then :w must REFUSE (disk
	// unchanged) without ! in both editors.
	roWrite := func(bin string) string {
		d := t.TempDir()
		f := filepath.Join(d, "ro.txt")
		os.WriteFile(f, []byte("orig\n"), 0o644)
		tm := New(24, 80)
		if err := tm.Start(bin, f); err != nil {
			t.Fatal(err)
		}
		tm.WaitFor(5*time.Second, func(dd []string) bool { return len(dd) > 0 && dd[len(dd)-1] != "" })
		tm.WaitQuiet(300*time.Millisecond, 3*time.Second)
		tm.Send([]byte(":set ro\roADD\x1b:w\r"))
		tm.WaitQuiet(400*time.Millisecond, 4*time.Second)
		tm.Send([]byte("\x1b:q!\r"))
		tm.Close()
		b, _ := os.ReadFile(f)
		return string(b)
	}
	// #45 lock -- instance 1 (bin1) holds the lock; instance 2 (bin2) must open
	// read-only and its no-force :w must REFUSE (disk keeps instance-1 state).
	// Cross-process because flock is advisory and shared.
	lockSecond := func(bin1, bin2 string) string {
		d := t.TempDir()
		f := filepath.Join(d, "L.txt")
		os.WriteFile(f, []byte("shared\n"), 0o644)
		t1 := New(24, 80)
		if err := t1.Start(bin1, f); err != nil {
			t.Fatal(err)
		}
		t1.WaitFor(5*time.Second, func(dd []string) bool { return len(dd) > 0 && dd[len(dd)-1] != "" })
		t1.WaitQuiet(300*time.Millisecond, 3*time.Second)
		t2 := New(24, 80)
		if err := t2.Start(bin2, f); err != nil {
			t.Fatal(err)
		}
		t2.WaitFor(5*time.Second, func(dd []string) bool { return len(dd) > 0 && dd[len(dd)-1] != "" })
		t2.WaitQuiet(300*time.Millisecond, 3*time.Second)
		t2.Send([]byte("oSECOND\x1b:w\r"))
		t2.WaitQuiet(400*time.Millisecond, 4*time.Second)
		t2.Send([]byte("\x1b:q!\r"))
		t2.Close()
		t1.Send([]byte("\x1b:q!\r"))
		t1.Close()
		b, _ := os.ReadFile(f)
		return string(b)
	}
	for _, ed := range []struct{ n, bin string }{{"nvi", nviPath}, {"govi", goviPath}} {
		if _, err := os.Stat(ed.bin); err != nil {
			t.Skipf("%s missing", ed.bin)
		}
		if ro := roWrite(ed.bin); ro != "orig\n" {
			t.Errorf("[%s] #45 :set ro + :w should REFUSE; disk=%q", ed.n, ro)
		}
		if l := lockSecond(ed.bin, ed.bin); l != "shared\n" {
			t.Errorf("[%s] #45 second instance :w should REFUSE (locked); disk=%q", ed.n, l)
		}
	}
	// #45 cross-process interop: nvi locks, govi must come up read-only (and vice
	// versa). Only meaningful with both binaries present.
	if _, e1 := os.Stat(nviPath); e1 == nil {
		if _, e2 := os.Stat(goviPath); e2 == nil {
			if l := lockSecond(nviPath, goviPath); l != "shared\n" {
				t.Errorf("#45 govi behind nvi lock should REFUSE; disk=%q", l)
			}
			if l := lockSecond(goviPath, nviPath); l != "shared\n" {
				t.Errorf("#45 nvi behind govi lock should REFUSE; disk=%q", l)
			}
		}
	}
}

// TestCoverageLock guards #45: the `lock` option (concurrent-edit protection).
// A second instance on an already-open file must come up read-only and refuse a
// plain write; the lock must survive a self-write (relock after temp+rename) and
// be released on close. Cross-process advisory flock means it interoperates with
// nvi, so these are asserted (not just logged).
func TestCoverageLock(t *testing.T) {
	for _, bin := range []string{nviPath, goviPath} {
		if _, err := os.Stat(bin); err != nil {
			t.Skipf("%s missing", bin)
		}
	}
	openLock := func(bin, f string) *Term {
		tm := New(24, 80)
		if err := tm.Start(bin, f); err != nil {
			t.Fatal(err)
		}
		tm.WaitFor(5*time.Second, func(d []string) bool { return len(d) > 0 && strings.TrimSpace(d[len(d)-1]) != "" })
		tm.WaitQuiet(300*time.Millisecond, 3*time.Second)
		return tm
	}
	// isRO reports whether a plain :w on the unmodified buffer is refused.
	isRO := func(tm *Term) bool {
		tm.Send([]byte(" ")) // dismiss nvi "press any key" pager (harmless in govi)
		tm.WaitQuiet(250*time.Millisecond, 2*time.Second)
		tm.Send([]byte(":w\r"))
		tm.WaitQuiet(350*time.Millisecond, 3*time.Second)
		d := tm.Dump()
		return strings.Contains(strings.ToLower(strings.TrimSpace(d[len(d)-1])), "read-only")
	}
	mkfile := func(body string) string {
		p := filepath.Join(t.TempDir(), "shared.txt")
		os.WriteFile(p, []byte(body), 0o644)
		return p
	}

	// holder/second binary pairs: same-process and both cross-process directions.
	for _, p := range []struct{ holder, second string }{
		{goviPath, goviPath}, {nviPath, goviPath}, {goviPath, nviPath},
	} {
		f := mkfile("hello world\n")
		h := openLock(p.holder, f)
		s := openLock(p.second, f)
		if !isRO(s) {
			t.Errorf("lock: holder=%s second=%s -> second NOT read-only (concurrent-edit guard failed)", p.holder, p.second)
		}
		s.Send([]byte("\x1b:q!\r"))
		s.Close()
		h.Send([]byte("\x1b:q!\r"))
		h.Close()
	}

	// self-write must KEEP the lock (govi relock after temp+rename).
	f := mkfile("hello world\n")
	h := openLock(goviPath, f)
	h.Send([]byte("x:w\r"))
	h.WaitQuiet(400*time.Millisecond, 4*time.Second)
	s := openLock(goviPath, f)
	if !isRO(s) {
		t.Errorf("lock: second instance not read-only after holder self-write (relock lost the lock)")
	}
	s.Send([]byte("\x1b:q!\r"))
	s.Close()
	h.Send([]byte("\x1b:q!\r"))
	h.Close()

	// lock released on close: a fresh instance is writable.
	f = mkfile("hello world\n")
	a := openLock(goviPath, f)
	a.Send([]byte(":q!\r"))
	a.Close()
	b := openLock(goviPath, f)
	if isRO(b) {
		t.Errorf("lock: fresh instance read-only after holder closed (lock not released)")
	}
	b.Send([]byte("\x1b:q!\r"))
	b.Close()
}

// TestCoverageManifest documents the commands/modes that are real (not
// non-objectives) but cannot be driven by this lockstep-PTY comparison model, so
// the coverage claim is honest about its boundary. These are validated elsewhere
// (govi internal/conformance, or by manual/interactive check) or are inherently
// terminal-lifecycle actions.
func TestCoverageManifest(t *testing.T) {
	notHarnessable := map[string]string{
		"ZZ / ZQ / :q / :wq / :x":  "exit the editor; the PTY closes -- exercised by the dirty-guard write tests, not a screen diff",
		"^Z / :suspend / :stop":    "raise SIGTSTP / job control; no controlling shell in the harness",
		"^C / <interrupt>":         "delivers SIGINT; timing-dependent, not a stable screen diff",
		":shell / :sh":             "spawns an interactive child shell; no stdin script in the harness",
		":preserve / :recover":     "crash-recovery subsystem; needs a separate -r restart cycle",
		":mkexrc":                  "writes an .exrc to disk; a filesystem effect, not a screen diff",
		"^L / ^R":                  "repaint; govi terminal repaints every input so the diff is always empty (cosmetic)",
		"^V/^X in :version etc.":   "version/usage text is intentionally editor-specific (build metadata)",
	}
	for cmd, why := range notHarnessable {
		t.Logf("NOT-HARNESSABLE  %-26s  %s", cmd, why)
	}
	excluded := []string{
		":open, :perl, :perldo, :tcl, :script (scripting / out of scope)",
		"lisp, redraw, slowopen, optimize, modeline, sourceany (dead/security options)",
	}
	covered := []string{
		"^W, :bg, :fg, :resize, :vsplit, :E -- TestParitySplits (implemented 2026-06)",
		":cscope add/find -- TestParityCscope (against the real cscope binary)",
	}
	for _, e := range covered {
		t.Logf("NOW-COVERED      %s", e)
	}
	for _, e := range excluded {
		t.Logf("EXCLUDED         %s", e)
	}
	_ = time.Second
}
