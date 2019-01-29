package library

import (
	"database/sql"
	"log"
	"os"
	"time"
)

// cleanUpDatabase walks through all database records and removes those which point
// to files which no longer exist. It also removes albums with no tracks into them.
func (lib *LocalLibrary) cleanUpDatabase() {
	lib.cleanupLock.RLock()
	alreadyRunning := lib.runningCleanup
	lib.cleanupLock.RUnlock()

	if alreadyRunning {
		log.Println("Previous cleanup operation is already running.")
		return
	}

	lib.cleanupLock.Lock()
	lib.runningCleanup = true
	lib.cleanupLock.Unlock()

	defer func() {
		lib.cleanupLock.Lock()
		lib.runningCleanup = false
		lib.cleanupLock.Unlock()
	}()

	lib.cleanupTracks()
}

// cleanupTracks walks through all tracks in the database and cleanups from it any
// which are not present on the filesystem.
func (lib *LocalLibrary) cleanupTracks() {
	var (
		cursor int
		limit  = 100
		total  = lib.getTableSize("tracks")
	)

	if total == 0 {
		return
	}

	for {
		var (
			tracks []track
			tr     track
		)

		getTracks := func(db *sql.DB) error {
			rows, err := db.Query(`
				SELECT
					id,
					fs_path
				FROM
					tracks
				ORDER BY
					id
				LIMIT ?, ?

			`, cursor, limit)
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				rows.Scan(&tr.id, &tr.fsPath)
				tracks = append(tracks, tr)
			}

			return nil
		}

		if err := lib.executeDBJobAndWait(getTracks); err != nil {
			log.Printf("Error getting tracks during cleanup: %s", err)
			return
		}

		cursor += limit

		if err := lib.checkAndRemoveTracks(tracks); err != nil {
			log.Printf("Error cleaning up tracks: %s", err)
			return
		}

		if cursor >= total {
			break
		}

		time.Sleep(5 * time.Second)
	}
}

// checkAndRemoveTracks makes a stat call for all tracks and removes from the db any
// which do not exist.
func (lib *LocalLibrary) checkAndRemoveTracks(tracks []track) error {
	for _, track := range tracks {
		_, err := os.Stat(track.fsPath)
		if err == nil || !os.IsNotExist(err) {
			continue
		}

		log.Printf("Cleaning up %d - '%s'\n", track.id, track.fsPath)
		lib.removeFile(track.fsPath)
	}

	return nil
}

type track struct {
	id     int64
	fsPath string
}
