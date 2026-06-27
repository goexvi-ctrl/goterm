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

### M (middle of screen) -- rounds the opposite way on an even line count  [B]
- Setup: file of N lines on a 12-row terminal (11 text rows), send `M`.
- Matches when an odd number of lines is displayed, and when the screen is full.
- Diverges when an EVEN number of lines is displayed: with 6 lines (rows 0-5)
  nvi lands on row 3 (L4), govi on row 2 (L3).
- Rule: nvi's middle line is `(top+bottom+1)/2` (rounds down toward the bottom);
  govi uses `(top+bottom)/2` (rounds up toward the top).
- Evidence (probe across N lines on 12x40):
  - N=5  -> both row 2
  - N=6  -> nvi row 3, govi row 2   (diverge)
  - N=7  -> both row 3
  - N=11 -> both row 5
  - N=20 -> both row 5

### :Nd (ex line delete) -- cursor column after delete  [B]
- Setup: sampleLines, send `:2d<CR>` (delete line 2; line 3 "  indented..."
  shifts up to become line 2, which has two leading spaces).
- Body matches (correct line deleted).
- Cursor: nvi lands at column 0; govi lands at column 2 (the first non-blank).
- So after an ex delete govi moves to the first non-blank of the new current
  line, while nvi stays in column 0.  (Note: interactive `dd` matches -- both
  land at column 0 there; the divergence is specific to the ex `:d` path.)

### :set number -- narrower line-number gutter  [C]
- Setup: sampleLines, send `:set number<CR>`.
- nvi renders an 8-column gutter: `"      1 alpha beta gamma"` (text begins at
  column 8).
- govi renders a 6-column gutter: `"    1 alpha beta gamma"` (text begins at
  column 6).
- Cursor reflects it: nvi 0,8 vs govi 0,6.  Cosmetic, but it shifts every body
  column, so any number-mode screen comparison will differ wholesale.

## Matched (verified identical)

These were checked and render/position identically; listed so the list of what
has been covered is explicit.

- Editing: `x`, `3x`, `X`, `dd`, `2dd`, `D`, `dw`, `d$`, `dG`, `J`, `r`, `~`,
  `cw`, `yy`+`p`, `dd`+`p`.
- Motion: `w`, `3w`, `b`, `e`, `0`, `$`, `^`, `f`, `t`, `G`, `gg`, `H`, `L`,
  `50%`.  (`M` diverges -- see above.)
- Search: `/pat`, `/pat`+`n`, `/pat`+`N`, `?pat`, `*`, no-match search.
- Ex: `:Nd` body (cursor differs -- above), `:N,Md`, `:s`, `:%s//g`, `:Nm`,
  `:Nt`, `:N` (goto).

## Known / accepted differences (not bugs)

- `^X` hex input: govi accepts 2, 4, or 6 hex digits; nvi accepts only 2.
  (Stated by the author as an intentional govi extension.)

## Notes on method

- Cursor-only divergences are re-run with `-count=2` to rule out DSR-response
  timing noise before being recorded; all entries above reproduced 2/2.
- Status-line text (e.g. differing error or info messages) is treated as
  cosmetic and is generally excluded from the body comparison by dropping the
  last screen row; only `:set number` above touches layout.
