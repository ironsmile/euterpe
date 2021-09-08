// +build windows

package daemon

import "os"

var StopSignals []os.Signal = []os.Signal{
	os.Interrupt,
	os.Kill,
}
