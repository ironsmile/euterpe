// Package src contains the de-facto main function of the application.
// It should set everything up, create a library and create a webserver.
//
// At the moment it is in package src because I import it from the project's root
// folder. This way the source is in the `src/` directory.
package src

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/ironsmile/httpms/src/config"
	"github.com/ironsmile/httpms/src/daemon"
	"github.com/ironsmile/httpms/src/helpers"
	"github.com/ironsmile/httpms/src/library"
	"github.com/ironsmile/httpms/src/webserver"
)

var (
	// PidFile is populated by an command line argument. Will be a filesystem path.
	// Nedomi will save its Process ID in this file.
	PidFile string

	// Debug is populated by an command line argument.
	Debug bool
)

func init() {
	pidUsage := "Pidfile. Default is [user_path]/pidfile.pid"
	pidDefault := "pidfile.pid"
	flag.StringVar(&PidFile, "p", pidDefault, pidUsage)

	flag.BoolVar(&Debug, "D", false, "Debug mode. Will log everything to the stdout.")
}

// Main is the only thing run in the project's root main.go file.
// For all intent and purposes this is the main function.
func Main() {
	flag.Parse()

	projRoot, err := helpers.ProjectRoot()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	err = ParseConfigAndStartWebserver(projRoot)

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

// SetupPidFileAndSignals creates a pidfile and starts a signal receiver goroutine
func SetupPidFileAndSignals(pidFile string, stopFunc context.CancelFunc) {
	helpers.SetUpPidFile(pidFile)

	signalChannel := make(chan os.Signal, 2)
	for _, sig := range daemon.StopSignals {
		signal.Notify(signalChannel, sig)
	}
	go func() {
		for range signalChannel {
			log.Println("Stop signal received. Removing pidfile and stopping.")
			stopFunc()
			helpers.RemovePidFile(pidFile)
		}
	}()
}

// Returns a new Library object using the application config.
// For the moment this is a LocalLibrary which will place its sqlite db file
// in the UserPath directory
func getLibrary(ctx context.Context, userPath string,
	cfg config.Config) (library.Library, error) {

	dbPath := helpers.AbsolutePath(cfg.SqliteDatabase, userPath)
	lib, err := library.NewLocalLibrary(ctx, dbPath)

	if err != nil {
		return nil, err
	}

	lib.ScanConfig = cfg.LibraryScan

	err = lib.Initialize()

	if err != nil {
		return nil, err
	}

	for _, path := range cfg.Libraries {
		lib.AddLibraryPath(path)
	}

	return lib, nil
}

// ParseConfigAndStartWebserver parses the config, sets the logfile, setups the
// pidfile, and makes an signal handler goroutine
func ParseConfigAndStartWebserver(projRoot string) error {

	var cfg config.Config
	err := cfg.FindAndParse()

	if err != nil {
		return err
	}

	userPath := filepath.Dir(cfg.UserConfigPath())

	if !Debug {
		err = helpers.SetLogsFile(helpers.AbsolutePath(cfg.LogFile, userPath))
		if err != nil {
			return err
		}
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	pidFile := helpers.AbsolutePath(PidFile, userPath)
	SetupPidFileAndSignals(pidFile, cancelCtx)
	defer helpers.RemovePidFile(pidFile)

	lib, err := getLibrary(ctx, userPath, cfg)
	if err != nil {
		return err
	}
	go lib.Scan()

	cfg.HTTPRoot = helpers.AbsolutePath(cfg.HTTPRoot, projRoot)

	srv := webserver.NewServer(ctx, cfg, lib)
	srv.Serve()
	srv.Wait()
	return nil
}
