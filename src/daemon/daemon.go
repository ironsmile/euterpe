// This package is resposible for making sure HTTPMS will run smoothly even
// after the calling terminal has been closed. For *nix systems this mean
// daemonizing. For Windows - I don't know yet.
package daemon

// This is the main function for this module. It should be run only once.
func Daemonize() error {
	return nil
}
