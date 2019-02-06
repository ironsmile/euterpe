package library

import (
	"database/sql"
	"fmt"
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
	lib.cleanupAlbums()
}

// cleanupTracks walks through all tracks in the database and cleanups from it any
// which are not present on the filesystem. It does that in batches with some rest
// between batches.
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

// cleanupAlbums walks through all albums in the database and cleanups from it any
// which have no associated tracks. It does that in batches with some rest between
// batches.
func (lib *LocalLibrary) cleanupAlbums() {
	var limit = 100

	for {
		var (
			albumIDs []int64
			albumID  int64
		)

		getAlbums := func(db *sql.DB) error {
			rows, err := db.Query(`
				SELECT
					a.id
				FROM albums a
				LEFT JOIN tracks t ON
					a.id = t.album_id
				WHERE
					t.id IS NULL
				GROUP BY
					a.id
				LIMIT ?

			`, limit)
			if err != nil {
				return err
			}
			defer rows.Close()

			for rows.Next() {
				rows.Scan(&albumID)
				albumIDs = append(albumIDs, albumID)
			}

			return nil
		}

		if err := lib.executeDBJobAndWait(getAlbums); err != nil {
			log.Printf("Error getting albums during cleanup: %s", err)
			return
		}

		if len(albumIDs) == 0 {
			break
		}

		if err := lib.checkAndRemoveAlbums(albumIDs); err != nil {
			log.Printf("Error cleaning up albums: %s", err)
			return
		}

		time.Sleep(5 * time.Second)
	}
}

// checkAndRemoveAlbums removes from the database the albums with IDs `albumIDs`
// but not before making sure there are no tracks asscociated with them.
func (lib *LocalLibrary) checkAndRemoveAlbums(albumIDs []int64) error {
	for _, albumID := range albumIDs {
		if err := lib.executeDBJobAndWait(func(db *sql.DB) error {
			var tracks int64

			rows, err := db.Query(`
				SELECT
					COUNT(*) as cnt
				FROM
					tracks
				WHERE
					album_id = ?
			`, albumID)
			if err != nil {
				return err
			}

			if !rows.Next() {
				return fmt.Errorf(
					"rows.Next returned false for COUNT SELECT query",
				)
			}

			err = rows.Scan(&tracks)
			rows.Close()

			if err != nil {
				return err
			}

			// Make sure there are no registered tracks for this album since
			// it was scheduled for removal.
			if tracks > 0 {
				return nil
			}

			_, err = db.Exec(`
				DELETE FROM albums
				WHERE id = ?
			`, albumID)
			if err != nil {
				return err
			}

			_, err = db.Exec(`
				DELETE FROM albums_artworks
				WHERE album_id = ?
			`, albumID)
			if err != nil {
				return err
			}

			return nil
		}); err != nil {
			log.Printf("Error deleting album %d: %s", albumID, err)
		}
	}

	return nil
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
