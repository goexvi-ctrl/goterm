package goterm

import "github.com/pborman/ansi"

// streamDecoder is configured for a modern terminal: UTF-8 text passes through
// intact, and C0 controls are split out as their own items (Type "C0") so they
// dispatch through funcMap like any other sequence.
var streamDecoder = ansi.Decoder{UTF8: true, C0: true}

// Write feeds bytes to the terminal, updating the current screen, and satisfies
// io.Writer.  It always consumes all of data and returns len(data), nil.
//
// Input may be split across Write calls at any point, down to a single byte: an
// escape sequence or a multi-byte UTF-8 rune truncated at the end of data is
// held and prepended to the next Write, so the caller need not align writes to
// any boundary.
//
// Processing is synchronous (see Term): once Write returns, the screen and Bell
// reflect all of data and nothing runs in the background.
func (t *Term) Write(data []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	in := data
	if len(t.pending) > 0 {
		in = append(t.pending, data...)
		t.pending = nil
	}
	for len(in) > 0 {
		head := in
		var s *ansi.S
		var err error
		in, s, err = streamDecoder.Decode(head)
		if s == nil {
			// The input ends with an incomplete UTF-8 rune; the decoder left it
			// unconsumed (in == head).  Hold it for the next Write.
			t.pending = append([]byte(nil), in...)
			break
		}
		if incomplete(err) {
			// The whole of head is a sequence truncated at the end of the
			// input; hold it (copied, since head aliases the input).
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

// process applies one decoded item to the current screen: a run of printable
// text, or a sequence (a C0 control or an escape) dispatched through funcMap.
func (t *Term) process(s *ansi.S) {
	scr := t.Current
	if s.Type == "" {
		// A pure printable run; the decoder splits C0 controls out separately.
		for _, r := range string(s.Code) {
			scr.put(r)
		}
		return
	}
	if fn := funcMap[s.Code]; fn != nil {
		fn(scr, Params(s.Params))
	}
}
