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

	go lib.watchEventRoutine()
}

// This function is resposible for selecting the watcher events
func (lib *LocalLibrary) watchEventRoutine() {
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
		case <-lib.ctx.Done():
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
		} else {
			// It was a directory... probably
			lib.watch.RemoveWatch(event.Name)
			lib.removeDirectory(event.Name)
		}
		return
	}

	if event.IsCreate() && st.IsDir() {
		lib.watch.Watch(event.Name)

		//!TODO: the next two lines are actually a race condition. An alternative way
		// for achieving this must be found. It seems that calling `Add` on the wait
		// group is the problem. One can detect it with `go test -race`.
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
