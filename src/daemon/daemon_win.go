// +build windows

package daemon

var StopSignals []os.Signal = []os.Signal{
	os.Interrupt,
	os.Kill,
}

func Daemonize() error {
	return nil
}
