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

## Pending
* Query responses onto Out: DSR (\E[6n -> CPR) and DA (\E[c), using Term.Send.
* Cross-Write partial sequences: buffer the unconsumed tail for the next Write.
* UTF-8: the vendored ansi decoder treats 0x80-0x9f as C1; make it UTF-8 aware
  (put() already takes full runes).
* Wide (double-width) characters: put() currently treats every rune as width 1.

## Deferred / out of scope
* Save/restore cursor (DECSC/DECRC, ESC[s/u): not exported by pborman/ansi;
  needs custom handling.
* Truecolor SGR (38;2;r;g;b / 48;2): Cell stores a palette index only.
* Scrolling regions (DECSTBM): operations currently act on the full screen.
* REP (repeat): depends on the write path.
