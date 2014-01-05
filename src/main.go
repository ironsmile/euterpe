// The Main function of HTTPMS. It should set everything up, create a library, create
// a webserver and daemonize itself.
//
// At the moment it is in package src because I import it from the project's root
// folder.
package src

import (
	"sync"

	"github.com/ironsmile/httpms/src/webserver"
)

// This function is the only thing run in the project's root main.go file.
// For all intent and purposes this is the main function.
func Main() {
	var wg sync.WaitGroup

	var wsCfg webserver.ServerConfig
	wsCfg.Address = ":8080"
	wsCfg.Root = "http_root"
	wsCfg.WaitGroup = &wg

	wg.Add(1)
	go webserver.Serve(wsCfg)

	wg.Wait()
}
