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

## Deferred / out of scope
* Save/restore cursor (DECSC/DECRC, ESC[s/u): not exported by pborman/ansi;
  needs custom handling.
* Mode setting (SM/RM), including alternate screen buffer (?1049), cursor
  visibility (?25), autowrap (?7): needs DEC private-mode parameter parsing
  and, for the alt screen, a second buffer.
* Truecolor SGR (38;2;r;g;b / 48;2): Cell stores a palette index only.
* Scrolling regions (DECSTBM): operations currently act on the full screen.
* REP (repeat): depends on the write path.
