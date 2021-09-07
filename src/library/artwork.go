package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ironsmile/euterpe/src/art"
)

// FindAndSaveAlbumArtwork implements the ArtworkManager interface for the local library.
// It would return a previously found artwork if any or try to find one in the
// filesystem or _on the internet_! This function returns ReadCloser and the caller
// is responsible for freeing the used resources by calling Close().
//
// When an artwork is found it will be saved in the database and once there it will be
// served from the db. Wait, wait! Serving binary files from the database?! Isn't that
// slow? Apparently no with sqlite3. See the following:
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
	size ImageSize,
) (io.ReadCloser, error) {
	r, foundSize, err := lib.findAndSaveAlbumArtworkOrOriginal(
		ctx,
		albumID,
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

	ret, _, err := lib.storeAlbumArtwork(albumID, converted, size)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (lib *LocalLibrary) findAndSaveAlbumArtworkOrOriginal(
	ctx context.Context,
	albumID int64,
	size ImageSize,
) (io.ReadCloser, ImageSize, error) {
	reader, foundSize, err := lib.albumArtworkFromDB(ctx, albumID, size)
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

	reader, err = lib.albumArtworkFromFS(ctx, albumID)
	if err == nil {
		return lib.storeAlbumArtwork(albumID, reader, OriginalImage)
	} else if err != ErrArtworkNotFound {
		return nil, size, err
	}

	reader, err = lib.albumArtworkFromInternet(ctx, albumID)
	if err == nil {
		return lib.storeAlbumArtwork(albumID, reader, OriginalImage)
	}

	if !errors.Is(err, art.ErrImageNotFound) && !errors.Is(err, ErrArtworkNotFound) {
		log.Printf("Finding album %d artwork on the internet error: %s\n", albumID, err)
	}

	if err := lib.saveAlbumArtworkNotFound(albumID); err != nil {
		return nil, size, err
	}

	return nil, size, ErrArtworkNotFound
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

func (lib *LocalLibrary) storeAlbumArtwork(
	albumID int64,
	artwork io.ReadCloser,
	size ImageSize,
) (io.ReadCloser, ImageSize, error) {
	defer artwork.Close()

	buff, err := ioutil.ReadAll(artwork)
	if err != nil {
		return nil, size, err
	}

	albumColumn := "artwork_cover"
	if size == SmallImage {
		albumColumn = "artwork_cover_small"
	}

	storeQuery := fmt.Sprintf(`
		INSERT INTO
			albums_artworks (album_id, %s, updated_at)
		VALUES				
			($1, $2, $3)
		ON CONFLICT (album_id) DO
		UPDATE SET
			%s = $2,
			updated_at = $3
	`, albumColumn, albumColumn)

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
		log.Printf("Error executing save artwork query: %s", err)
		return nil, size, err
	}

	return newBytesReadCloser(buff), size, nil
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
	if lib.artFinder == nil {
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

	cover, err := lib.artFinder.GetFrontImage(ctx, artistName, albumName)
	if errors.Is(err, art.ErrImageNotFound) {
		return nil, ErrArtworkNotFound
	}
	if err != nil {
		return nil, err
	}

	return newBytesReadCloser(cover), nil
}

// albumArtworkFromDB returns the original image from the database if one is stored,
// otherwise it returns the original (full size) image. Its second return argument
// could be used to determine which image was actually returned.
func (lib *LocalLibrary) albumArtworkFromDB(
	ctx context.Context,
	albumID int64,
	size ImageSize,
) (io.ReadCloser, ImageSize, error) {

	buff, unixTime, err := lib.albumArtworkFromDBForSize(ctx, albumID, size)
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
	buff, unixTime, err = lib.albumArtworkFromDBForSize(ctx, albumID, OriginalImage)
	if err != nil {
		return nil, size, err
	}

	if len(buff) < 1 {
		return selectNotFound(unixTime)
	}

	return newBytesReadCloser(buff), OriginalImage, nil
}

func (lib *LocalLibrary) albumArtworkFromDBForSize(
	ctx context.Context,
	albumID int64,
	size ImageSize,
) ([]byte, int64, error) {

	var (
		buff          []byte
		unixTime      int64
		blobColumn    = "artwork_cover"
		imageSQLQuery = `
			SELECT
				%s,
				updated_at
			FROM
				albums_artworks
			WHERE
				album_id = ?
		`
	)
	if size == SmallImage {
		blobColumn = "artwork_cover_small"
	}

	work := func(db *sql.DB) error {
		smt, err := db.PrepareContext(ctx, fmt.Sprintf(imageSQLQuery, blobColumn))

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
		return nil, 0, err
	}

	return buff, unixTime, nil
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

		// Artwork which is in the exact directory of the album should have slight
		// advantage. This is to cover cases where there are directories of albums
		// inside other albums.
		if filepath.Dir(path) == albumPath {
			pathScore += 4
		} else {
			pathScore -= 4
		}

		if pathScore > score {
			selectedArtwork = path
			score = pathScore
		}
	}

	log.Printf("Selected album [%d] artwork: %s", albumID, selectedArtwork)
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
