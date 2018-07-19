package src

import (
	"fmt"
	"runtime"
)

var Version = "v1.2.0-development"

func printVersionInformation() {
	fmt.Printf("HTTP Media Server (httpms) %s\n", Version)
	fmt.Printf("Build with %s\n", runtime.Version())
}
