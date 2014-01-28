package daemon

import "flag"

var (
	Debug bool
)

func init() {
	flag.BoolVar(&Debug, "D", false, "Debug mode. Does not daemonize.")
}
