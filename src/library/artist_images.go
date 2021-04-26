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
	size ImageSize,
) (io.ReadCloser, error) {
	r, foundSize, err := lib.findAndSaveArtistImageOrOriginal(
		ctx,
		artistID,
		size,
	)
	if err != nil {
		return r, err
	}

	if foundSize == size {
		return r, nil
	}

	// The returned artwork will have to be closed in any case now since it
	// will not be returned to the caller.
	defer func() {
		_ = r.Close()
	}()

	if foundSize != OriginalImage {
		return nil, fmt.Errorf("internal error, size mismatch")
	}

	converted, err := lib.scaleImage(ctx, r, size)
	if err != nil {
		return nil, fmt.Errorf("error scaling image: %w", err)
	}

	ret, _, err := lib.storeArtistImage(artistID, converted, size)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (lib *LocalLibrary) findAndSaveArtistImageOrOriginal(
	ctx context.Context,
	artistID int64,
	size ImageSize,
) (io.ReadCloser, ImageSize, error) {
	reader, foundSize, err := lib.artistImageFromDB(ctx, artistID, size)
	if err == ErrCachedArtworkNotFound {
		return nil, size, ErrArtworkNotFound
	} else if err == nil || err != ErrArtworkNotFound {
		return reader, foundSize, err
	}

	if err := lib.aquireArtworkSem(ctx); err != nil {
		// When error is returned it means that the semaphore was not acquired.
		// So we can return safely without releasing it.
		return nil, size, err
	}
	defer lib.releaseArtworkSem()

	if err := ctx.Err(); err != nil {
		return nil, size, err
	}

	reader, err = lib.artistImageFromInternet(ctx, artistID)
	if err == nil {
		return lib.storeArtistImage(artistID, reader, OriginalImage)
	}

	if errors.Is(err, art.ErrNoDiscogsAuth) {
		// pass, don't do anything. No need for logs when this is a result
		// from the server's configuration.
	} else if !errors.Is(err, art.ErrImageNotFound) &&
		!errors.Is(err, ErrArtworkNotFound) {
		log.Printf("Finding artist %d image on the internet error: %s\n", artistID, err)
	}

	if err := lib.saveArtistImageNotFound(artistID); err != nil {
		return nil, size, err
	}

	return nil, size, ErrArtworkNotFound
}

func (lib *LocalLibrary) artistImageFromDB(
	ctx context.Context,
	artistID int64,
	size ImageSize,
) (io.ReadCloser, ImageSize, error) {
	buff, unixTime, err := lib.artistImageFromDBForSize(ctx, artistID, size)
	if err != nil {
		return nil, size, err
	}

	if len(buff) >= 1 {
		// The image with the desires size has been found!
		return newBytesReadCloser(buff), size, nil
	}

	selectNotFound := func(lastChanged int64) (io.ReadCloser, ImageSize, error) {
		if time.Now().Before(time.Unix(lastChanged, 0).Add(notFoundCacheTTL)) {
			return nil, size, ErrCachedArtworkNotFound
		}
		return nil, size, ErrArtworkNotFound
	}

	// No image in the database. Is either a normal "not found" for images which
	// haven't  been queried recently. For everything else it is "cached not found"
	// which means that all the channels for obtaining the image have been tried out
	// recently and nothing has been found.
	if size == OriginalImage {
		return selectNotFound(unixTime)
	}

	// No image of the desired size was found. Let us try and see if the original image
	// is in the database and use it to generate the desired size.
	buff, unixTime, err = lib.artistImageFromDBForSize(ctx, artistID, OriginalImage)
	if err != nil {
		return nil, size, err
	}

	if len(buff) < 1 {
		return selectNotFound(unixTime)
	}

	return newBytesReadCloser(buff), OriginalImage, nil
}

func (lib *LocalLibrary) artistImageFromDBForSize(
	ctx context.Context,
	artistID int64,
	size ImageSize,
) ([]byte, int64, error) {

	var (
		buff          []byte
		unixTime      int64
		blobColumn    = "image"
		imageSQLQuery = `
			SELECT
				%s,
				updated_at
			FROM
				artists_images
			WHERE
				artist_id = ?
		`
	)
	if size == SmallImage {
		blobColumn = "image_small"
	}

	work := func(db *sql.DB) error {
		smt, err := db.PrepareContext(ctx, fmt.Sprintf(imageSQLQuery, blobColumn))

		if err != nil {
			log.Printf("could not prepare album artwork sql statement: %s", err)
			return err
		}
		defer smt.Close()

		err = smt.QueryRowContext(ctx, artistID).Scan(&buff, &unixTime)
		if err == sql.ErrNoRows {
			return ErrArtworkNotFound
		} else if err != nil {
			log.Printf("error getting album cover from db: %s", err)
			return err
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		return nil, 0, err
	}

	return buff, unixTime, nil
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

func (lib *LocalLibrary) storeArtistImage(
	albumID int64,
	image io.ReadCloser,
	size ImageSize,
) (io.ReadCloser, ImageSize, error) {
	defer image.Close()

	buff, err := ioutil.ReadAll(image)
	if err != nil {
		return nil, size, err
	}

	imageColumn := "image"
	if size == SmallImage {
		imageColumn = "image_small"
	}

	storeQuery := fmt.Sprintf(`
		INSERT INTO
			artists_images (artist_id, %s, updated_at)
		VALUES				
			($1, $2, $3)
		ON CONFLICT (artist_id) DO
		UPDATE SET
			%s = $2,
			updated_at = $3
	`, imageColumn, imageColumn)

	work := func(db *sql.DB) error {
		stmt, err := db.Prepare(storeQuery)

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
		return nil, size, err
	}

	return newBytesReadCloser(buff), size, nil
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
