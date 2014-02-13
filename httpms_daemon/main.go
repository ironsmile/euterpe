// If you want to run httpms as a daemon use this binary. It assumes the httpms is
// properly installed. The main binary should be in the $PATH.

package main

import (
	"flag"
	"log"
	"os"
	"os/exec"

	"github.com/ironsmile/httpms/src/daemon"
)

var (
	PidFile string
)

func init() {
	pidUsage := "Pidfile. Default is [user_path]/pidfile.pid"
	pidDefault := "pidfile.pid"
	flag.StringVar(&PidFile, "p", pidDefault, pidUsage)
}

func main() {
	flag.Parse()

	if err := daemon.Daemonize(); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	var cmd *exec.Cmd

	cmd = exec.Command("httpms", "-p", PidFile)

	if out, err := cmd.Output(); err != nil {
		log.Println(err)
		os.Exit(1)
	} else {
		log.Print(out)
	}
}
