package src

import (
	"fmt"
	"runtime"
)

// Version stores the current verion of HTTPMS. It is set during building.
var Version = "v1.2-source"

func printVersionInformation() {
	fmt.Printf("HTTP Media Server (httpms) %s\n", Version)
	fmt.Printf("Build with %s\n", runtime.Version())
}
