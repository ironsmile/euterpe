/*
Package version provides version information and utilities.
*/
package version

import (
	"fmt"
	"io"
	"runtime"
)

// Version stores the current version of Euterpe. It is set during building.
var Version = "dev-unreleased"

// Print writes a plain text version information in out.
func Print(out io.Writer) {
	fmt.Fprintf(out, "Euterpe Media Server %s\n", Version)
	fmt.Fprintf(out, "Build with %s\n", runtime.Version())
}
