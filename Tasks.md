# Tasks

Fleshing out funcMap with the ANSI escape sequences commonly needed by
text editors. Sequence handling only; input-stream parsing and the
printable-character write path are out of scope for now.

Status: NEW | STARTED | CODED | TESTED | DONE

## Foundations
* DONE - Use value-type cells (Line []Cell), add current-rendition pen and
  tab-stop state, blank() helper.

## Sequence groups
* DONE - Cursor movement (pre-existing): BS, CUU, CUD, CUF, CUB, CHA, CNL,
  CPL, CR.
* TESTED - Cursor positioning: CUP, HVP, VPA, HPA.
* TESTED - Cursor relative position: HPR, HPB, VPR, VPB.
* TESTED - Erase: ED, EL, ECH.
* TESTED - Scroll and line feed: NEL, RI, SU, SD.
* TESTED - Insert/delete lines: IL, DL.
* TESTED - Insert/delete characters: ICH, DCH.
* TESTED - Tabs: HT, CHT, CBT, HTS, TBC.
* TESTED - Select Graphic Rendition: SGR (attributes + 8/16/256 colors).
* TESTED - Flag modes (SM/RM): DECTCEM (?25), DECAWM (?7), IRM (4).  Stored
  as flags for the future write/render path.
* TESTED - Alternate screen buffer (SM/RM ?1049, ?47, ?1047): Term wraps a
  primary and alternate Screen with a Current pointer; ?1049 saves/clears on
  entry and restores on exit.
* TESTED - Line feed and bell: LF/IND (down one line, scroll at bottom, column
  unchanged) and BEL (increments Term.Bell; ClearBell resets it).
* TESTED - Term outbound stream: Out, a single buffered chan []byte (the one
  serial return line), carrying query responses and any other return bytes.
  Producers (parse loop, DSR/DA) pending.
* TESTED - Write feed loop: Term.Write([]byte) (io.Writer) decodes the stream,
  prints runes with deferred wrap and InsertMode, routes C0 controls and
  escapes through funcMap.  put()/index() added to Screen.

* TESTED - Device reports onto Out (via Term.Send): DSR 5 (ESC[0n), DSR 6 (CPR
  ESC[row;colR), and primary DA (ESC[?6c, basic VT102 class).

* TESTED - Cross-Write partial sequences: a sequence truncated at the end of a
  Write is held in Term.pending and prepended to the next Write, so the caller
  may split input anywhere (down to one byte per call).  Includes a multi-byte
  UTF-8 rune split across Writes.
* TESTED - UTF-8 and C0 split: Write uses ansi.Decoder{UTF8: true, C0: true}.
  UTF-8 text round-trips (box-drawing, arrows); C0 controls arrive as their own
  items so process() dispatches everything (controls and escapes) via funcMap
  and text runs are pure printable.

* TESTED - Wide and combining characters: put() uses go-runewidth for display
  width (East Asian Ambiguous = 1).  Width-2 glyphs occupy a lead cell plus a
  Wide continuation and wrap whole at the margin; width-0 combining marks append
  to the preceding cell's lead.

## Pending
* Overwriting half of an existing wide glyph leaves an orphaned cell (the lead
  or stray Wide continuation is not cleaned up).

## Deferred / out of scope
* Save/restore cursor (DECSC/DECRC, ESC[s/u): not exported by pborman/ansi;
  needs custom handling.
* Truecolor SGR (38;2;r;g;b / 48;2): Cell stores a palette index only.
* Scrolling regions (DECSTBM): operations currently act on the full screen.
* REP (repeat): depends on the write path.

## PTY / application launching
* TESTED - Term.Start(name, args...) runs an application on a creack/pty PTY
  sized to the screen; an output pump feeds the app's bytes through Write and a
  forwarder drains Out (responses + Send keystrokes) to the app's input.  Write
  now takes a mutex; Dump/WaitFor are the locked screen accessors.  Smoke test:
  /bin/cat echoes typed input onto the screen.
* TESTED - Start advertises TERM=ansi (this emulator implements the ansi
  terminfo) so applications emit sequences we support.
* TESTED - nvi end to end: launch nvi, insert two lines (i...Esc, o...Esc), and
  confirm the screen renders "Hello, world" / "second line" over tildes.  This
  is the project goal: observe what an editor draws and assert on it.
