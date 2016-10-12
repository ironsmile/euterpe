package library

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

// Scan scans all of the folders in paths for media files. New files will be added to the
// database.
//!TODO: make scan also remove files which have been deleted since the previous scan
func (lib *LocalLibrary) Scan() {
	// Make sure there are no other scans working at the moment
	lib.scanWG.Wait()

	start := time.Now()

	lib.initializeWatcher()
	initialWait := lib.ScanConfig.InitialWait
	if !LibraryFastScan && initialWait > 0 {
		log.Printf("Pausing initial library scan for %s as configured", initialWait)
		time.Sleep(initialWait)
	}

	for _, path := range lib.paths {
		lib.walkWG.Add(1)
		go lib.scanPath(path, lib.mediaChan)
	}

	lib.scanWG.Add(1)
	go func() {
		defer func() {
			log.Printf("Walking took %s", time.Since(start))
			lib.scanWG.Done()
		}()
		lib.walkWG.Wait()
	}()

	go func() {
		lib.WaitScan()
		log.Printf("Scaning took %s", time.Since(start))
	}()
}

// WaitScan blocks the current goroutine until the scan has been finished
func (lib *LocalLibrary) WaitScan() {
	lib.scanWG.Wait()
	lib.waitForDBWriterIdleSignal()
}

// This is the goroutine which actually scans a library path.
// For now it ignores everything but the list of supported files. It is so
// because jplayer cannot play anything else. Sends every suitable
// file into the media channel
func (lib *LocalLibrary) scanPath(scannedPath string, media chan<- string) {
	start := time.Now()

	defer func() {
		log.Printf("Walking %s took %s", scannedPath, time.Since(start))
		lib.walkWG.Done()
	}()

	filesPerOperation := lib.ScanConfig.FilesPerOperation
	sleepPerOperation := lib.ScanConfig.SleepPerOperation

	var scannedFiles int64

	walkFunc := func(path string, info os.FileInfo, err error) error {

		if err != nil {
			log.Println(err)
			return nil
		}

		if lib.isSupportedFormat(path) {
			media <- path
		}

		if lib.watch != nil && info.IsDir() {
			lib.watch.Watch(path)
		}

		scannedFiles++

		if !LibraryFastScan && filesPerOperation > 0 &&
			scannedFiles >= filesPerOperation && sleepPerOperation > 0 {

			log.Printf("Scan limit of %d files reached for [%s], sleeping for %s",
				filesPerOperation, scannedPath, sleepPerOperation)

			time.Sleep(sleepPerOperation)
			scannedFiles = 0
		}

		return nil
	}

	err := filepath.Walk(scannedPath, walkFunc)

	if err != nil {
		log.Println(err)
	}
}
