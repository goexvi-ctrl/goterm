# govi vs nvi divergences

Behavioral differences found by driving govi (`/Users/claude/bin/govi`) and nvi
(`/opt/homebrew/bin/nvi`) through identical input on identical terminals and
comparing the rendered screen and cursor.  Mined by the `TestDiverge*` tests in
`divergence_test.go` (fresh editors per scenario, so each divergence is
attributable to the single sequence under test).

Status: B = behavioral (a command does something different), C = cosmetic
(layout/status text only), K = known/accepted (intentional govi difference).

Each entry: what was sent, what nvi does, what govi does, and the precise rule
where known.

## Confirmed

### <Esc> followed by more bytes in one write does not exit insert mode  [B] -- highest priority
- Setup: enter insert (`i`/`a`/`A`/`o`/`cc`/`cw`), type text, then `<Esc>`, then
  MORE bytes, all delivered in a single write (e.g. `cwX<Esc>w.`).
- nvi exits insert on the `<Esc>` and runs the following bytes as commands.
- govi does NOT exit; it inserts the bytes after `<Esc>` as literal text.
- Affects every insert entry, not just `cw`:
  - govi `iZ<Esc>x`   -> `"Zxalpha beta gamma"`   (nvi -> `"alpha beta gamma"`)
  - govi `aZ<Esc>x`   -> `"aZxlpha beta gamma"`   (nvi correct)
  - govi `cwX<Esc>w.` -> `"Xw. beta gamma"`       (nvi -> `"X X gamma"`)
  - govi `cwX<Esc>j0cwY<Esc>` -> `"Xj0cwY beta gamma"` (everything up to the
    next Esc is swallowed as text)
- Delivering the `<Esc>` in its own write (a pause after it) works correctly:
  govi `cwZ<Esc>` then `x` -> `" beta gamma"`.  So the bug is about `<Esc>`
  immediately followed by more input, not about `<Esc>` itself.
- Likely cause: govi treats `<Esc>` + following bytes as a single key/escape
  sequence (Meta/arrow disambiguation) without the timeout nvi uses, so a
  scripted/pasted burst that contains a mid-stream `<Esc>` is misread.  nvi
  disambiguates correctly even when the bytes arrive together.
- METHOD IMPLICATION: any comparison that sends a mid-stream `<Esc>` will report
  a spurious divergence rooted here.  Until govi is fixed, send `<Esc>` (and the
  following keystroke) as a separate write when testing other commands.

### Paging commands do not move the viewport (^F ^B ^D ^U ^E ^Y)  [B]
- Setup: 60-line numbered file ("001".."060") on a 12-row terminal (11 text
  rows).
- nvi pages the viewport (and counts by window-2 = 9 for a full page, window/2
  for a half page) and puts the cursor at the top of the new page; govi leaves
  the viewport fixed and only moves the cursor within the current page.
- Per command (top line shown, cursor row):

  | keys | nvi top / cursor | govi top / cursor |
  |------|------------------|-------------------|
  | `^F` (from line 1)       | top 010, cur row 0  | top 001, cur row 9  |
  | `^F^F`                   | top 019, cur row 0  | top 001, cur row 10 |
  | `^B` (from end)          | top 041, cur row 10 | top 050, cur row 1  |
  | `^D` (from line 1)       | top 007, cur row 0  | top 001, cur row 5  |
  | `^U` (from end)          | top 044, cur row 10 | top 050, cur row 5  |
  | `^E` (scroll down 1)     | top 002             | top 001 (no scroll) |
  | `^Y` (scroll up 1)       | top moves up 1      | no scroll           |

- Note the half-page count also differs: nvi `^D` advances the cursor line by 6,
  govi by 5 (window/2 rounding), on top of the no-scroll difference.
- This is the original `^F` finding, now generalized across the whole family.

### goto off-screen line: viewport placement differs  [B]
- Setup: 60-line file, `20G` (jump forward to a line below the current page).
- Both put the cursor on line 020, but nvi places it mid-screen (top 015, cursor
  row 5) while govi places it on the bottom row (top 010, cursor row 10).
- So a forward jump to an off-screen line scrolls the target to the middle in
  nvi and to the bottom in govi.

### M (middle of screen): rounds the opposite way on an even line count  [B]
- Setup: file of N lines on a 12-row terminal, send `M`.
- Matches on an odd displayed-line count and on a full screen; diverges on an
  EVEN count: with 6 lines nvi lands on row 3 (L4), govi on row 2 (L3).
- Rule: nvi's middle line is `(top+bottom+1)/2` (rounds toward the bottom); govi
  uses `(top+bottom)/2` (rounds toward the top).
- Evidence (12x40): N=5 both row 2; N=6 nvi 3 / govi 2; N=7 both row 3;
  N=11 both row 5; N=20 both row 5.

### :Nd (ex line delete): cursor column after delete  [B]
- Setup: sampleLines, `:2d<CR>` (line 3 "  indented..." shifts up to line 2).
- Body matches; cursor differs: nvi lands at column 0, govi at column 2 (the
  first non-blank).  Interactive `dd` matches (both column 0), so the divergence
  is specific to the ex `:d` path.

### :set number: narrower line-number gutter  [C]
- nvi renders an 8-column gutter (`"      1 alpha..."`, text at column 8); govi a
  6-column gutter (`"    1 alpha..."`, text at column 6).  Shifts every body
  column, so number-mode comparisons differ wholesale.

## Matched (verified identical)

- Editing: `x`, `3x`, `X`, `dd`, `2dd`, `D`, `dw`, `d$`, `dG`, `J`, `r`, `~`,
  `cw` (Esc at end), `yy`+`p`, `dd`+`p`.
- Motion: `w`, `3w`, `b`, `e`, `0`, `$`, `^`, `f`, `t`, `G`, `gg`, `H`, `L`,
  `50%`.  (`M` diverges -- above.)
- Search: `/pat`, `/pat`+`n`, `/pat`+`N`, `?pat`, `*`, no-match search.
- Ex: `:Nd` body (cursor differs), `:N,Md`, `:s`, `:%s//g`, `:Nm`, `:Nt`, `:N`.
- Registers/repeat: `m`+`` ` ``, `m`+`'`, `"ayy`+`"ap`, `.` after `x`, `.` after
  `dd`, `qa..q`+`@a` macro, `"A` append-register.  (`.` after `cw` diverges --
  that is the <Esc> bug above, not a `.` bug.)
- Structure: `>>`, `2>>`, `<<`, `>G`, `%` (paren and brace match), `==`.

## Known / accepted differences (not bugs)

- `^X` hex input: govi accepts 2, 4, or 6 hex digits; nvi accepts only 2.
  (Intentional govi extension, per the author.)

## Notes on method

- Cursor-only divergences are re-run with `-count=2` to rule out DSR-response
  timing noise before being recorded; all entries above reproduced.
- Status-line text (differing error/info messages) is treated as cosmetic and
  excluded by dropping the last screen row from the body comparison.
- The `<Esc>` bug above means a single root cause can masquerade as many
  command-level divergences; mid-stream `<Esc>` should be split out when probing
  unrelated commands.
