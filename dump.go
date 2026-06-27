package goterm

import (
	"strings"
)

// Dump returns the contents of the screen as a slice of UTF8 strings, one
// string per row.  Trailing spaces are removed from each line.  "" represents a
// line with no text.
func (s *Screen) Dump() []string {
	var lines []string
	for _, line := range s.Lines {
		var b strings.Builder
		for _, c := range line {
			if !c.Wide {
				b.WriteString(c.Value)
			}
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	return lines
}
