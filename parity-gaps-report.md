# parity.md gap report -- the 🟡 Partial and ❌ Not yet entries

Untracked analysis (not part of GOTERM_DIVERGENCES.md or the govi repo). Written
2026-06-28 against `govi/docs/parity.md`; pruned as items were fixed.

For every still-open 🟡 Partial / ❌ Not yet row in parity.md this report gives a
verdict:

- **CAN** -- addressable with bounded effort; the supporting machinery already
  exists in govi. Effort is a rough size (S = a few hours, M = a day-ish, L =
  multi-day / new subsystem).
- **DEFERRED** -- technically addressable but low user value, cosmetic, or a
  deliberate design choice; not worth doing now.
- **CAN'T (here)** -- blocked on a subsystem govi does not have (split screens) or
  on a frontend that structurally cannot host the feature (GUI vs tty), or it is
  an explicit non-objective.

Caveat: verdicts and effort sizes are an assessment from the parity notes, the
divergence catalog, nvi's documented behavior, and the harness findings -- not
from a line-by-line read of govi's source. Treat effort sizes as planning hints.

Resolved since the first draft (removed from the lists below):
- `z^`/`z+` screen types -> FIXED (GOTERM_DIVERGENCES #40).
- `secure` not gating `!` filters -> FIXED (#41).
- `writeany` / overwrite guard -> FIXED (#42; it was a real data-loss bug -- govi
  had no overwrite guard at all).
- `lock` (concurrent-edit protection) -> FIXED (#45).
- input `^U`/line-erase and ex `:k`/`:ma`/`:mark` -> turned out ALREADY DONE; the
  parity.md rows were stale and have been corrected.

That cleared most of the write-path / data-safety cluster, which is now done.

Resolved by the 2026-07-01 parity review (see govi/docs/parity-review.md):
- #4 `<interrupt>` -> DONE earlier (engine/interrupt.go, three-stage ^C).
- #5 `^W` and #12/#22 split screens -> DONE (implemented 2026-06-30; verified
  7/7 by TestParitySplits -- this report's CAN'T verdicts are obsolete).
- #8 `:tagn`/`:tagp`/`:tagt` -> DONE (verified by TestParityExCmds).
- #10 `:di[splay]` -> DONE except output format for buffers/tags
  (GOTERM_DIVERGENCES #54).
- #14 `autowrite` -> DONE (nvi file_m1 semantics; catalog #53).
- Not on this list but found and fixed in the same review: insert `^T`
  at-cursor indent (#49), `:visual file` (#50), the `window` option (#51),
  `:preserve` surviving exit (#52).

---

## Summary (22 open entries, in parity.md section order)

| # | Entry | Section | govi | Verdict |
|---|---|---|---|---|
| 1 | `^E` `^Y` over wrapped lines | vi cmd | 🟡 | DEFERRED (architectural -- soft-map) |
| 2 | `^L` `^R` repaint | vi cmd | 🟡 | DEFERRED (no-op by design) |
| 3 | `Q` ex mode display | vi cmd | 🟡 | DEFERRED (design choice) |
| 4 | `<interrupt>` cancellation | vi cmd | 🟡 | CAN (M) |
| 5 | `^W` switch screens | vi cmd | ❌ | CAN'T (split screens) |
| 6 | input `^D`/`^T` autoindent | vi input | 🟡 | CAN (M) |
| 7 | input `0^D` `^^D` | vi input | ❌ | CAN (M, with #6) |
| 8 | `:tagn`/`:tagp`/`:tagt` | ex | ❌ | CAN (M) |
| 9 | `:mk[exrc]` | ex | ❌ | CAN (M) |
| 10 | `:di[splay]` | ex | ❌ | CAN (M, gated on msg-pagination) |
| 11 | `:[line]z` ex screenful | ex | ❌ | CAN (M, gated on msg-pagination) |
| 12 | `:bg`/`:fg`/`:res`/`:sc`/`:vs` | ex | ❌ | CAN'T (split screens) |
| 13 | `:sh[ell]` in GoVi.app | ex | ❌ (GUI) | CAN'T-ish (GUI design) |
| 14 | `autowrite` (aw) | option | ❌ | CAN (M) |
| 15 | `backup` | option | ❌ | CAN (M) |
| 16 | Marks intra-line column fixup | subsystem | 🟡 | CAN (M) |
| 17 | Maps / abbreviations (recursive) | subsystem | 🟡 | CAN (M) |
| 18 | Line wrapping `@`-fill | subsystem | 🟡 | DEFERRED (architectural -- soft-map) |
| 19 | File encodings (iconv) | subsystem | 🟡 | DEFERRED (large, modern-irrelevant) |
| 20 | Left-right scrolling | subsystem | ❌ | DEFERRED (rendering rework, niche) |
| 21 | Command-line editing (`cedit`) | subsystem | ❌ | DEFERRED (niche) |
| 22 | Split screens / windows | subsystem | ❌ | CAN'T (split screens) |

Headline: of the 22 still-open gaps, 11 are genuinely addressable (CAN), 7 are
deferred (design/cosmetic or the long-line soft-map cluster), and 4 are CAN'T-here
-- 3 blocked on the split-screen subsystem (a de-facto non-objective for the tty
frontend, since GoVi.app provides native windows/tabs instead) plus GoVi.app
`:shell` (no controlling terminal in the GUI).

---

## CAN be addressed

### 4. `<interrupt>` -- not all operations cancellable  (🟡, M)
Search and some operations honor interrupt; long-running ones (a big `:g//`,
`:%s//`, a slow `!` filter read) do not all check for it. nvi polls an interrupt
flag in its main loops. **Addressable** by threading a cancellation check
(context/atomic flag) through the global-command loop, the substitute loop, and
the filter read. Medium: it touches several loops and needs care not to leave the
buffer half-modified (nvi stops between lines).

### 6. input `^D` / `^T` autoindent + 7. `0^D` / `^^D`  (🟡 / ❌, M together)
This is the `ctrl-t` coverage finding: in input mode nvi's `^T` inserts whitespace
to round the cursor up to the next `shiftwidth` column **and** records autoindent
state; `^D` backs off one `shiftwidth`; `0^D` erases all autoindent on the line;
`^^D` (caret then `^D`) erases autoindent for the current line only but restores
it next line. govi currently approximates `^D`/`^T` as a whole-line shift
left/right and does not implement the `0^D`/`^^D` variants at all. **Addressable**
together by implementing nvi's autoindent column model (track the autoindent
"target" column per input line). Medium; the two ❌ variants fall out of the same
model.

### 8. `:tagnext` / `:tagprev` / `:tagtop`  (❌, M)
The tag *stack* already exists -- vi `^]` pushes and `^T` pops, and `:tag` works
(all ✅✔). These three ex commands walk the same stack (next/prev match for the
last tag, and discard-all). **Addressable**, medium: they need the multiple-match
list nvi keeps per tag (so `:tagnext` can step through ambiguous matches), which
may not be retained today; if only the single best match is stored, that list has
to be added.

### 9. `:mkexrc`  (❌, M)
Writes current settings to an exrc file. govi already has the full option table
(`:set all` renders it) plus maps and abbreviations. **Addressable**, medium:
iterate the option registry emitting `set` lines for non-default values, then the
active `:map`/`:ab` definitions, to the target file. The only subtlety is matching
nvi's output format and the `OPT_NOSAVE` exclusions.

### 10. `:display b|c|s|t` + 11. ex `:[line]z`  (❌, M, gated)
Both are *informational display* commands: `:display` lists buffers/marks/screens/
tags; ex `:z` prints a screenful of lines around an address with a type modifier.
The underlying data exists (cut buffers, marks, the line store), so the logic is
**addressable**, medium. They are gated on the same unresolved piece as the
deferred half of #37: govi routes multi-line informational output to the single
status line, where nvi paginates it into the body with a `+=+=` separator. Once
that info-message/overlay pagination is settled, `:display` and `:z` are
straightforward to render. Worth doing *with* that work, not before.

### 14. `autowrite` (aw)  (❌, M)
Auto-writes a modified buffer before commands that leave it: `:n`, `:rew`, `:tag`,
`^]`, `!`, `^^`. **Addressable**, medium: add the pre-write hook at those
navigation/tag/filter sites, honoring the `!`-suppression and the `readonly`/
`writeany` interactions. Real user-facing behavior, commonly relied on. (The
write-safety pieces it leans on -- the overwrite guard, `readonly`, `writeany` --
are now all in place from #42/#45.)

### 15. `backup`  (❌, M)
Keep a backup copy (by suffix/path) when `:write` overwrites a file.
**Addressable**, medium: at write time, copy the existing target to the backup
name before the new contents land. Mostly I/O plumbing; the care is in atomicity
and matching nvi's suffix semantics.

### 16. Marks -- intra-line column fixup  (🟡, M)
govi fixes marks at line granularity (a mark follows its line across inserts/
deletes of *whole lines*) but the intra-line column fixup is partial: editing text
*before* a mark on the same line should shift the mark's column. **Addressable**,
medium: extend the mark-adjust hook that already runs on line edits to also adjust
the stored column when the edit is on the mark's own line. Bounded but fiddly
(every insert/delete/substitute on a marked line must adjust).

### 17. Maps / abbreviations -- non-recursive only  (🟡, M)
nvi defaults `remap=on`, so a mapping's expansion is itself re-scanned for
mappings (recursive). govi is noremap-only. **Addressable**, medium: feed map
output back through the map resolver with a recursion/loop guard (and an iteration
cap, as nvi has) and honor the `remap` option to toggle it. The data structures
exist; this is a change to the expansion loop.

---

## DEFERRED (addressable in principle, but low value / by design)

### 1. `^E` / `^Y` over wrapped lines  (🟡)
RE-SCOPED (catalog #44): on non-wrapping files govi's `^E`/`^Y` match nvi exactly
(single steps, counts, EOF clamp, boundary frame, column maintenance). The only
residual is granularity over WRAPPED long lines: nvi scrolls by screen rows (its
soft map), so one `^E` can put a wrap-continuation segment at the top and keep the
cursor on the same file line; govi's viewport top is a whole logical line, so one
`^E` advances to the next file line. Closing it needs a sub-line viewport offset
threaded through the renderer/scroll/paging -- the same soft-map rework as #18.
Display-only (buffer always correct). Deferred as the long-line/soft-map cluster.

### 2. `^L` / `^R` repaint  (🟡)
Marked partial only because govi's terminal frontend repaints the screen on every
input, so an explicit repaint command is a no-op -- there is no corruption for it
to fix. This is correct by construction; "addressing" it would mean *introducing*
partial repaints just so a repaint command has something to do. Not worth it. The
GUI repaints automatically (—).

### 3. `Q` ex-mode display model  (🟡)
The command *works* (`:visual` returns, ex commands run). It is 🟡 because the
display is a scrolling line REPL (tty) / bottom-growing transcript (GUI) rather
than nvi's full-screen ex presentation. Matching nvi's exact ex-screen model is a
rework with little user benefit -- ex mode is rarely used interactively, and the
behavior is faithful. Deferred as a deliberate frontend choice.

### 18. Line wrapping -- no `@`-fill for a partial bottom line  (🟡)
RE-SCOPED (catalog #43): govi's static wrapped rendering already matches nvi in the
common cases probed; the `@`-fill did not reproduce there. The behavior that does
diverge is the soft-map gap shared with #1/#44 -- a logical line taller than the
screen, where nvi keeps prior context and the cursor at a screen-row offset while
govi scrolls the whole logical line to the top. nvi's `@` continuation fill is one
face of that screen-row (SMAP) machinery. Cosmetic/positional only; folded into the
long-line/soft-map cluster, not a standalone small fix.

### 19. File encodings (iconv)  (🟡)
govi is UTF-8 only; nvi uses iconv for arbitrary encodings. A full iconv-style
recode subsystem is a large effort for a case that is rare on a modern UTF-8
system. Deferred (not a non-objective -- could be revisited -- but not now).

### 20. Left-right scrolling (`leftright`/`sidescroll`)  (❌)
govi always wraps long lines; nvi can instead scroll horizontally. This is a
different long-line *rendering mode*, not a small toggle: it changes cursor-column
math, the gutter, and the redraw path. Moderate/high effort for a niche
preference, and related to the soft-map cluster (#1/#18). Deferred. (The options
remain settable so `:set all` matches.)

### 21. Command-line editing (`cedit`)  (❌)
An ex command-line edit window (edit the `:` line as a mini-buffer). Niche nvi
feature; low demand. Deferred.

---

## CAN'T be addressed here (blocked on split screens, or GUI structure)

### 13. `:shell` in GoVi.app  (❌, GUI)
The terminal frontend spawns an interactive shell (tcell suspend); the GUI has no
controlling terminal to hand a shell. It *could* host an embedded terminal panel,
but that is a sizable GUI feature for marginal value when the user already has
Terminal.app. Effectively a GUI design limit. (`!` filters still work in the GUI;
the terminal `:shell` is ✅.)

### 5. `^W` switch screens / 12. `:bg` `:fg` `:resize` `:script` `:vsplit` / 22. Split screens / windows  (❌)
All of these depend on the multi-window / split-screen subsystem, which govi does
not implement: a single engine view fills the frontend. Adding true nvi-style
splits is a large architectural change (multiple views over the buffer, focus
management, per-view viewport state, the screen-resize protocol). It is treated as
a **de-facto non-objective for the terminal frontend**: GoVi.app instead offers
native macOS windows and tabs (see "GoVi.app additions"), which cover the
user-facing need a different way. So these stay ❌ in the tty frontend by design,
not by oversight. `:script`/`:vsplit` ride on the same missing subsystem.

These also explain the related GoVi.app `—` cells (`^Z`/job control, `:suspend`/
`:stop`, Signals): those are terminal-process concepts with no AppKit analog; the
GUI uses the app lifecycle instead. Not gaps to "fix" -- N/A by frontend.

---

## Suggested order if someone picks these up

The write-path / data-safety cluster (secure, writeany, lock, overwrite guard) is
already DONE (#41/#42/#45); what's left of it is `autowrite` and `backup`.

1. **Finish the write-path cluster**: `autowrite` (#14) and `backup` (#15) -- they
   reuse the write/navigation hooks already added for #42/#45.
2. **Autoindent model**: input `^D`/`^T` + `0^D`/`^^D` (#6/#7) -- one coherent
   input-mode pass.
3. **Tag stack ex commands** (#8) and **`:mkexrc`** (#9).
4. **Recursive maps** (#17) and **intra-line mark fixup** (#16).
5. **Gated on info-message pagination**: `:display` (#10), ex `:z` (#11) -- do with
   that overlay work.
6. **Long-line / soft-map cluster** (architectural, deferred): `^E`/`^Y` over
   wrapped lines (#1), `@`-fill (#18), and left-right scrolling (#20) all want the
   same screen-row (SMAP) viewport model.
7. Leave split screens (#5/#12/#22), iconv (#19), cedit (#21), and GUI `:shell`
   (#13) unless priorities change.
