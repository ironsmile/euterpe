// +build linux darwin freebsd openbsd

// Package daemon is resposible for making sure HTTPMS will run smoothly even
// after the calling terminal has been closed. For *nix systems this mean
// daemonizing. For Windows - I don't know yet.
package daemon

import "syscall"

// StopSignals contains all the signals which will make our daemon remove its pidfile
var StopSignals = []syscall.Signal{
	syscall.SIGINT,
	syscall.SIGKILL,
	syscall.SIGTERM,
}
