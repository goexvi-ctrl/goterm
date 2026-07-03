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
* TESTED - Start now advertises the emulator's own terminal type "goterm"
  (ansi plus smcup/rmcup), compiled with tic at first use into a TERMINFO dir
  (terminfo.go; falls back to TERM=ansi without tic).  Plain ansi hid every
  alternate-screen bug: editors never emitted ?1049.  AltScreenActive() exposes
  which buffer is live; TestParityTermState pins vi-on-alternate-screen /
  ex-off-it and restore-on-quit for both editors.
* TESTED - nvi end to end: launch nvi, insert two lines (i...Esc, o...Esc), and
  confirm the screen renders "Hello, world" / "second line" over tildes.  This
  is the project goal: observe what an editor draws and assert on it.

## Comparison harness helpers
* TESTED - Term.WaitQuiet(idle, timeout): settle until no Write for idle (output
  has stopped), for driving an app when the exact target screen is unknown.
* TESTED - DiffScreens(a, b) []RowDiff and FormatDiffs: compare two Dump outputs
  row by row for diffing one editor's screen against another.

## Editor comparison
* TESTED - Snapshot returns a race-safe copy of the screen cells (colors and
  attributes, which Dump omits).  DiffCells / FormatCellDiffs compare two
  snapshots cell by cell.
* TESTED - editorPair drives two editors in lockstep (same size, same input);
  use WaitFor for readiness (an editor can go quiet mid-startup), WaitQuiet to
  let a redraw finish.
* TESTED - govi vs nvi comparison report.  Finding so far: on an invalid ex
  command, nvi shows the error in reverse video with a trailing period; govi
  shows it plain with no period.  The attribute diff is needed to catch the
  reverse video (text diff alone misses it).

## govi vs nvi findings
* Term.Cursor() exposes the cursor position (locked) for behavioral comparison.
* WaitQuiet fix: Send now also resets the idle clock (lastActivity is atomic,
  updated by Write and Send), so WaitQuiet after a command waits for that
  command's response instead of returning immediately on an already-quiet
  screen.  This also fixed apparent input loss at small screen sizes (it was the
  same lag, not a size bug -- 12x40 works).
* FINDING - control-key paging (^F/^B/^D/^U): nvi repaginates the viewport
  (e.g. ^F advances the top by window-2 lines, cursor to the top of the new
  page); govi moves the cursor but leaves the viewport unchanged.  Basic motions
  (j, G, 1G) match.  See TestCompareScroll.
* NOTE - status messages differ cosmetically (nvi "new file: line 1" /
  "unmodified: line 1"; govi "N lines, M characters"); error messages: nvi adds
  a trailing period and uses reverse video, govi neither.  Lower priority.
* TESTED - 2026-07-03 QA review battery (qa_test.go): TestQACompound/Wrap/
  Edge/ExEdge/Disk, 117 probes over compound operations, wrapped-line
  behavior, boundary conditions, ex corners, and on-disk write-back.
  Adjudicated findings live in the govi tree: govi/qa/REPORT.md (untracked
  by design).  Three harness bugs found and fixed along the way: the goterm
  terminfo entry now declares xenl (the emulator does deferred wrap; without
  the flag ncurses desynced on exactly-full-width lines), the emulator now
  implements REP / CSI Ps b (ncurses compresses repeated-character runs via
  the ansi rep capability; TestWriteRepeat), and TestCompareScroll now sets
  EXINIT="set nolock" plus a settle before its first send (shared-fixture
  lock race ate the first keystroke about half the time once the terminfo
  change shifted startup timing).
