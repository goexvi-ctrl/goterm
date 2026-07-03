package goterm

import (
	"io"
	"testing"

	"github.com/pborman/ansi"
)

// Term must satisfy io.Writer.
var _ io.Writer = (*Term)(nil)

func TestWriteText(t *testing.T) {
	tm := New(3, 10)
	n, err := tm.Write([]byte("hello"))
	if n != 5 || err != nil {
		t.Fatalf("Write returned (%d, %v), want (5, nil)", n, err)
	}
	if got := tm.Current.Dump()[0]; got != "hello" {
		t.Errorf("row 0 = %q, want %q", got, "hello")
	}
	if tm.Current.Row != 0 || tm.Current.Col != 5 {
		t.Errorf("cursor = %d,%d, want 0,5", tm.Current.Row, tm.Current.Col)
	}
}

func TestWriteEmpty(t *testing.T) {
	tm := New(3, 10)
	if n, err := tm.Write(nil); n != 0 || err != nil {
		t.Errorf("Write(nil) = (%d, %v), want (0, nil)", n, err)
	}
}

func TestWriteLineFeedKeepsColumn(t *testing.T) {
	// A bare LF moves down but does not return to column 0.
	tm := New(3, 10)
	tm.Write([]byte("ab\ncd"))
	d := tm.Current.Dump()
	if d[0] != "ab" {
		t.Errorf("row 0 = %q, want %q", d[0], "ab")
	}
	if d[1] != "  cd" {
		t.Errorf("row 1 = %q, want %q", d[1], "  cd")
	}
	if tm.Current.Row != 1 || tm.Current.Col != 4 {
		t.Errorf("cursor = %d,%d, want 1,4", tm.Current.Row, tm.Current.Col)
	}
}

func TestWriteCRLF(t *testing.T) {
	tm := New(3, 10)
	tm.Write([]byte("ab\r\ncd"))
	if got := tm.Current.Dump()[1]; got != "cd" {
		t.Errorf("row 1 = %q, want %q", got, "cd")
	}
	if tm.Current.Row != 1 || tm.Current.Col != 2 {
		t.Errorf("cursor = %d,%d, want 1,2", tm.Current.Row, tm.Current.Col)
	}
}

func TestWriteBackspace(t *testing.T) {
	tm := New(3, 10)
	tm.Write([]byte("abc\b"))
	if tm.Current.Col != 2 {
		t.Errorf("Col after backspace = %d, want 2", tm.Current.Col)
	}
}

func TestWriteTab(t *testing.T) {
	tm := New(3, 20)
	tm.Write([]byte("a\tb"))
	// 'a' at col 0, tab advances to col 8, 'b' at col 8.
	if got := tm.Current.Lines[0][8].Value; got != "b" {
		t.Errorf("cell 0,8 = %q, want 'b'", got)
	}
}

func TestWriteAutoWrap(t *testing.T) {
	tm := New(2, 3)
	tm.Write([]byte("abcd"))
	d := tm.Current.Dump()
	if d[0] != "abc" {
		t.Errorf("row 0 = %q, want %q", d[0], "abc")
	}
	if d[1] != "d" {
		t.Errorf("row 1 = %q, want %q", d[1], "d")
	}
	if tm.Current.Row != 1 || tm.Current.Col != 1 {
		t.Errorf("cursor = %d,%d, want 1,1", tm.Current.Row, tm.Current.Col)
	}
}

func TestWriteDeferredWrap(t *testing.T) {
	// Filling exactly the last column must NOT scroll; the wrap is deferred to
	// the next printable rune (split across two Writes here).
	tm := New(2, 3)
	tm.Write([]byte("abc"))
	if tm.Current.Row != 0 {
		t.Errorf("after filling the line, Row = %d, want 0 (no premature wrap)", tm.Current.Row)
	}
	tm.Write([]byte("d"))
	if tm.Current.Row != 1 || tm.Current.Col != 1 {
		t.Errorf("cursor = %d,%d, want 1,1", tm.Current.Row, tm.Current.Col)
	}
	if got := tm.Current.Dump()[1]; got != "d" {
		t.Errorf("row 1 = %q, want %q", got, "d")
	}
}

func TestWriteNoAutoWrap(t *testing.T) {
	tm := New(2, 3)
	tm.Current.AutoWrap = false
	tm.Write([]byte("abcd"))
	// 'd' overwrites 'c' at the last column; nothing wraps.
	d := tm.Current.Dump()
	if d[0] != "abd" {
		t.Errorf("row 0 = %q, want %q", d[0], "abd")
	}
	if d[1] != "" {
		t.Errorf("row 1 = %q, want blank", d[1])
	}
	if tm.Current.Row != 0 || tm.Current.Col != 2 {
		t.Errorf("cursor = %d,%d, want 0,2", tm.Current.Row, tm.Current.Col)
	}
}

func TestWriteInsertMode(t *testing.T) {
	// End-to-end: SM 4 enables insert mode, then a printed char shifts the rest
	// of the line right.
	tm := New(1, 5)
	tm.Write([]byte("abc\r\x1b[4hX"))
	if got := tm.Current.Dump()[0]; got != "Xabc" {
		t.Errorf("row 0 = %q, want %q", got, "Xabc")
	}
	if tm.Current.Col != 1 {
		t.Errorf("Col = %d, want 1", tm.Current.Col)
	}
}

func TestWriteSGRPen(t *testing.T) {
	// An escape parsed mid-stream updates the pen, and the next glyph adopts it.
	tm := New(3, 10)
	tm.Write([]byte("\x1b[31mA"))
	if tm.Current.Cur.Foreground != Red {
		t.Errorf("pen fg = %d, want Red", tm.Current.Cur.Foreground)
	}
	cell := tm.Current.Lines[0][0]
	if cell.Value != "A" || cell.Foreground != Red {
		t.Errorf("cell = %+v, want 'A' in Red", cell)
	}
}

func TestWriteCursorPosition(t *testing.T) {
	tm := New(5, 10)
	tm.Write([]byte("\x1b[3;4H"))
	if tm.Current.Row != 2 || tm.Current.Col != 3 {
		t.Errorf("cursor = %d,%d, want 2,3", tm.Current.Row, tm.Current.Col)
	}
}

func TestWriteAltScreenSwitch(t *testing.T) {
	tm := New(3, 10)
	tm.Write([]byte("\x1b[?1049hx"))
	if tm.Current != tm.Alternate {
		t.Fatal("?1049h should switch to the alternate screen")
	}
	if tm.Alternate.Lines[0][0].Value != "x" {
		t.Error("text after the switch should land on the alternate screen")
	}
}

func TestWriteBell(t *testing.T) {
	tm := New(3, 10)
	tm.Write([]byte("a\ab"))
	if tm.Bell != 1 {
		t.Errorf("Bell = %d, want 1", tm.Bell)
	}
}

func TestWriteDSRCursorPosition(t *testing.T) {
	tm := New(24, 80)
	// Move the cursor, then query its position; the reply is 1-based.
	tm.Write([]byte("\x1b[3;4H\x1b[6n"))
	if got := string(<-tm.Out); got != "\x1b[3;4R" {
		t.Errorf("CPR = %q, want %q", got, "\x1b[3;4R")
	}
	// The query itself must not print to the screen.
	if line := tm.Current.Dump()[2]; line != "" {
		t.Errorf("query printed to screen: row 2 = %q", line)
	}
}

func TestWriteDSRStatus(t *testing.T) {
	tm := New(5, 10)
	tm.Write([]byte("\x1b[5n"))
	if got := string(<-tm.Out); got != "\x1b[0n" {
		t.Errorf("DSR 5 reply = %q, want %q", got, "\x1b[0n")
	}
}

func TestWriteDA(t *testing.T) {
	tm := New(5, 10)
	tm.Write([]byte("\x1b[c"))
	if got := string(<-tm.Out); got != "\x1b[?6c" {
		t.Errorf("DA reply = %q, want %q", got, "\x1b[?6c")
	}
}

func TestDeviceReportStandaloneNoop(t *testing.T) {
	// A standalone screen has no return stream; the reports must not panic.
	s := NewScreen(5, 10)
	funcMap[ansi.DSR](s, Params{"6"})
	funcMap[ansi.DA](s, Params{"0"})
}

func TestWriteDump(t *testing.T) {
	tm := New(3, 10)
	tm.Write([]byte("hi\r\nthere"))
	got := tm.Current.Dump()
	want := []string{"hi", "there", ""}
	if len(got) != len(want) {
		t.Fatalf("Dump returned %d rows, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Dump row %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestWriteByteAtATime(t *testing.T) {
	tm := New(5, 10)
	for _, b := range []byte("\x1b[2;3Hhi") {
		tm.Write([]byte{b})
	}
	if got := tm.Current.Dump()[1]; got != "  hi" {
		t.Errorf("row 1 = %q, want %q", got, "  hi")
	}
	if tm.Current.Row != 1 || tm.Current.Col != 4 {
		t.Errorf("cursor = %d,%d, want 1,4", tm.Current.Row, tm.Current.Col)
	}
	if len(tm.pending) != 0 {
		t.Errorf("pending not drained: %q", tm.pending)
	}
}

func TestWriteSplitCSI(t *testing.T) {
	tm := New(5, 10)
	tm.Write([]byte("\x1b[3")) // truncated mid-sequence
	if string(tm.pending) != "\x1b[3" {
		t.Errorf("pending = %q, want %q", tm.pending, "\x1b[3")
	}
	tm.Write([]byte(";4H"))
	if tm.Current.Row != 2 || tm.Current.Col != 3 {
		t.Errorf("cursor = %d,%d, want 2,3", tm.Current.Row, tm.Current.Col)
	}
	if len(tm.pending) != 0 {
		t.Errorf("pending not drained: %q", tm.pending)
	}
}

func TestWriteSplitAfterText(t *testing.T) {
	tm := New(3, 10)
	tm.Write([]byte("ab\x1b[")) // text, then a truncated escape
	if string(tm.pending) != "\x1b[" {
		t.Errorf("pending = %q, want %q", tm.pending, "\x1b[")
	}
	if got := tm.Current.Dump()[0]; got != "ab" {
		t.Errorf("row 0 = %q, want %q (text before the split must print)", got, "ab")
	}
	tm.Write([]byte("31mC"))
	if got := tm.Current.Dump()[0]; got != "abC" {
		t.Errorf("row 0 = %q, want %q", got, "abC")
	}
	if cell := tm.Current.Lines[0][2]; cell.Foreground != Red {
		t.Errorf("'C' foreground = %d, want Red (pen from split SGR)", cell.Foreground)
	}
}

func TestWriteSplitLoneEscape(t *testing.T) {
	tm := New(3, 10)
	tm.Write([]byte("x\x1b")) // a bare trailing ESC
	if string(tm.pending) != "\x1b" {
		t.Errorf("pending = %q, want ESC", tm.pending)
	}
	tm.Write([]byte("[1m"))
	if tm.Current.Cur.Attributes&int(BoldMask) == 0 {
		t.Error("bold not set after completing the split SGR")
	}
}

func TestWriteUTF8(t *testing.T) {
	tm := New(3, 10)
	tm.Write([]byte("┌─┐"))
	if got := tm.Current.Dump()[0]; got != "┌─┐" {
		t.Errorf("row 0 = %q, want %q", got, "┌─┐")
	}
	// Each box-drawing glyph is single-width, so the cursor advanced by three.
	if tm.Current.Col != 3 {
		t.Errorf("Col = %d, want 3", tm.Current.Col)
	}
	if tm.Current.Lines[0][1].Value != "─" {
		t.Errorf("cell 0,1 = %q, want '─'", tm.Current.Lines[0][1].Value)
	}
}

func TestWriteUTF8SplitRune(t *testing.T) {
	// "─" is the three bytes e2 94 80; split it across two Writes.
	tm := New(3, 10)
	r := []byte("─")
	tm.Write(r[:2])
	if string(tm.pending) != string(r[:2]) {
		t.Errorf("pending = %x, want %x (partial rune held)", tm.pending, r[:2])
	}
	if tm.Current.Dump()[0] != "" {
		t.Error("nothing should print until the rune is complete")
	}
	tm.Write(r[2:])
	if got := tm.Current.Dump()[0]; got != "─" {
		t.Errorf("row 0 = %q, want %q", got, "─")
	}
	if len(tm.pending) != 0 {
		t.Errorf("pending not drained: %x", tm.pending)
	}
}

func TestWriteUTF8ByteAtATime(t *testing.T) {
	tm := New(3, 10)
	for _, b := range []byte("→") { // e2 86 92
		tm.Write([]byte{b})
	}
	if got := tm.Current.Dump()[0]; got != "→" {
		t.Errorf("row 0 = %q, want %q", got, "→")
	}
}

func TestWriteWideChar(t *testing.T) {
	tm := New(2, 5)
	tm.Write([]byte("世")) // CJK ideograph U+4E16, double-width
	if got := tm.Current.Dump()[0]; got != "世" {
		t.Errorf("row 0 = %q, want %q", got, "世")
	}
	if tm.Current.Col != 2 {
		t.Errorf("Col = %d, want 2 (double width)", tm.Current.Col)
	}
	if lead := tm.Current.Lines[0][0]; lead.Value != "世" || lead.Wide {
		t.Errorf("lead = %+v, want the glyph and not Wide", lead)
	}
	if !tm.Current.Lines[0][1].Wide {
		t.Error("continuation cell should have Wide set")
	}
}

func TestWriteWideWrap(t *testing.T) {
	// A double-width glyph that would straddle the right margin wraps whole,
	// leaving the last column blank.
	tm := New(2, 3)
	tm.Write([]byte("ab世"))
	d := tm.Current.Dump()
	if d[0] != "ab" {
		t.Errorf("row 0 = %q, want %q", d[0], "ab")
	}
	if d[1] != "世" {
		t.Errorf("row 1 = %q, want %q", d[1], "世")
	}
	if tm.Current.Row != 1 || tm.Current.Col != 2 {
		t.Errorf("cursor = %d,%d, want 1,2", tm.Current.Row, tm.Current.Col)
	}
}

func TestWriteCombining(t *testing.T) {
	tm := New(1, 5)
	tm.Write([]byte("é")) // 'e' + COMBINING ACUTE ACCENT
	if got := tm.Current.Dump()[0]; got != "é" {
		t.Errorf("row 0 = %q, want %q", got, "é")
	}
	if got := tm.Current.Lines[0][0].Value; got != "é" {
		t.Errorf("cell value = %q, want %q", got, "é")
	}
	if tm.Current.Col != 1 {
		t.Errorf("Col = %d, want 1 (a combining mark does not advance)", tm.Current.Col)
	}
}

func TestWriteCombiningOnWide(t *testing.T) {
	// A combining mark after a wide glyph attaches to the glyph's lead cell,
	// stepping back over the Wide continuation.
	tm := New(1, 5)
	tm.Write([]byte("世́"))
	if got := tm.Current.Lines[0][0].Value; got != "世́" {
		t.Errorf("lead value = %q, want %q", got, "世́")
	}
	if tm.Current.Col != 2 {
		t.Errorf("Col = %d, want 2", tm.Current.Col)
	}
}

func TestWriteScrollsAtBottom(t *testing.T) {
	tm := New(2, 3)
	// Three CRLF-separated lines on a 2-row screen scroll the first one off.
	tm.Write([]byte("a\r\nb\r\nc"))
	d := tm.Current.Dump()
	if d[0] != "b" {
		t.Errorf("row 0 = %q, want %q", d[0], "b")
	}
	if d[1] != "c" {
		t.Errorf("row 1 = %q, want %q", d[1], "c")
	}
}

func TestWriteRepeat(t *testing.T) {
	// REP (CSI Ps b) repeats the preceding graphic character; the ansi/goterm
	// terminfo advertises rep, so ncurses compresses runs this way.
	tm := New(2, 10)
	tm.Write([]byte("z\x1b[4b"))
	if d := tm.Current.Dump(); d[0] != "zzzzz" {
		t.Errorf("row 0 = %q, want %q", d[0], "zzzzz")
	}
	// REP with nothing preceding is a no-op.
	tm2 := New(2, 10)
	tm2.Write([]byte("\x1b[5b"))
	if d := tm2.Current.Dump(); d[0] != "" {
		t.Errorf("REP with no prior char: row 0 = %q, want empty", d[0])
	}
	// REP participates in wrapping like typed text.
	tm3 := New(2, 4)
	tm3.Write([]byte("a\x1b[5b"))
	d := tm3.Current.Dump()
	if d[0] != "aaaa" || d[1] != "aa" {
		t.Errorf("wrapped REP = %q,%q, want aaaa,aa", d[0], d[1])
	}
}
