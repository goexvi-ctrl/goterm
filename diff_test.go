package goterm

import "testing"

func TestDiffScreensIdentical(t *testing.T) {
	a := []string{"foo", "bar", ""}
	if d := DiffScreens(a, a); d != nil {
		t.Errorf("identical screens diff = %v, want nil", d)
	}
}

func TestDiffScreensRows(t *testing.T) {
	a := []string{"hello", "world", "same"}
	b := []string{"hello", "WORLD", "same"}
	d := DiffScreens(a, b)
	if len(d) != 1 {
		t.Fatalf("got %d diffs, want 1: %v", len(d), d)
	}
	if d[0] != (RowDiff{Row: 1, A: "world", B: "WORLD"}) {
		t.Errorf("diff = %+v, want row 1 world/WORLD", d[0])
	}
}

func TestDiffScreensDifferentLengths(t *testing.T) {
	a := []string{"x"}
	b := []string{"x", "extra"}
	d := DiffScreens(a, b)
	if len(d) != 1 || d[0] != (RowDiff{Row: 1, A: "", B: "extra"}) {
		t.Errorf("diff = %v, want row 1 \"\"/extra", d)
	}
}

func TestDiffCells(t *testing.T) {
	a := [][]Cell{{{Value: "x"}, {Value: "y"}}}
	b := [][]Cell{{{Value: "x"}, {Value: "y", Attributes: int(InverseMask)}}}
	d := DiffCells(a, b)
	if len(d) != 1 {
		t.Fatalf("got %d diffs, want 1: %v", len(d), d)
	}
	if d[0].Row != 0 || d[0].Col != 1 {
		t.Errorf("diff at %d,%d, want 0,1", d[0].Row, d[0].Col)
	}
	if DiffCells(a, a) != nil {
		t.Error("identical snapshots should diff to nil")
	}
}

func TestFormatCellDiffs(t *testing.T) {
	d := []CellDiff{{
		Row: 1, Col: 2,
		A: Cell{Value: "T", Foreground: DefaultForeground, Background: DefaultBackground, Attributes: int(InverseMask)},
		B: Cell{Value: "T", Foreground: DefaultForeground, Background: DefaultBackground},
	}}
	want := `cell 1,2: "T"[inverse] | "T"`
	if got := FormatCellDiffs(d); got != want {
		t.Errorf("format = %q, want %q", got, want)
	}
}

func TestSnapshotIsCopy(t *testing.T) {
	tm := New(2, 3)
	tm.Write([]byte("\x1b[7mA")) // inverse 'A' at 0,0
	snap := tm.Snapshot()
	if snap[0][0].Value != "A" || snap[0][0].Attributes&int(InverseMask) == 0 {
		t.Errorf("snapshot cell = %+v, want inverse 'A'", snap[0][0])
	}
	// Mutating the live screen must not change the snapshot.
	tm.Write([]byte("\x1b[H\x1b[mB"))
	if snap[0][0].Value != "A" {
		t.Errorf("snapshot changed after later writes: %+v", snap[0][0])
	}
}

func TestFormatDiffs(t *testing.T) {
	if got := FormatDiffs(nil); got != "" {
		t.Errorf("empty diffs format = %q, want \"\"", got)
	}
	got := FormatDiffs([]RowDiff{{Row: 2, A: "a", B: "b"}})
	want := `row 2: "a" | "b"`
	if got != want {
		t.Errorf("format = %q, want %q", got, want)
	}
}
