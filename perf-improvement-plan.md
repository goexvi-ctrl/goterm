# GoVi performance improvement plan (proposal -- not yet executed)

Planning doc, written 2026-06-28 from the
benchmark + profiling in `perf_test.go`. It proposes work; it changes no code.

## 1. Where we are (evidence, not guesswork)

Mixed editing session, 10k-line file, best-of-2 CPU (see `perf_test.go`):

| editor | wall | CPU | vs vim |
|---|---|---|---|
| vim 9.1 | 2.8s | 2.8s | 1.0x |
| nvi | 8.6s | 8.4s | ~3x |
| **govi** | **26.3s** | **29.9s** | **~10x** |

Per-operation CPU (govi/vim, govi/nvi):

| op | govi/vim | govi/nvi | nature |
|---|---|---|---|
| search `n`, paging `^F^B` | 31-52x | 37-46x | per-keystroke full redraw |
| substitute `:%s///g` | 26x | 4.7x | regex over all lines |
| global `:g//s` | 10x | 3.8x | regex + per-line |
| shift `> <` | 24x | 1.9x | bulk line rewrite |

Profile of a redraw-heavy run (macOS `sample`, cumulative frames):

- `engine/buffer.(*original).line` -- **3130** samples (the hot path: `pread` +
  rune-decode + alloc of a line from the piece-table's on-disk original).
- Go GC/runtime (`scanObject`, `sweep`, `mallocgc`, `stringtoslicerune`) -- a large
  tail (per-line allocations create garbage).
- `github.com/gdamore/tcell` (the renderer) -- **~71** samples. **tcell is NOT the
  bottleneck**; do not spend effort there.

### Root causes (two, both upstream of the renderer)
1. **Cadence:** the frontend rebuilds the *entire* visible screen's text on every
   input event (`Frontend.processEvents -> engine.Input -> screen.screenLines ->
   lineRunes -> displayWidth -> buffer.line`), even for a pure cursor move. nvi/vim
   recompute almost nothing on navigation.
2. **Cost per line:** each `buffer.line` is a cold `pread`+decode+allocate from the
   piece-table original, with weak caching -- so every rebuild is expensive and
   generates GC garbage. This same call dominates the regex/substitute scans too.

## 2. Targets (directional)

- Navigation/redraw ops (search, paging, `h j k l`, `^F`/`^B`): from ~40x toward
  **near-free** (<= 2-3x of nvi/vim) -- they should do almost no work.
- Substitute/global: from 4-5x toward **<= ~2x of nvi**.
- Mixed session: from ~10x(vim)/~3x(nvi) toward **~3-4x(vim) / <= ~1.5x(nvi)**.

These are goals to steer by, not contractual numbers. The realistic ceiling is
nvi-class (govi is GC'd Go; vim's hand-tuned C regex/line store is a stretch).

## 3. Guardrails (non-negotiable)

This project's entire point is **bug-for-bug parity with nvi**. Performance work
must not regress behavior.

- After every change: rebuild `/Users/claude/bin/govi`, then run the full goterm
  comparison + coverage suites green:
  `go test -run 'TestDiverge|TestCoverage' .` -- 0 new DIVERGE.
- Run govi's own `internal/conformance` (engine-vs-nvi oracle) in the govi tree.
- Re-run the perf harness to confirm the change actually helped and nothing
  regressed: `PERF=1 ... go test -run 'TestPerfSession|TestPerfBreakdown' .`
- **Caching is the main correctness risk** (stale line content after an edit). Every
  cache must be invalidated on the exact mutation paths; add targeted tests that
  edit then read the same line/region.

Measurement hygiene already learned (carry forward):
- Same `TERM=ansi` for all editors; best-of-N; the insert/Esc-heavy "edit churn"
  mode is excluded (nvi/vim per-Esc timer artifact, not CPU).

## 4. Workstreams (ordered by impact-to-effort)

Each item: hypothesis -> investigate -> change -> expected impact -> verify -> risk.

### P0. Baseline + safety net  (S)
- Capture current `TestPerfSession`/`TestPerfBreakdown`/`TestPerfProfile` numbers as
  the baseline to beat (store in this doc or a CSV).
- Confirm the parity + conformance suites are green *before* touching anything, so
  later failures are attributable.
- Add a couple of perf "micro" cases if useful (e.g., pure `j`/`k` navigation, a
  single full-file `:%s`) for tighter attribution.

### P1. Stop rebuilding the whole screen every keystroke  (M-L) -- BIGGEST LEVER
- Hypothesis: navigation is ~40x because each input re-derives all ~24 visible rows
  via `screen.screenLines -> lineRunes -> buffer.line`. A cursor move changes 0-2
  rows of *content*; the rest is identical.
- Investigate: read `engine/screen.go` (`screenLines`, `lineRunes`, `displayWidth`)
  and `frontend/tcell` (the per-event render call). Find where the screen model is
  produced and whether anything is memoized between events.
- Change: dirty-tracking / incremental screen build -- keep the last computed screen
  model (per logical line: decoded runes + display cells) and recompute a row only
  when its underlying line, the viewport top, or wrapping changed. Cursor-only moves
  should recompute nothing but the cursor position. Hand tcell the same cell buffer
  (it already diffs and emits only changes).
- Expected: search/paging/navigation drop toward nvi/vim (near-free). Also shrinks
  the mixed-session number a lot, since real sessions are navigation-heavy.
- Verify: `paging`/`search` breakdown modes should fall ~10-30x; parity suites green
  (esp. anything touching rendering: `:set number`/`list`, wrapping, wide chars).
- Risk: medium-high -- redraw correctness (scrolling, wrap, `number`/`list` gutter,
  reverse-video status, wide/East-Asian cells). The goterm `DiffCells` tests are the
  safety net here.

### P2. A decoded-line cache in the buffer layer  (M) -- pairs with P1
- Hypothesis: `buffer.(*original).line` is 3130 samples because it re-`pread`s and
  re-decodes the same lines repeatedly (across keystrokes and across `:g`/`:%s`
  passes) with weak caching.
- Investigate: read `engine/buffer` (`original`, `Paged`, `line`, `Get`). Determine
  the current caching (none? per-page? per-line?) and the invalidation points.
- Change: add a bounded decoded-line cache (e.g., LRU keyed by line number +
  buffer-generation counter, or per-page decoded arrays) returning an immutable
  decoded line; bump the generation / evict affected entries on every edit.
- Expected: collapses the dominant leaf; helps redraw, search, and regex alike.
- Verify: breakdown `subst`/`global` improve; `(*original).line` drops in a re-
  profile; parity suites green; add an edit-then-read invalidation test.
- Risk: **staleness** -- the cache must be invalidated on every mutation
  (insert/delete/change/substitute/join/put/undo/redo, `:r`, filters). This is the
  highest-correctness-risk item; gate it behind the full parity + conformance run.

### P3. Cut per-line allocation + decode cost  (M)
- Hypothesis: the GC tail (`stringtoslicerune`, `mallocgc`, scan/sweep) comes from
  allocating a `string`/`[]rune` per line access; `runeWidth`/`displayWidth` add
  per-rune work.
- Change: ASCII fast path (most lines are ASCII -> skip rune decode and treat
  width=1); reuse scratch buffers (`sync.Pool` or per-screen reused slices) instead
  of allocating per line; compute display width lazily / cache it with the line.
- Expected: lower CPU and much less GC (the >100% cpu / GC threads shrink).
- Verify: re-profile shows the GC tail shrink; wide-char parity tests still pass.
- Risk: low-medium -- the ASCII fast path must fall back correctly for multibyte and
  control chars (`list` mode `^I`/`^X`, tabs).

### P4. Reduce piece-table syscalls  (M)
- Hypothesis: `pread`-per-line means a syscall (cgo `asmcgocall`) per line fetch;
  the big `asmcgocall` count is partly this.
- Change: read the original file in blocks/pages and serve lines from an in-memory
  page cache (or `mmap` the original read-only), so `line` is a memory slice, not a
  syscall. Likely subsumed by P2's cache but attacks the syscall/cgo tax directly.
- Expected: removes per-line syscalls from hot paths; complements P2/P3.
- Verify: `pread`/`asmcgocall` fall in the profile; large-file editing still correct
  (the piece-table's whole point -- watch memory use on multi-GB files).
- Risk: medium -- must not break the large-file/paged design or recovery.

### P5. Substitution / global throughput  (M)
- Hypothesis: `:%s`/`:g` are 4-5x nvi partly via line fetch (P2 helps) and partly
  via per-line regex on decoded runes + rebuilding lines.
- Investigate: the substitute/global loop in the engine; how the regex runs (bytes
  vs runes), whether the compiled RE is reused, and how replacement rebuilds lines.
- Change: run the match on bytes where the pattern allows (avoid rune conversion
  unless the RE needs it), reuse the compiled program across the whole range, and
  build the replacement with a reused buffer; only touch lines that actually match.
- Expected: `subst`/`global` move toward ~2x nvi.
- Verify: the ~55-case `:s`/`:g` parity battery + breakdown modes; numbers improve.
- Risk: medium -- regex semantics are pinned to nvi (BRE + Spencer quirks); do not
  alter matching behavior, only its execution cost.

### P6. Runtime / GC tuning  (S) -- opportunistic, do last
- Try `GOGC` higher / `GOMEMLIMIT`, and `sync.Pool` for the hottest allocations, as
  a cheap multiplier once allocations are already reduced (P3). Measure; keep only if
  it clearly helps without bloating idle memory.
- Risk: low; purely a tuning knob, no behavior change.

## 5. Sequencing & milestones

1. **P0** baseline + green suites.
2. **P1 + P2 together** (incremental redraw + line cache) -- the headline win;
   re-profile to confirm `buffer.line` and the redraw modes collapse.
3. **P3** (allocations) and **P4** (syscalls) -- shrink the GC/cgo tail.
4. **P5** (substitute/global) -- close the regex gap.
5. **P6** tuning, then re-run the full `TestPerfSession`/`TestPerfBreakdown` and
   record the new table against the P0 baseline.

Stop when navigation is near-free and the mixed session is nvi-class; chasing vim's
C regex beyond that has diminishing returns.

## 6. Risks & non-goals

- **Parity is sacred.** Any item that can't keep `TestDiverge*`/`TestCoverage*` and
  `internal/conformance` green is rolled back, regardless of speedup.
- **Not tcell.** The profile shows the renderer is ~2% of samples; replacing or
  re-tuning tcell is a non-goal.
- **Don't break large-file editing or recovery** when changing the piece-table/IO
  (P4); the paged store exists for multi-GB files.
- **Caching correctness** is the dominant risk (P2/P4); invest in invalidation tests.
- GoVi.app shares the engine, so P1-P5 (engine-level) benefit both frontends; P1's
  incremental-redraw work is partly frontend-specific (tcell vs AppKit) -- keep the
  shared screen-model improvements in the engine.

## 7. How to measure (already built)

```sh
# rebuild the binary under test
cd /Users/claude/src/nvi/govi && go build -o /Users/claude/bin/govi ./cmd/govi

# behavior must stay green (run from goterm)
cd /Users/claude/src/goterm && go test -run 'TestDiverge|TestCoverage' .

# perf: mixed session + per-op breakdown (nvi/vim/govi)
PERF=1 PERF_LINES=10000 PERF_ROUNDS=40 go test -run TestPerfSession -v -timeout 600s .
PERF=1 PERF_TIMEOUT=120 go test -run TestPerfBreakdown -v -timeout 600s .

# function-level profile of govi during redraw-heavy input
PERF=1 PERF_SAMPLE_OUT=/tmp/govi.sample.txt go test -run TestPerfProfile -v .
```
Compare each change against the P0 baseline; keep what helps and stays green.
