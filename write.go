package goterm

import "github.com/pborman/ansi"

// Write feeds bytes to the terminal, updating the current screen, and satisfies
// io.Writer.  It always consumes all of data and returns len(data), nil.
//
// Processing is synchronous (see Term): once Write returns, the screen and Bell
// reflect all of data and nothing runs in the background.
//
// Limitations: an escape sequence split across two Write calls is dropped (the
// trailing partial is not buffered for the next call), and non-ASCII (UTF-8)
// text depends on the ansi decoder, which currently treats bytes 0x80-0x9f as
// C1 controls rather than UTF-8 continuation bytes.
func (t *Term) Write(data []byte) (int, error) {
	in := data
	for {
		var s *ansi.S
		in, s, _ = ansi.Decode(in)
		if s == nil {
			break
		}
		if s.Error == nil {
			t.process(s)
		}
	}
	return len(data), nil
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
