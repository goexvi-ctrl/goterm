module goterm

go 1.26.1

require (
	github.com/mattn/go-runewidth v0.0.24
	github.com/pborman/ansi v1.1.0
)

require github.com/clipperhouse/uax29/v2 v2.2.0 // indirect

replace github.com/pborman/ansi => ./ansi
