// The Main function of HTTPMS. It should set everything up, create a library, create
// a webserver and daemonize itself.
//
// At the moment it is in package src because I import it from the project's root
// folder.
package src

import (
	"log"
	"os"

	"github.com/ironsmile/httpms/src/config"
	"github.com/ironsmile/httpms/src/helpers"
	"github.com/ironsmile/httpms/src/library"
	"github.com/ironsmile/httpms/src/webserver"
)

// Returns a new Library object using the application config.
// For the moment this is a LocalLibrary which will place its sqlite db file
// in the UserPath directory
func getLibrary(userPath string, cfg *config.Config) library.Library {
	dbPath := helpers.AbsolutePath(cfg.SqliteDatabase, userPath)
	lib, err := library.NewLocalLibrary(dbPath)

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

	lib := getLibrary(userPath, cfg)

	log.Printf("%#v\n", cfg)

	helpers.SetLogsFile(helpers.AbsolutePath(cfg.LogFile, userPath))

	srv := webserver.NewServer(*cfg, lib)

	srv.Serve()

	srv.Wait()
}
