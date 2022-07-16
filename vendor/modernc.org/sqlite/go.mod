module modernc.org/sqlite

go 1.16

require (
	github.com/mattn/go-sqlite3 v1.14.12
	golang.org/x/sys v0.0.0-20211007075335-d3039528d8ac
	modernc.org/ccgo/v3 v3.16.6
	modernc.org/libc v1.16.7
	modernc.org/mathutil v1.4.1
	modernc.org/tcl v1.13.1
)

retract [v1.16.0, v1.17.2] // https://gitlab.com/cznic/sqlite/-/issues/100
