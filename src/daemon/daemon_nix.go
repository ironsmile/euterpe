// +build !windows

package daemon

import "syscall"

// StopSignals contains all the signals which will make our daemon remove its pidfile
var StopSignals = []syscall.Signal{
	syscall.SIGINT,
	syscall.SIGKILL,
	syscall.SIGTERM,
}
