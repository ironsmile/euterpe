package src

import (
	"sync"

	"github.com/ironsmile/httpms/src/webserver"
)

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
