package goterm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// parityreview_test.go probes the govi/docs/parity.md rows that no existing
// battery exercises (see the 2026-07 parity review). Like the other mining
// batteries it is a REPORT: cases log `match` or `DIVERGE` and do not fail.
// Cases whose desc says "expect DIVERGE" confirm a documented gap; a `match`
// there means parity.md overstates the gap.

// prLines is the shared small fixture.
func prLines() []string {
	return []string{
		"alpha beta gamma",
		"the quick brown fox",
		"  indented line here",
		"punct: foo, bar; baz.",
		"last line of file",
	}
}

// TestParityViKeys covers vi-mode keys with no battery case: escape-cancel of
// partial commands, the ^\ ex switch, Q ex switch, and the wrapped-line ^E/^Y
// gap (#44).
func TestParityViKeys(t *testing.T) {
	s := prLines()
	cases := []divCase{
		// ^[ cancels a pending operator/register/count (parity.md ^[ row).
		{"esc-operator", s, "d\x1bw", "d<ESC>w: ESC cancels the pending d; w just moves"},
		{"esc-count", s, "5\x1bx", "5<ESC>x: ESC discards the count; x deletes one char"},
		{"esc-count-op", s, "2d\x1bw", "2d<ESC>w: ESC cancels count+operator"},
		{"esc-register", s, "\"a\x1bx", "\"a<ESC>x: ESC cancels the pending register"},
		// Unbound-in-nvi keys (parity.md footnote): each must be a bell/no-op
		// in BOTH editors, leaving screen and cursor unchanged.
		{"unbound-g", s, "jg", "g is not a vi command in nvi 1.81"},
		{"unbound-K", s, "jK", "K is unbound"},
		{"unbound-v", s, "jv", "v is unbound (no visual mode)"},
		{"unbound-V", s, "jV", "V is unbound"},
		{"unbound-ctrl-o", s, "j\x0f", "^O is unbound"},
		{"unbound-equals", s, "j=", "= is unbound"},
		{"unbound-ctrl-us", s, "j\x1f", "^_ is unbound"},
	}
	div := 0
	for _, c := range cases {
		if runDivCase(t, 24, 80, c) {
			div++
		}
	}
	t.Logf("parity-vikeys: %d/%d diverged", div, len(cases))
}

// TestParityWrappedScroll confirms the documented #44 gap: ^E/^Y scroll by
// logical line over a WRAPPED line where nvi scrolls by screen row. parity.md
// marks ^E/^Y as partial for exactly this; expect DIVERGE here.
func TestParityWrappedScroll(t *testing.T) {
	long := strings.Repeat("wrap ", 30) // ~150 chars: 4 rows at 40 cols
	file := []string{long, "second", "third", "fourth", "fifth", "sixth",
		"seventh", "eighth", "ninth", "tenth"}
	cases := []divCase{
		{"ctrl-e-wrapped", file, "\x05", "expect DIVERGE #44: ^E over a wrapped top line"},
		{"ctrl-y-wrapped", file, "8G\x19", "expect DIVERGE #44: ^Y toward a wrapped line"},
	}
	div := 0
	for _, c := range cases {
		if runDivCase(t, 12, 40, c) {
			div++
		}
	}
	t.Logf("parity-wrapscroll: %d/%d diverged (2/2 = documented gap confirmed)", div, len(cases))
}

// TestParitySplits covers the split-screen family parity.md marks done but no
// battery has ever driven: :E (horizontal), :vsplit, ^W switching, :resize,
// :bg/:fg. Each case ends in a state whose BODY (both panes + divider status
// lines are body rows for the non-final panes) must match nvi's.
func TestParitySplits(t *testing.T) {
	// A split's internal status row lands in the compared body and shows the
	// file path, so both editors must open the SAME file (runArgs handles the
	// lock conflict), not per-editor copies whose temp dirs differ.
	f := tmpFile(t, "buf.txt", strings.Join(prLines(), "\n")+"\n")
	type sc struct{ name, keys, desc string }
	cases := []sc{
		{"split-E", ":E\r", ":E opens a second (horizontal) screen on the same file"},
		{"split-vs", ":vsplit\r", ":vsplit opens a vertical split"},
		{"split-switch", ":E\r\x17j", "^W switches to the next screen; j moves there"},
		{"split-resize", ":E\r:resize +2\r", ":resize grows the current split"},
		{"split-bg-fg", ":E\r:bg\r", ":bg hides the current screen"},
		{"split-fg", ":E\r:bg\r:fg\r", ":fg brings the backgrounded screen back"},
		{"split-quit", ":E\r:q\r", "closing a split returns to one screen"},
	}
	div := 0
	for _, c := range cases {
		if runArgs(t, c.name, c.desc, c.keys, 24, 80, f) {
			div++
		}
	}
	t.Logf("parity-splits: %d/%d diverged", div, len(cases))
}

// TestParityExSwitch covers the vi<->ex mode switches: Q, ^\, and :vi. Driven
// one step per send (the ex reader does not consume bursts; see the catalog's
// EX-MODE METHOD note).
func TestParityExSwitch(t *testing.T) {
	steps := func(name, desc string, stepList ...string) (string, []string, string) {
		return name, stepList, desc
	}
	type sc struct {
		name  string
		steps []string
		desc  string
	}
	var cases []sc
	add := func(name string, desc string, stepList ...string) {
		cases = append(cases, sc{name, stepList, desc})
	}
	_ = steps
	add("Q-print", "Q enters ex mode; :p prints the current line",
		"Q", "p\r")
	add("Q-back", "Q then vi returns to the full-screen display",
		"Q", "vi\r")
	add("ctrl-backslash", "^\\ switches to ex mode like Q",
		"\x1c", "p\r")
	div := 0
	for _, c := range cases {
		if runSeqCase(t, 24, 80, seqCase{c.name, prLines(), c.steps, c.desc}) {
			div++
		}
	}
	t.Logf("parity-exswitch: %d/%d diverged", div, len(cases))
	_ = time.Second
}

// TestParityExCmds covers ex commands with no battery case: the ex ^D scroll
// (parity.md marks it missing -- expect DIVERGE), :cd, :ex as an :edit alias,
// :r file, the tag-stack trio, :wn, and vi-mode :visual file.
func TestParityExCmds(t *testing.T) {
	fdir := shortDir(t)
	f1 := tmpFileAt(t, fdir, "one.txt", "file one line A\nfile one line B\n")
	f2 := tmpFileAt(t, fdir, "two.txt", "file two line A\nfile two line B\n")
	rd := tmpFileAt(t, fdir, "read-me.txt", "READ CONTENT\n")

	// Ex-mode ^D scroll: one step per send (ex reader does not take bursts).
	if runSeqCase(t, 12, 80, seqCase{"ex-ctrl-d", numberedLines(40),
		[]string{"Q", "\x04"}, "expect DIVERGE: ex ^D scroll is not implemented in govi"}) {
		t.Log("parity-excmds: ex-ctrl-d diverged (documented gap)")
	}

	type sc struct{ name, keys, desc string }
	cases := []sc{
		{"ex-alias-edit", ":ex " + f2 + "\r", ":ex file behaves as :edit file"},
		{"visual-file", ":visual " + f2 + "\r", "vi-mode :visual file edits the file"},
		{"read-file", ":r " + rd + "\r", ":r file inserts its lines after the current line"},
	}
	div := 0
	for _, c := range cases {
		if runArgs(t, c.name, c.desc, c.keys, 24, 80, f1, f2) {
			div++
		}
	}
	// :wn WRITES the current file, so each editor needs its own copy. The
	// write+switch queues two nvi messages, which always opens the message
	// overlay; the trailing "0" dismisses its continue prompt (and is a
	// harmless cursor motion once both editors show the next file).
	if runSepCase(t, "wn", ":wn writes and moves to the next file", "Ox\x1b:wn\r0", 24, 80,
		[]prFixture{{"one.txt", "file one line A\n"}, {"two.txt", "file two line A\n"}}) {
		div++
	}

	// :cd -- verified via a relative :e after changing directory.
	cdDir := t.TempDir()
	tmpFileAt(t, cdDir, "rel.txt", "RELATIVE FILE\n")
	if runArgs(t, "cd-relative", ":cd then :e resolves relative to the new cwd",
		":cd "+cdDir+"\r:e rel.txt\r", 24, 80, f1) {
		div++
	}

	// Tag stack: two entries for the same tag exercise :tagnext/:tagprev/:tagtop.
	tagDir := t.TempDir()
	src := tmpFileAt(t, tagDir, "src.txt",
		"alpha one\nsym first here\nfiller\nsym second here\nomega\n")
	tags := tmpFileAt(t, tagDir, "tags",
		"sym\t"+src+"\t/^sym first here$/\n"+
			"sym\t"+src+"\t/^sym second here$/\n")
	setT := ":set tags=" + tags + "\r"
	tcases := []sc{
		{"tagnext", setT + ":tag sym\r:tagnext\r", ":tagnext moves to the second match"},
		{"tagprev", setT + ":tag sym\r:tagnext\r:tagprev\r", ":tagprev returns to the first"},
		{"tagtop", setT + ":tag sym\r:tagnext\r:tagtop\r", ":tagtop pops the whole stack"},
	}
	for _, c := range tcases {
		if runArgs(t, c.name, c.desc, c.keys, 24, 80, src) {
			div++
		}
	}
	t.Logf("parity-excmds: %d/%d diverged", div, len(cases)+1+len(tcases))
}

// tmpFileAt writes body to dir/name and returns the path.
func tmpFileAt(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// shortDir returns a fresh temp directory with a SHORT path (directly under
// /tmp). Several nvi messages include the file path; t.TempDir()'s long path
// pushes such a message past one screen line, which triggers nvi's multi-line
// message overlay (a known display divergence) and hides the behavior a probe
// is actually testing. A short path keeps the message on the dropped status
// row, so the body diff sees only the behavior.
func shortDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("/tmp", "prv")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}

// prFixture is one on-disk fixture for runSepCase.
type prFixture struct{ name, body string }

// runSepCase starts each editor on ITS OWN copy of the same fixtures (in
// short-pathed dirs; see shortDir) so a probe that WRITES a file cannot make
// the editors collide on disk ("file modified more recently than this copy").
// keys must not embed fixture paths. Diffs body+cursor like runDivCase.
func runSepCase(t *testing.T, name, desc, keys string, rows, cols int, fx []prFixture) bool {
	t.Helper()
	mk := func() []string {
		d := shortDir(t)
		args := make([]string, 0, len(fx))
		for _, f := range fx {
			args = append(args, tmpFileAt(t, d, f.name, f.body))
		}
		return args
	}
	nvi := startArgs(t, nviPath, rows, cols, mk()...)
	govi := startArgs(t, goviPath, rows, cols, mk()...)
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
	defer pair.send([]byte("\x1b:q!\r"))
	if len(bodyDiffs) == 0 && nr == gr && nc == gc {
		t.Logf("[%-12s] match   (cursor %d,%d)  %s", name, nr, nc, desc)
		return false
	}
	t.Logf("[%-12s] DIVERGE  %s", name, desc)
	if nr != gr || nc != gc {
		t.Logf("    cursor: nvi=%d,%d govi=%d,%d", nr, nc, gr, gc)
	}
	if len(bodyDiffs) > 0 {
		t.Logf("    body (nvi | govi):\n%s", indent(FormatDiffs(bodyDiffs)))
	}
	return true
}

// TestParityOptions probes the functional (parity.md checkmarked) options that
// no battery exercises, plus inertness spot-checks for options parity.md marks
// settable-but-inert (those say "expect DIVERGE"; a match means the option
// actually works or the probe is inert -- either way parity.md needs a look).
func TestParityOptions(t *testing.T) {
	s := prLines()
	magicLines := []string{"fox line here", "literal f.x here", "tail"}
	searchLines := []string{"first line", "target line", "third line"}
	cases := []divCase{
		// Functional claims.
		{"ignorecase", s, ":set ic\r/QUICK\rx", "ic finds lower-case quick"},
		{"noignorecase", s, "/QUICK\rx", "guard: without ic, /QUICK fails; x hits line 1"},
		{"magic", magicLines, "/f.x\rx", "guard: magic . matches fox first"},
		{"nomagic", magicLines, ":set nomagic\r/f.x\rx", "nomagic makes . literal; matches f.x line"},
		{"tildeop", s, ":set tildeop\r~w", "tildeop turns ~ into an operator"},
		{"wrapscan-off", searchLines, "/target\r:set nows\rnx", "nows: n past the last match fails; x edits the same line"},
		{"window", numberedLines(40), ":set window=6\r\x06", "window=6 changes the ^F page distance"},
		{"shell-filter", s, ":set shell=/bin/sh\r:%!sort\r", "shell= drives the filter's shell; body shows the sort"},
		// Inertness spot-checks (parity.md marks these settable but inert).
		{"scroll-inert", numberedLines(40), ":set scroll=3\r\x04", "expect DIVERGE: nvi honors scroll= for ^D; govi uses half-page"},
		{"paragraphs-inert", []string{"one", ".QQ", "two", "three"}, ":set paragraphs=QQ\r}x", "expect DIVERGE: nvi honors paragraphs=QQ; govi uses built-ins"},
		{"sections-inert", []string{"one", ".QQ", "two", "three"}, ":set sections=QQ\r]]x", "expect DIVERGE: nvi honors sections=QQ"},
		{"extended-inert", []string{"baaa here", "a+ literal"}, ":set extended\r/a+\rx", "expect DIVERGE: nvi honors extended (ERE) search"},
		{"remap-nonrecursive", s, ":map X Y\r:map Y dd\rX", "expect DIVERGE: nvi remaps X->Y->dd; govi sends Y literally"},
		{"leftright-inert", []string{strings.Repeat("x", 60) + "END", "second"}, ":set leftright\r$", "expect DIVERGE at 40 cols: nvi shifts the view; govi wraps"},
		// Bulk: the full option table.
		{"set-all", s, ":set all\r", ":set all layout, names, and defaults"},
	}
	div := 0
	for _, c := range cases {
		rows, cols := 24, 80
		if c.name == "leftright-inert" {
			rows, cols = 12, 40
		}
		if runDivCase(t, rows, cols, c) {
			div++
		}
	}
	t.Logf("parity-options: %d/%d diverged", div, len(cases))
}

// TestParityAutowrite: with aw set, :n on a modified buffer writes instead of
// erroring, so both editors reach the second file.
func TestParityAutowrite(t *testing.T) {
	fx := []prFixture{{"one.txt", "file one line A\n"}, {"two.txt", "file two line A\n"}}
	div := 0
	// aw writes the current file on :n, so per-editor copies; the trailing
	// "0" dismisses nvi's two-message overlay prompt (see the :wn case).
	if runSepCase(t, "autowrite", ":set aw makes :n write the modified buffer",
		"Oxx\x1b:set aw\r:n\r0", 24, 80, fx) {
		div++
	}
	if runSepCase(t, "no-autowrite", "guard: without aw, :n on a modified buffer fails",
		"Oxx\x1b:n\r", 24, 80, fx) {
		div++
	}
	t.Logf("parity-autowrite: %d/2 diverged", div)
}

// runStatusCase is runDivCase including the status row: for options whose
// whole effect IS the status line (ruler, showmode), the usual dropped-row
// comparison would see nothing.
func runStatusCase(t *testing.T, rows, cols int, c divCase) bool {
	t.Helper()
	nvi := New(rows, cols)
	govi := New(rows, cols)
	if err := nvi.Start(nviPath, writeLines(t, c.file)); err != nil {
		t.Fatal(err)
	}
	defer nvi.Close()
	if err := govi.Start(goviPath, writeLines(t, c.file)); err != nil {
		t.Fatal(err)
	}
	defer govi.Close()
	pair := editorPair{A: nvi, B: govi}
	if !pair.waitReady(func(d []string) bool { return strings.TrimSpace(d[len(d)-1]) != "" }) {
		t.Fatalf("[%s] an editor did not become ready", c.name)
	}
	pair.settle()
	pair.send([]byte(c.keys))
	pair.settle()
	diffs := DiffScreens(nvi.Dump(), govi.Dump())
	nr, nc := nvi.Cursor()
	gr, gc := govi.Cursor()
	if len(diffs) == 0 && nr == gr && nc == gc {
		t.Logf("[%-12s] match   (cursor %d,%d)  %s", c.name, nr, nc, c.desc)
		return false
	}
	t.Logf("[%-12s] DIVERGE  %s", c.name, c.desc)
	if nr != gr || nc != gc {
		t.Logf("    cursor: nvi=%d,%d govi=%d,%d", nr, nc, gr, gc)
	}
	if len(diffs) > 0 {
		t.Logf("    full screen (nvi | govi):\n%s", indent(FormatDiffs(diffs)))
	}
	return true
}

// TestParityStatusOptions probes ruler and showmode, whose effect lives on the
// status row (compared here, unlike the body-only batteries).
func TestParityStatusOptions(t *testing.T) {
	s := prLines()
	cases := []divCase{
		{"ruler", s, ":set ruler\rjll", "ruler shows row,col on the status line"},
		{"showmode", s, ":set showmode\ri", "showmode names the mode on the status line"},
	}
	div := 0
	for _, c := range cases {
		if runStatusCase(t, 24, 80, c) {
			div++
		}
	}
	t.Logf("parity-status: %d/%d diverged", div, len(cases))
}

// TestParityDisplay probes the :display subcommands beyond the roster's
// buffers case: screens (with a split open) and tags (with a pushed tag).
func TestParityDisplay(t *testing.T) {
	dir := shortDir(t)
	src := tmpFileAt(t, dir, "src.txt", "alpha here\nbeta here\ngamma\n")
	tags := tmpFileAt(t, dir, "tags", "beta\t"+src+"\t/^beta here$/\n")
	type sc struct{ name, keys, desc string }
	cases := []sc{
		{"display-screens", ":E\r:display screens\r", ":display screens lists the split screens"},
		{"display-tags", ":set tags=" + tags + "\r:tag beta\r:display tags\r", ":display tags shows the tag stack"},
	}
	div := 0
	for _, c := range cases {
		if runArgs(t, c.name, c.desc, c.keys, 24, 80, src) {
			div++
		}
	}
	t.Logf("parity-display: %d/%d diverged", div, len(cases))
}

// TestParityFilec: with filec set to the same character explicitly, both
// editors complete a unique file-name prefix on the colon line.
func TestParityFilec(t *testing.T) {
	dir := shortDir(t)
	tmpFileAt(t, dir, "uniqname.txt", "COMPLETED FILE\n")
	f := tmpFileAt(t, dir, "start.txt", "start file\n")
	// ^V quotes the <tab> in the :set value; the second <tab> triggers
	// completion of the unique prefix.
	keys := ":set filec=\x16\t\r:e " + dir + "/uniq\t\r"
	div := 0
	if runArgs(t, "filec", "filec=<tab> completes :e file names", keys, 24, 80, f) {
		div++
	}
	t.Logf("parity-filec: %d/1 diverged", div)
}

// TestParityCscope drives the cscope integration (parity.md marks it done;
// the coverage manifest still lists it as excluded) against the real cscope.
func TestParityCscope(t *testing.T) {
	if _, err := os.Stat("/opt/homebrew/bin/cscope"); err != nil {
		t.Skip("cscope not installed")
	}
	dir := shortDir(t)
	src := tmpFileAt(t, dir, "code.c",
		"int helper(void) { return 1; }\n\nint main(void) { return helper(); }\n")
	tmpFileAt(t, dir, "cscope.files", src+"\n")
	type sc struct{ name, keys, desc string }
	cases := []sc{
		{"cs-add-find", ":cscope add " + dir + "\r:cscope find g helper\r",
			"cs add + find g jumps to the definition"},
	}
	div := 0
	for _, c := range cases {
		if runArgs(t, c.name, c.desc, c.keys, 24, 80, src) {
			div++
		}
	}
	t.Logf("parity-cscope: %d/%d diverged", div, len(cases))
}

// TestParityFSEffects checks commands whose effect is on disk, editor by
// editor: :mkexrc (parity.md marks it missing in govi) and :preserve into a
// :set recdir directory (parity.md marks both functional).
func TestParityFSEffects(t *testing.T) {
	run := func(bin string, env []string, file string, keys string) {
		t.Helper()
		tm := New(24, 80)
		for _, kv := range env {
			i := strings.IndexByte(kv, '=')
			t.Setenv(kv[:i], kv[i+1:])
		}
		if err := tm.Start(bin, file); err != nil {
			t.Fatal(err)
		}
		defer tm.Close()
		tm.WaitFor(5*time.Second, func(d []string) bool {
			return strings.TrimSpace(d[len(d)-1]) != ""
		})
		tm.WaitQuiet(200*time.Millisecond, 2*time.Second)
		tm.Send([]byte(keys))
		tm.WaitQuiet(300*time.Millisecond, 3*time.Second)
		tm.Send([]byte("\x1b:q!\r"))
		tm.WaitQuiet(200*time.Millisecond, 2*time.Second)
	}
	names := map[string]string{nviPath: "nvi", goviPath: "govi"}

	// :mkexrc
	for _, bin := range []string{nviPath, goviPath} {
		d := shortDir(t)
		f := tmpFileAt(t, d, "buf.txt", "hello\n")
		rc := filepath.Join(d, "made.exrc")
		run(bin, nil, f, ":mkexrc "+rc+"\r")
		if _, err := os.Stat(rc); err == nil {
			t.Logf("[mkexrc      ] %s: wrote %s", names[bin], rc)
		} else {
			t.Logf("[mkexrc      ] %s: did NOT write an .exrc", names[bin])
		}
	}

	// :preserve honoring recdir
	for _, bin := range []string{nviPath, goviPath} {
		d := shortDir(t)
		rec := filepath.Join(d, "rec")
		os.MkdirAll(rec, 0o755)
		f := tmpFileAt(t, d, "buf.txt", "hello\n")
		run(bin, []string{"EXINIT=set recdir=" + rec}, f, "ix\x1b:preserve\r")
		ents, _ := os.ReadDir(rec)
		t.Logf("[preserve    ] %s: %d recovery entries in recdir", names[bin], len(ents))
	}
}

// TestParityTaglength: taglength=N compares only the first N characters of a
// tag lookup.
func TestParityTaglength(t *testing.T) {
	dir := t.TempDir()
	src := tmpFileAt(t, dir, "src.txt", "top line\nverylongname target\nbottom\n")
	tags := tmpFileAt(t, dir, "tags", "verylongname\t"+src+"\t/^verylongname target$/\n")
	div := 0
	if runArgs(t, "taglength", ":set tl=4 makes :tag veryZZZZ find verylongname",
		":set tags="+tags+"\r:set taglength=4\r:tag veryZZZZ\r", 24, 80, src) {
		div++
	}
	t.Logf("parity-taglength: %d/1 diverged", div)
}
