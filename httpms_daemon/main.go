// If you want to run httpms as a daemon use this binary. It assumes the httpms is
// properly installed. The main binary should be in the $PATH.

package main

import (
	"flag"
	"log"
	"log/syslog"
	"os"
	"os/exec"
	"path/filepath"

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

	myPlace, err := filepath.EvalSymlinks(os.Args[0])

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	myPlace, err = filepath.Abs(filepath.Dir(myPlace))

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	path, err := exec.LookPath(filepath.Join(myPlace, "httpms"))
	if err != nil {
		path, err = exec.LookPath("httpms")
		if err != nil {
			log.Println("Was not able to find httpms binary")
			os.Exit(1)
		}
	}

	logger, err := syslog.NewLogger(syslog.LOG_ERR, 0)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	if err := daemon.Daemonize(); err != nil {
		logger.Println(err)
		os.Exit(1)
	}

	if out, err := exec.Command(path, "-p", PidFile).Output(); err != nil {
		logger.Println(err)
		os.Exit(1)
	} else {
		logger.Println(out)
	}
}
