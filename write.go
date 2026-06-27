package goterm

import "github.com/pborman/ansi"

// Write feeds bytes to the terminal, updating the current screen, and satisfies
// io.Writer.  It always consumes all of data and returns len(data), nil.
//
// An escape sequence may be split across Write calls at any point (down to a
// single byte per call): a sequence truncated at the end of data is held and
// prepended to the next Write, so the caller need not align writes to sequence
// boundaries.
//
// Processing is synchronous (see Term): once Write returns, the screen and Bell
// reflect all of data and nothing runs in the background.
//
// Limitation: non-ASCII (UTF-8) text depends on the ansi decoder, which
// currently treats bytes 0x80-0x9f as C1 controls rather than UTF-8
// continuation bytes.
func (t *Term) Write(data []byte) (int, error) {
	in := data
	if len(t.pending) > 0 {
		in = append(t.pending, data...)
		t.pending = nil
	}
	for len(in) > 0 {
		head := in
		var s *ansi.S
		var err error
		in, s, err = ansi.Decode(head)
		if s == nil {
			break
		}
		if incomplete(err) {
			// The whole of head is a sequence truncated at the end of the
			// input; hold it (copied, since head aliases the input) for the
			// next Write.
			t.pending = append([]byte(nil), head...)
			break
		}
		if s.Error == nil {
			t.process(s)
		}
		if len(in) >= len(head) {
			break // safety: Decode made no progress
		}
	}
	return len(data), nil
}

// incomplete reports whether err means the input ended in the middle of an
// escape sequence, so more bytes are needed to finish it.
func incomplete(err error) bool {
	switch err {
	case ansi.LoneEscape, ansi.IncompleteCSI, ansi.NoST:
		return true
	default:
		return false
	}
}

// process applies one decoded item to the current screen: either a run of plain
// text (printable runes interspersed with C0 controls) or a recognized escape
// sequence dispatched through funcMap.
func (t *Term) process(s *ansi.S) {
	scr := t.Current
	if s.Type == "" {
		// A plain-text run; the decoder does not split out C0 controls, so a
		// control such as CR or LF arrives inline and is routed via funcMap.
		for _, r := range string(s.Code) {
			if r < 0x20 || r == 0x7f {
				if fn := funcMap[ansi.Name(string(r))]; fn != nil {
					fn(scr, nil)
				}
				continue
			}
			scr.put(r)
		}
		return
	}
	if fn := funcMap[s.Code]; fn != nil {
		fn(scr, Params(s.Params))
	}
}
