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
