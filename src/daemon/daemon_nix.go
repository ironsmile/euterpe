// +build linux darwin freebsd

// Package daemon is resposible for making sure HTTPMS will run smoothly even
// after the calling terminal has been closed. For *nix systems this mean
// daemonizing. For Windows - I don't know yet.
package daemon

import (
	"fmt"
	"os"
	"syscall"
)

// StopSignals contains all the signals which will make our daemon remove its pidfile
var StopSignals = []syscall.Signal{
	syscall.SIGINT,
	syscall.SIGKILL,
	syscall.SIGTERM,
}

// Daemonize is the main function for this module. It should be run only once.
func Daemonize() error {
	var ret uintptr
	var err syscall.Errno

	ret, _, err = syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	if err != 0 {
		return err
	}
	switch ret {
	case 0:
		break
	default:
		os.Exit(0)
	}

	st, e := syscall.Setsid()
	if st == -1 {
		return fmt.Errorf("Setsid returned %d", st)
	}
	if e != nil {
		return e
	}
	os.Chdir("/")

	f, e := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if e == nil {
		fd := f.Fd()
		syscall.Dup2(int(fd), int(os.Stdin.Fd()))
		syscall.Dup2(int(fd), int(os.Stdout.Fd()))
		syscall.Dup2(int(fd), int(os.Stderr.Fd()))
	}

	return nil
}
