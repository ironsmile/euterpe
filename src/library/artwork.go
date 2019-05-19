package library

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ironsmile/httpms/ca"
)

// FindAndSaveAlbumArtwork implements the ArtworkManager interface for the local library.
// It would return a previously found artwork if any or try to find one in the
// filesystem or _on the internet_! This function returns ReadCloser and the caller
// is resposible for freeing the used resources by calling Close().
//
// When an artwork is found it will be saved in the database and once there it will be
// served from the db. Wait, wait! Serving binary files from the database?! Isn't that
// slow? Apparantly no with sqlite3. See the following:
//
// * https://www.sqlite.org/intern-v-extern-blob.html
// * https://www.sqlite.org/fasterthanfs.html
//
// This behaviour have an additional bonus that artwork found on the internet will not
// be saved on the filesystem and thus "pollute" it with unexpected files. It will be
// nicely contained in the app's database.
//
// !TODO: Make sure there is no race conditions while getting/saving artwork for
// particular album. Wink, wink, the database.
func (lib *LocalLibrary) FindAndSaveAlbumArtwork(
	ctx context.Context,
	albumID int64,
) (io.ReadCloser, error) {
	reader, err := lib.albumArtworkFromDB(ctx, albumID)
	if err == ErrCachedArtworkNotFound {
		return nil, ErrArtworkNotFound
	} else if err == nil || err != ErrArtworkNotFound {
		return reader, err
	}

	if err := lib.aquireArtworkSem(ctx); err != nil {
		// When error is returned it means that the semaphore was not acquired.
		// So we can return safely without releaseing it.
		return nil, err
	}
	defer lib.releaseArtworkSem()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	reader, err = lib.albumArtworkFromFS(ctx, albumID)
	if err == nil {
		return lib.saveAlbumArtwork(albumID, reader)
	} else if err != ErrArtworkNotFound {
		return nil, err
	}

	reader, err = lib.albumArtworkFromInternet(ctx, albumID)
	if err == nil {
		return lib.saveAlbumArtwork(albumID, reader)
	}

	if err != ca.ErrImageNotFound && err != ErrArtworkNotFound {
		log.Printf("Finding album %d artwork on the internet error: %s\n", albumID, err)
	}

	if err := lib.saveAlbumArtworkNotFound(albumID); err != nil {
		return nil, err
	}

	return nil, ErrArtworkNotFound
}

// Used to limit the concurrent requests for getting artwork. On error the semaphore
// is not acquired. The caller *must not* try to release it.
func (lib *LocalLibrary) aquireArtworkSem(ctx context.Context) error {
	select {
	case lib.artworkSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (lib *LocalLibrary) releaseArtworkSem() {
	<-lib.artworkSem
}

func (lib *LocalLibrary) saveAlbumArtwork(
	albumID int64,
	artwork io.ReadCloser,
) (io.ReadCloser, error) {
	defer artwork.Close()

	buff, err := ioutil.ReadAll(artwork)
	if err != nil {
		return nil, err
	}

	work := func(db *sql.DB) error {
		stmt, err := db.Prepare(`
				INSERT OR REPLACE INTO
					albums_artworks (album_id, artwork_cover, updated_at)
				VALUES
					(?, ?, ?)
		`)

		if err != nil {
			return err
		}

		defer stmt.Close()

		_, err = stmt.Exec(albumID, buff, time.Now().Unix())

		if err != nil {
			return err
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing save artwork query: %s", err)
		return nil, err
	}

	return newBytesReadCloser(buff), nil
}

func (lib *LocalLibrary) saveAlbumArtworkNotFound(albumID int64) error {
	work := func(db *sql.DB) error {
		stmt, err := db.Prepare(`
				INSERT OR REPLACE INTO
					albums_artworks (album_id, updated_at)
				VALUES
					(?, ?)
		`)

		if err != nil {
			return err
		}

		defer stmt.Close()

		_, err = stmt.Exec(albumID, time.Now().Unix())
		if err != nil {
			return err
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing save artwork not found query: %s", err)
		return err
	}

	return nil
}

func (lib *LocalLibrary) albumArtworkFromInternet(
	ctx context.Context,
	albumID int64,
) (io.ReadCloser, error) {
	if lib.coverArtFinder == nil {
		return nil, ErrArtworkNotFound
	}

	var (
		albumName  string
		artistName string
		count      int
	)

	work := func(db *sql.DB) error {
		row, err := db.QueryContext(ctx, `
			SELECT
				name
			FROM
				albums
			WHERE
				id = ?
		`, albumID)

		if err != nil {
			return fmt.Errorf("query database: %s", err)
		}

		defer func(row *sql.Rows) {
			row.Close()
		}(row)

		if !row.Next() {
			return ErrAlbumNotFound
		}

		if err := row.Scan(&albumName); err != nil {
			return fmt.Errorf("scanning db result: %s", err)
		}

		row, err = db.QueryContext(ctx, `
			SELECT
				a.name,
				COUNT(*) as cnt
			FROM tracks AS t
			LEFT JOIN artists AS a ON a.id = t.artist_id
			WHERE album_id = ?
			GROUP BY artist_id
			ORDER BY cnt DESC
			LIMIT 1;
		`, albumID)

		if err != nil {
			return fmt.Errorf("query database: %s", err)
		}

		defer func(row *sql.Rows) {
			row.Close()
		}(row)

		if row.Next() {
			if err := row.Scan(&artistName, &count); err != nil {
				return fmt.Errorf("scanning db result: %s", err)
			}
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		return nil, err
	}

	cover, err := lib.coverArtFinder.GetFrontImage(ctx, artistName, albumName)
	if err == ca.ErrImageNotFound {
		return nil, ErrArtworkNotFound
	}
	if err != nil {
		return nil, err
	}

	return newBytesReadCloser(cover.Data), nil
}

func (lib *LocalLibrary) albumArtworkFromDB(
	ctx context.Context,
	albumID int64,
) (io.ReadCloser, error) {

	var (
		buff     []byte
		unixTime int64
	)

	work := func(db *sql.DB) error {
		smt, err := db.PrepareContext(ctx, `
			SELECT
				artwork_cover,
				updated_at
			FROM
				albums_artworks
			WHERE
				album_id = ?
		`)

		if err != nil {
			log.Printf("could not prepare album artwork sql statement: %s", err)
			return err
		}
		defer smt.Close()

		err = smt.QueryRowContext(ctx, albumID).Scan(&buff, &unixTime)
		if err == sql.ErrNoRows {
			return ErrArtworkNotFound
		} else if err != nil {
			log.Printf("error getting album cover from db: %s", err)
			return err
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		return nil, err
	}

	if len(buff) < 1 && time.Now().Before(time.Unix(unixTime, 0).Add(24*7*time.Hour)) {
		return nil, ErrCachedArtworkNotFound
	}

	return newBytesReadCloser(buff), nil
}

func (lib *LocalLibrary) albumArtworkFromFS(
	ctx context.Context,
	albumID int64,
) (io.ReadCloser, error) {
	albumPath, err := lib.GetAlbumFSPathByID(albumID)

	if err != nil {
		return nil, err
	}

	imagesRegexp := regexp.MustCompile(`(?i).*\.(png|gif|jpeg|jpg)$`)
	var possibleArtworks []string

	err = filepath.Walk(albumPath, func(path string, info os.FileInfo, err error) error {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip directories
			return nil
		}
		if imagesRegexp.MatchString(path) {
			possibleArtworks = append(possibleArtworks, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(possibleArtworks) < 1 {
		return nil, ErrArtworkNotFound
	}

	var (
		selectedArtwork string
		score           int
	)

	for _, path := range possibleArtworks {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		pathScore := 5

		fileBase := strings.ToLower(filepath.Base(path))

		if strings.HasPrefix(fileBase, "cover.") || strings.HasPrefix(fileBase, "front.") {
			pathScore = 15
		} else if strings.Contains(fileBase, "cover") || strings.Contains(fileBase, "front") {
			pathScore = 10
		} else if strings.Contains(fileBase, "artwork") {
			pathScore = 8
		}

		if strings.HasPrefix(fileBase, ".") {
			// Hidden file, it should have a lower score when compared to normal files.
			pathScore -= 4
		}

		if pathScore > score {
			selectedArtwork = path
			score = pathScore
		}
	}

	return os.Open(selectedArtwork)
}

// SaveAlbumArtwork implements the ArtworkManager interface for the local library.
//
// It saves the artwork in `r` in the database. It will read up to 5MB of data from
// `r` and if this limit is reached, the artwork is considered too big and will not
// be saved in the db.
func (lib *LocalLibrary) SaveAlbumArtwork(
	ctx context.Context,
	albumID int64,
	r io.Reader,
) error {
	var readLimit int64 = 5 * 1024 * 1024

	lr := &io.LimitedReader{
		R: r,
		N: readLimit,
	}

	buff, err := ioutil.ReadAll(lr)
	if err != nil && err != io.EOF {
		return fmt.Errorf(
			"reading the request body for storing album %d: %s",
			albumID,
			err,
		)
	}

	if int64(len(buff)) >= readLimit {
		return ErrArtworkTooBig
	}

	if len(buff) == 0 {
		return NewArtworkError("uploaded artwork is empty")
	}

	work := func(db *sql.DB) error {
		stmt, err := db.Prepare(`
			INSERT OR REPLACE INTO
				albums_artworks (album_id, artwork_cover, updated_at)
			VALUES
				(?, ?, ?)
		`)
		if err != nil {
			return err
		}

		defer stmt.Close()

		_, err = stmt.Exec(albumID, buff, time.Now().Unix())
		return err
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		return err
	}

	return nil
}

// RemoveAlbumArtwork removes the artwork from the library database.
//
// Note that this operation does not make sense for artwork which came from disk. Because
// future requests will find it again and store in the database.
func (lib *LocalLibrary) RemoveAlbumArtwork(ctx context.Context, albumID int64) error {
	return lib.saveAlbumArtworkNotFound(albumID)
}
