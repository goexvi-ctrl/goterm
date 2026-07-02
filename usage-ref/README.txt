usage-ref/ -- reference dumps of :viusage and :exusage for nvi and govi.

Untracked, for offline reference only. NOT part of GOTERM_DIVERGENCES.md.
Captured 2026-06-28 on a 240x120 terminal (so nothing pages).

Files:
  nvi.viusage.txt    nvi  :viusage  (all vi-mode commands)
  nvi.exusage.txt    nvi  :exusage  (all ex-mode commands)
  govi.viusage.txt   govi :viusage
  govi.exusage.txt   govi :exusage
  viusage.raw.diff.txt   diff -u nvi.viusage  govi.viusage  (literal lines)
  exusage.raw.diff.txt   diff -u nvi.exusage  govi.exusage
  usage.commands.compare.txt   normalized which-commands-each-lists set diff
  mkcompare.py       generates usage.commands.compare.txt from the .txt dumps

Formatting differs between editors (nvi one glyph per line + right-aligned
"name:"; govi groups glyphs like "a A i I o O insert text" and left-aligns), so
the raw diffs are noisy. usage.commands.compare.txt normalizes that, but its glyph
extraction is heuristic -- treat it as a guide and confirm against the raw dumps.

Regenerate the captures:
  cd /Users/claude/src/goterm
  CAPTURE_USAGE=1 go test -run TestCaptureUsage -v .
Regenerate diffs + comparison:
  cd usage-ref
  diff -u nvi.viusage.txt govi.viusage.txt > viusage.raw.diff.txt
  diff -u nvi.exusage.txt govi.exusage.txt > exusage.raw.diff.txt
  python3 mkcompare.py

The generator test (usagecap_test.go) is gated behind CAPTURE_USAGE so a normal
`go test` does not run it.
