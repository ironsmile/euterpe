// The Main function of HTTPMS. It should set everything up, create a library, create
// a webserver and daemonize itself.
//
// At the moment it is in package src because I import it from the project's root
// folder.
package src

import (
	"log"
	"os"
	"path/filepath"

	"github.com/ironsmile/httpms/src/config"
	"github.com/ironsmile/httpms/src/helpers"
	"github.com/ironsmile/httpms/src/library"
	"github.com/ironsmile/httpms/src/webserver"
)

// Returns ServerConfig using the application Config
func getServerConfig(cfg *config.Config) webserver.ServerConfig {
	var wsCfg webserver.ServerConfig
	wsCfg.Address = cfg.Listen
	wsCfg.Root = "http_root"

	if cfg.SSL {
		wsCfg.SSL = true
		wsCfg.SSLCert = cfg.SSLCertificate.Crt
		wsCfg.SSLKey = cfg.SSLCertificate.Key
	}

	if cfg.Auth {
		wsCfg.Auth = true
		wsCfg.AuthUser = cfg.Authenticate.User
		wsCfg.AuthPass = cfg.Authenticate.Password
	}

	return wsCfg
}

// Returns a new Library object using the application config.
// For the moment this is a LocalLibrary which will place its sqlite db file
// in the UserPath directory
func getLibrary(userPath string, cfg *config.Config) library.Library {
	lib, err := library.NewLocalLibrary(filepath.Join(userPath, "httpms.db"))

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	err = lib.Initialize()

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	for _, path := range cfg.Libraries {
		lib.AddLibraryPath(path)
	}

	lib.Scan()

	return lib
}

// This function is the only thing run in the project's root main.go file.
// For all intent and purposes this is the main function.
func Main() {

	userPath, err := helpers.ProjectUserPath()

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	cfg := new(config.Config)
	err = cfg.FindAndParse()

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	wsCfg := getServerConfig(cfg)
	lib := getLibrary(userPath, cfg)

	log.Printf("%#v\n", cfg)
	log.Printf("%#v\n", wsCfg)

	helpers.SetLogsFile(filepath.Join(userPath, "logfile"))

	srv := webserver.NewServer(wsCfg, lib)

	srv.Serve()

	srv.Wait()
}
