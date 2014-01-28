// The Main function of HTTPMS. It should set everything up, create a library, create
// a webserver and daemonize itself.
//
// At the moment it is in package src because I import it from the project's root
// folder.
package src

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/ironsmile/httpms/src/config"
	"github.com/ironsmile/httpms/src/daemon"
	"github.com/ironsmile/httpms/src/helpers"
	"github.com/ironsmile/httpms/src/library"
	"github.com/ironsmile/httpms/src/webserver"
)

// Returns a new Library object using the application config.
// For the moment this is a LocalLibrary which will place its sqlite db file
// in the UserPath directory
func getLibrary(userPath string, cfg config.Config) library.Library {
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

	return lib
}

var (
	PidFile string
)

func init() {
	pidUsage := "Pidfile. Default is [user_path]/pidfile.pid"
	pidDefault := "pidfile.pid"
	flag.StringVar(&PidFile, "p", pidDefault, pidUsage)
}

// This function is the only thing run in the project's root main.go file.
// For all intent and purposes this is the main function.
func Main() {
	flag.Parse()

	projRoot, err := helpers.ProjectRoot()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	if !daemon.Debug {
		err = daemon.Daemonize()
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}

	ParseConfigAndStartWebserver(projRoot)
}

// Does what the name says
func ParseConfigAndStartWebserver(projRoot string) {

	var cfg config.Config
	err := cfg.FindAndParse()

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	userPath := filepath.Dir(cfg.UserConfigPath())

	if !daemon.Debug {
		helpers.SetLogsFile(helpers.AbsolutePath(cfg.LogFile, userPath))
	}

	PidFile = helpers.AbsolutePath(PidFile, userPath)
	helpers.SetUpPidFile(PidFile)
	defer helpers.RemovePidFile(PidFile)

	lib := getLibrary(userPath, cfg)
	lib.Scan()

	cfg.HTTPRoot = helpers.AbsolutePath(cfg.HTTPRoot, projRoot)

	srv := webserver.NewServer(cfg, lib)
	srv.Serve()
	srv.Wait()
}
