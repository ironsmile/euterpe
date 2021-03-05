package library

import (
	"fmt"
	"log"
	"os"

	"github.com/howeyc/fsnotify"
)

// Creates the directory watcher if none was created before. On failure logs the
// problem and leaves the watcher unintialized. LocalLibrary should work even
// without a watch.
func (lib *LocalLibrary) initializeWatcher() {
	lib.watchLock.Lock()
	defer lib.watchLock.Unlock()

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
	defer log.Println("Directory watcher event receiver stopped.")

	lib.watchLock.RLock()
	if lib.watch == nil {
		lib.watchLock.RUnlock()
		log.Printf("lib.watch is nil. Stopping the watch event routine.")
		return
	}
	lib.watchLock.RUnlock()

	defer func() {
		lib.watchLock.Lock()
		lib.watch.Close()
		lib.watchLock.Unlock()
	}()

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
			lib.watchLock.Lock()
			err := lib.watch.RemoveWatch(event.Name)
			lib.watchLock.Unlock()
			if err != nil {
				fmt.Printf("error removing watcher for %s: %s\n", event.Name, err)
			}

			lib.removeDirectory(event.Name)
		}
		return
	}

	if event.IsCreate() && st.IsDir() {
		if err := lib.watch.Watch(event.Name); err != nil {
			fmt.Printf("error starting a watcher for %s: %s\n", event.Name, err)
		}

		lib.waitScanLock.Lock()
		lib.walkWG.Add(1)
		lib.waitScanLock.Unlock()

		lib.scanPath(event.Name)
		return
	}

	if event.IsCreate() && !st.IsDir() {
		if lib.isSupportedFormat(event.Name) {
			if err := lib.AddMedia(event.Name); err != nil {
				fmt.Printf("error adding newly created file: %s\n", err)
			}
		}
		return
	}

	if event.IsModify() && !st.IsDir() {
		if lib.isSupportedFormat(event.Name) {
			lib.removeFile(event.Name)
			if err := lib.AddMedia(event.Name); err != nil {
				fmt.Printf("error adding modified file: %s\n", err)
			}
		}
		return
	}
}
