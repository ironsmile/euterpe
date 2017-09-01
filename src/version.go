package src

import (
	"fmt"
	"runtime"
)

const version = "v1.1.0"

func printVersionInformation() {
	fmt.Printf("HTTP Media Server (httpms) %s\n", version)
	fmt.Printf("Build with %s\n", runtime.Version())
}
