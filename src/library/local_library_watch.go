package library

import (
	"log"
	"os"

	"github.com/howeyc/fsnotify"
)

// Creates the directory watcher if none was created before. On failure logs the
// problem and leaves the watcher unintialized. LocalLibrary should work even
// without a watch.
func (lib *LocalLibrary) initializeWatcher() {
	if lib.watch != nil {
		return
	}
	newWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Directory watcher was not initialized properly. ")
		log.Printf("New files will not be added to the library. Reason: ")
		log.Println(err)
		return
	}
	lib.watch = newWatcher
	lib.watchClosedChan = make(chan bool)

	go lib.watchEventRoutine()
}

// Stops the filesystem watching and all supporing it goroutines.
func (lib *LocalLibrary) stopWatcher() {
	if lib.watch != nil {
		lib.watchClosedChan <- true
		lib.watch.Close()
		lib.watch = nil
		close(lib.watchClosedChan)
	}
}

// This function is resposible for selecting the watcher events
func (lib *LocalLibrary) watchEventRoutine() {

	// To make sure we will not write in the database at the same time as the
	// scanning goroutines we will wait them to end.
	lib.WaitScan()
	defer func() {
		log.Println("Directory watcher event receiver stopped.")
	}()

	if lib.watch == nil {
		return
	}

	for {
		select {
		case ev := <-lib.watch.Event:
			if ev == nil {
				return
			}
			lib.handleWatchEvent(ev)
		case err := <-lib.watch.Error:
			if err == nil {
				return
			}
			log.Println("Directory watcher error:", err)
		case <-lib.watchClosedChan:
			return
		}
	}
}

// Deals with the watcher events.
//  * new directories should be watched and they themselves scanned
//  * new files should be added to the library
//  * deleted files should be removed from the library
//  * deleted directories should be unwatched
//  * modfied files should be updated in the database
//  * renamed ...
func (lib *LocalLibrary) handleWatchEvent(event *fsnotify.FileEvent) {

	if event.IsAttrib() {
		// The event was just an attribute change
		return
	}

	st, stErr := os.Stat(event.Name)

	if stErr != nil && !event.IsRename() && !event.IsDelete() {
		log.Printf("Watch event stat received error: %s\n", stErr.Error())
		return
	}

	if event.IsDelete() || event.IsRename() {
		if lib.isSupportedFormat(event.Name) {
			// This is a file
			lib.removeFile(event.Name)
			return
		} else {
			// It was a directory... probably
			lib.watch.RemoveWatch(event.Name)
			lib.removeDirectory(event.Name)
			return
		}
	}

	if event.IsCreate() && st.IsDir() {
		// fmt.Printf("Adding watch for %s\n", event.Name)
		lib.watch.Watch(event.Name)
		lib.walkWG.Add(1)
		go lib.scanPath(event.Name, lib.mediaChan)
		return
	}

	if event.IsCreate() && !st.IsDir() {
		if lib.isSupportedFormat(event.Name) {
			lib.mediaChan <- event.Name
		}
		return
	}

	if event.IsModify() && !st.IsDir() {
		if lib.isSupportedFormat(event.Name) {
			lib.removeFile(event.Name)
			lib.mediaChan <- event.Name
		}
		return
	}
}
