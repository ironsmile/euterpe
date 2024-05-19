package library

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	taglib "github.com/wtolson/go-taglib"
)

// Scan scans all of the folders in paths for media files. New files will be added to the
// database.
func (lib *LocalLibrary) Scan() {
	// Make sure there are no other scans working at the moment
	lib.waitScanLock.RLock()
	lib.walkWG.Wait()
	lib.waitScanLock.RUnlock()

	start := time.Now()

	lib.initializeWatcher()
	initialWait := lib.ScanConfig.InitialWait
	if !LibraryFastScan && initialWait > 0 {
		log.Printf("Pausing initial library scan for %s as configured", initialWait)
		time.Sleep(initialWait)
	}

	lib.waitScanLock.Lock()
	for _, path := range lib.paths {
		lib.walkWG.Add(1)
		go lib.scanPath(path)
	}
	lib.waitScanLock.Unlock()

	lib.waitScanLock.RLock()
	lib.walkWG.Wait()
	lib.waitScanLock.RUnlock()
	log.Printf("Scaning took %s", time.Since(start))

	start = time.Now()
	lib.cleanUpDatabase()
	log.Printf("Cleaning up took %s", time.Since(start))
}

// This is the goroutine which actually scans a library path.
// For now it ignores everything but the list of supported files. It is so
// because jplayer cannot play anything else. Sends every suitable
// file into the media channel
func (lib *LocalLibrary) scanPath(scannedPath string) {
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
			log.Printf("error while scanning %s: %s", path, err)
			return nil
		}

		if !info.IsDir() && lib.isSupportedFormat(path) {
			err := lib.AddMedia(path)
			if err != nil {
				log.Printf("Error adding `%s`: %s\n", path, err)
			}
		}

		lib.watchLock.RLock()
		if lib.watch != nil && info.IsDir() && !lib.noWatch {
			if err := lib.watch.Watch(path); err != nil {
				log.Printf("Starting a file system watch for %s failed: %s", path, err)
			}
		}
		lib.watchLock.RUnlock()

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
		log.Printf("error while walking %s: %s", scannedPath, err)
	}
}

// Rescan goes through the database and for every file reads the meta data again from
// the disk and updates it.
func (lib *LocalLibrary) Rescan(ctx context.Context) error {
	lib.runningRescan = true
	defer func() {
		lib.runningRescan = false
	}()

	const batchSize = 500
	var cursor int64

	for {
		mediaFiles, err := lib.getMediaFilenames(ctx, cursor, batchSize)
		if err != nil {
			return fmt.Errorf("error getting media files from the db: %w", err)
		}
		if len(mediaFiles) < 1 {
			break
		}
		cursor += int64(len(mediaFiles))

		for _, fileName := range mediaFiles {
			st, err := os.Stat(fileName)
			if err != nil {
				log.Printf("Filesystem error (stat) for %s: %s\n", fileName, err)
				continue
			}

			file, err := taglib.Read(fileName)
			if err != nil {
				log.Printf("Taglib error for %s: %s\n", fileName, err)
				continue
			}

			fi := fileInfo{
				Size:     st.Size(),
				FilePath: fileName,
				Modified: st.ModTime(),
			}
			if err := lib.insertMediaIntoDatabase(file, fi); err != nil {
				log.Printf("failed updating file %s: %s\n", fileName, err)
			}
			file.Close()
		}
	}

	return nil
}

// getMediaFilenames returns batchSize media files after moving the db offset at
// cursor size.
func (lib *LocalLibrary) getMediaFilenames(
	ctx context.Context,
	cursor,
	batchSize int64,
) ([]string, error) {
	var files []string
	work := func(db *sql.DB) error {
		stmt, err := db.PrepareContext(ctx, `
			SELECT
				fs_path
			FROM
				tracks
			LIMIT $1 OFFSET $2
		`)
		if err != nil {
			return fmt.Errorf("could not prepare statement: %w", err)
		}

		rows, err := stmt.QueryContext(ctx, batchSize, cursor)
		if err != nil {
			return fmt.Errorf("executing db query failed: %w", err)
		}

		for rows.Next() {
			var path string
			if err := rows.Scan(&path); err != nil {
				log.Printf("error scanning: %s\n", err)
			}

			files = append(files, path)
		}

		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		return files, fmt.Errorf(
			"getting files for cursor %d and batch size %d failed: %w",
			cursor,
			batchSize,
			err,
		)
	}

	return files, nil
}
