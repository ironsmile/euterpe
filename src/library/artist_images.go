package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"time"

	"github.com/ironsmile/httpms/src/art"
)

// FindAndSaveArtistImage implements the ArtistImageManager for the local library.
// It will return a previously saved into the database artist image if any or
// try to find one on the internet (assuming it is configured). This function
// returns ReadCloser and the caller is responsible for freeing the used resources
// by calling Close().
//
// When image for an artist is found on the internet then it will be saved in the
// database for later retrieval.
func (lib *LocalLibrary) FindAndSaveArtistImage(
	ctx context.Context,
	artistID int64,
) (io.ReadCloser, error) {
	reader, err := lib.artistImageFromDB(ctx, artistID)
	if err == ErrCachedArtworkNotFound {
		return nil, ErrArtworkNotFound
	} else if err == nil || err != ErrArtworkNotFound {
		return reader, err
	}

	if err := lib.aquireArtworkSem(ctx); err != nil {
		// When error is returned it means that the semaphore was not acquired.
		// So we can return safely without releasing it.
		return nil, err
	}
	defer lib.releaseArtworkSem()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	reader, err = lib.artistImageFromInternet(ctx, artistID)
	if err == nil {
		return lib.saveArtistImage(artistID, reader)
	}

	if errors.Is(err, art.ErrNoDiscogsAuth) {
		// pass, don't do anything. No need for logs when this is a result
		// from the server's configuration.
	} else if !errors.Is(err, art.ErrImageNotFound) &&
		!errors.Is(err, ErrArtworkNotFound) {
		log.Printf("Finding artist %d image on the internet error: %s\n", artistID, err)
	}

	if err := lib.saveArtistImageNotFound(artistID); err != nil {
		return nil, err
	}

	return nil, ErrArtworkNotFound
}

func (lib *LocalLibrary) artistImageFromDB(
	ctx context.Context,
	artistID int64,
) (io.ReadCloser, error) {
	var (
		buff     []byte
		unixTime int64
	)

	work := func(db *sql.DB) error {
		smt, err := db.PrepareContext(ctx, `
			SELECT
				image,
				updated_at
			FROM
				artists_images
			WHERE
				artist_id = ?
		`)

		if err != nil {
			log.Printf("could not prepare artist image sql statement: %s", err)
			return err
		}
		defer smt.Close()

		err = smt.QueryRowContext(ctx, artistID).Scan(&buff, &unixTime)
		if err == sql.ErrNoRows {
			return ErrArtworkNotFound
		} else if err != nil {
			log.Printf("error getting artist image from db: %s", err)
			return err
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		return nil, err
	}

	if len(buff) < 1 {
		if time.Now().Before(time.Unix(unixTime, 0).Add(24 * 7 * time.Hour)) {
			return nil, ErrCachedArtworkNotFound
		}
		return nil, ErrArtistNotFound
	}

	return newBytesReadCloser(buff), nil
}

func (lib *LocalLibrary) artistImageFromInternet(
	ctx context.Context,
	artistID int64,
) (io.ReadCloser, error) {
	if lib.artFinder == nil {
		return nil, ErrArtworkNotFound
	}

	var artistName string

	work := func(db *sql.DB) error {
		row, err := db.QueryContext(ctx, `
			SELECT
				name
			FROM
				artists
			WHERE
				id = ?
		`, artistID)

		if err != nil {
			return fmt.Errorf("query database: %s", err)
		}

		defer func(row *sql.Rows) {
			row.Close()
		}(row)

		if !row.Next() {
			return ErrArtistNotFound
		}

		if err := row.Scan(&artistName); err != nil {
			return fmt.Errorf("scanning db result: %s", err)
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		return nil, err
	}

	cover, err := lib.artFinder.GetArtistImage(ctx, artistName)
	if errors.Is(err, art.ErrImageNotFound) {
		return nil, ErrArtworkNotFound
	}
	if err != nil {
		return nil, err
	}

	return newBytesReadCloser(cover), nil
}

func (lib *LocalLibrary) saveArtistImage(
	albumID int64,
	image io.ReadCloser,
) (io.ReadCloser, error) {
	defer image.Close()

	buff, err := ioutil.ReadAll(image)
	if err != nil {
		return nil, err
	}

	work := func(db *sql.DB) error {
		stmt, err := db.Prepare(`
				INSERT OR REPLACE INTO
					artists_images (artist_id, image, updated_at)
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
		log.Printf("Error executing save artist image query: %s", err)
		return nil, err
	}

	return newBytesReadCloser(buff), nil
}

func (lib *LocalLibrary) saveArtistImageNotFound(artistID int64) error {
	work := func(db *sql.DB) error {
		stmt, err := db.Prepare(`
				INSERT OR REPLACE INTO
					artists_images (artist_id, updated_at)
				VALUES
					(?, ?)
		`)

		if err != nil {
			return err
		}

		defer stmt.Close()

		_, err = stmt.Exec(artistID, time.Now().Unix())
		if err != nil {
			return err
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf(
			"Error executing save artist image not found query: %s",
			err,
		)
		return err
	}

	return nil
}

// SaveArtistImage implements the ArtistImageManager interface for the local library.
//
// It saves the image in `r` in the database. It will read up to 5MB of data from
// `r` and if this limit is reached, the image is considered too big and will not
// be saved in the db.
func (lib *LocalLibrary) SaveArtistImage(
	ctx context.Context,
	artistID int64,
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
			"reading the request body for storing artist image %d: %s",
			artistID,
			err,
		)
	}

	if int64(len(buff)) >= readLimit {
		return ErrArtworkTooBig
	}

	if len(buff) == 0 {
		return NewArtworkError("uploaded artist image is empty")
	}

	work := func(db *sql.DB) error {
		stmt, err := db.Prepare(`
			INSERT OR REPLACE INTO
				artists_images (artist_id, image, updated_at)
			VALUES
				(?, ?, ?)
		`)
		if err != nil {
			return err
		}

		defer stmt.Close()

		_, err = stmt.Exec(artistID, buff, time.Now().Unix())
		return err
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		return err
	}

	return nil
}

// RemoveArtistImage removes particular artist image from the database.
func (lib *LocalLibrary) RemoveArtistImage(ctx context.Context, artistID int64) error {
	return lib.saveArtistImageNotFound(artistID)
}
