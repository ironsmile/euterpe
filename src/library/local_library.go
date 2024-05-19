package library

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/howeyc/fsnotify"
	taglib "github.com/wtolson/go-taglib"

	// Blind import is the way a SQL driver is imported. This is the proposed way
	// from the golang documentation.
	_ "github.com/mattn/go-sqlite3"

	"github.com/ironsmile/euterpe/src/art"
	"github.com/ironsmile/euterpe/src/config"
	"github.com/ironsmile/euterpe/src/helpers"
	"github.com/ironsmile/euterpe/src/scaler"
)

const (
	// UnknownLabel will be used in case some media tag is missing. As a consequence
	// if there are many files with missing title, artist and album only
	// one of them will be saved in the library.
	UnknownLabel = "Unknown"

	// SQLiteMemoryFile can be used as a database path for the sqlite's Open method.
	// When using it, one would create a memory database which does not write
	// anything on disk. See https://www.sqlite.org/inmemorydb.html for more info
	// on the subject of in-memory databases. We are using a shared cache because
	// this causes all the different connections in the database/sql pool to be
	// connected to the same "memory file". Without this. every new connection
	// would end up creating a new memory database.
	SQLiteMemoryFile = "file::memory:?cache=shared"

	// sqlSchemaFile is the file which contains the initial SQL Schema for the
	// media library. It must be one of the files in `sqlFilesFS`.
	sqlSchemaFile = "library_schema.sql"
)

var (
	// LibraryFastScan is a flag, populated by the -fast-library-scan argument.
	//
	// When `false` (the default), scanning the local library will honour the
	// configuration for occasional sleeping while scanning the file system.
	//
	// When `true`, scanning will be as fast as possible. This may generate high
	// IO load for the duration of the scan.
	LibraryFastScan bool

	// ErrNotFound is returned when particular resource cannot be found in the library.
	ErrNotFound = errors.New("not found")

	// ErrAlbumNotFound is returned when no album could be found for particular operation.
	ErrAlbumNotFound = fmt.Errorf("Album: %w", ErrNotFound)

	// ErrArtistNotFound is returned when no artist could be found for particular operation.
	ErrArtistNotFound = fmt.Errorf("Artist: %w", ErrNotFound)

	// ErrArtworkNotFound is returned when no artwork can be found for particular album.
	ErrArtworkNotFound = NewArtworkError(fmt.Errorf("Artwork: %w", ErrNotFound))

	// ErrCachedArtworkNotFound is returned when the database has been queried and
	// its cache says the artwork was not found in the recent past. No need to continue
	// searching further once you receive this error.
	ErrCachedArtworkNotFound = NewArtworkError(
		fmt.Errorf("Artwork (cached): %w", ErrNotFound),
	)

	// ErrArtworkTooBig is returned from operation when the artwork is too big for it to
	// handle.
	ErrArtworkTooBig = NewArtworkError(errors.New("Artwork Is Too Big"))
)

// ArtworkError represents some kind of artwork error.
type ArtworkError struct {
	Err error
}

// Error implements the error interface.
func (a *ArtworkError) Error() string {
	return a.Err.Error()
}

func (a *ArtworkError) Unwrap() error {
	return a.Err
}

// NewArtworkError returns a new artwork error which will have `err` as message.
func NewArtworkError(err error) *ArtworkError {
	return &ArtworkError{Err: err}
}

func init() {
	flag.BoolVar(&LibraryFastScan, "fast-library-scan", false,
		"Do not honour the configuration set in 'library_scan'. With this flag,\n"+
			"scanning will be done as fast as possible. This may be useful when\n"+
			"running the daemon for the first time with big libraries.")
}

// LocalLibrary implements the Library interface. Will represent files found on the
// local storage
type LocalLibrary struct {
	// The configuration for how to scan the libraries.
	ScanConfig config.ScanSection

	database string         // The location of the library's database
	paths    []string       // FS locations which contain the library's media files
	db       *sql.DB        // Database handler
	walkWG   sync.WaitGroup // Used to log how much time scanning took

	// If something needs to work with the database it has to construct
	// a DatabaseExecutable and send it through this channel.
	dbExecutes chan DatabaseExecutable

	// artworkSem is used to make sure there are no more than certain amount
	// of artwork resolution tasks at a given moment.
	artworkSem chan struct{}

	// Directory watcher
	watch     *fsnotify.Watcher
	watchLock *sync.RWMutex

	ctx           context.Context
	ctxCancelFunc context.CancelFunc

	waitScanLock sync.RWMutex

	artFinder art.Finder

	fs         fs.FS
	sqlFilesFS fs.FS

	imageScaler scaler.Scaler

	// cleanupLock is used to secure a thread safe access to the runningCleanup property.
	cleanupLock *sync.RWMutex

	// runningCleanup shows whether there is an already running clean-up.
	runningCleanup bool

	// runningRescan shows that at the moment a complete rescan is running.
	runningRescan bool

	// When noWatch is set then no file system watchers will be created
	// for the scanned directories.
	noWatch bool
}

// Close closes the database connection. It is safe to call it as many times as you want.
func (lib *LocalLibrary) Close() {
	lib.ctxCancelFunc()
	lib.db.Close()
}

// AddLibraryPath adds a library directory to the list of libraries which will be
// scanned and consequently watched.
func (lib *LocalLibrary) AddLibraryPath(path string) {
	if _, err := fs.Stat(lib.fs, path); err != nil {
		log.Printf("error adding path: %s", err)
		return
	}

	lib.paths = append(lib.paths, path)
}

// Search searches in the library. Will match against the track's name, artist and album.
func (lib *LocalLibrary) Search(ctx context.Context, args SearchArgs) []SearchResult {
	searchTerm := fmt.Sprintf("%%%s%%", args.Query)

	var output []SearchResult
	work := func(db *sql.DB) error {
		limitCount := int64(-1)
		if args.Count > 0 {
			limitCount = int64(args.Count)
		}

		orderBy := "al.name, t.number"
		where := []string{strings.Join(
			[]string{
				"t.name LIKE @searchTerm",
				"al.name LIKE @searchTerm",
				"at.name LIKE @searchTerm",
			},
			" OR ",
		)}

		queryArgs := []any{
			sql.Named("searchTerm", searchTerm),
			sql.Named("offset", args.Offset),
			sql.Named("count", limitCount),
		}

		rows, err := queryTracks(ctx, db, where, orderBy, queryArgs)
		if err != nil {
			log.Printf("Search query not successful: %s\n", err.Error())
			return nil
		}

		defer rows.Close()
		for rows.Next() {
			res, err := scanTrack(rows)
			if err != nil {
				log.Printf("Error scanning search result: %s\n", err)
				continue
			}

			output = append(output, res)
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing search db work: %s", err)
		return output
	}
	return output
}

// SearchAlbums searches the local library for albums. See Library.SearchAlbums
// for more.
func (lib *LocalLibrary) SearchAlbums(ctx context.Context, args SearchArgs) []Album {
	searchTerm := fmt.Sprintf("%%%s%%", args.Query)

	var output []Album
	work := func(db *sql.DB) error {
		limitCount := int64(-1)
		if args.Count > 0 {
			limitCount = int64(args.Count)
		}

		rows, err := db.QueryContext(ctx, `
			SELECT
				t.album_id as album_id,
				al.name as album,
				CASE WHEN COUNT(DISTINCT t.artist_id) = 1
					THEN at.name
					ELSE "Various Artists"
					END AS artist,
				COUNT(t.id) as songCount,
				SUM(t.duration) as duration,
				MAX(us.last_played) as last_played,
				SUM(us.play_count) as play_count,
				asr.favourite,
				asr.user_rating
			FROM
				tracks as t
					LEFT JOIN albums as al ON al.id = t.album_id
					LEFT JOIN artists as at ON at.id = t.artist_id
					LEFT JOIN user_stats as us ON us.track_id = t.id
					LEFT JOIN albums_stats as asr ON asr.album_id = t.album_id
			WHERE
				t.name LIKE ? OR
				al.name LIKE ? OR
				at.name LIKE ?
			GROUP BY
				t.album_id
			ORDER BY
				al.name, t.album_id
			LIMIT
				?, ?
		`, searchTerm, searchTerm, searchTerm, args.Offset, limitCount)
		if err != nil {
			log.Printf("Search album query not successful: %s\n", err.Error())
			return nil
		}

		defer rows.Close()
		for rows.Next() {
			var (
				res        Album
				lastPlayed sql.NullInt64
				playCount  sql.NullInt64
				fav        sql.NullInt64
				rating     sql.NullInt16
			)

			err := rows.Scan(
				&res.ID, &res.Name, &res.Artist,
				&res.SongCount, &res.Duration, &lastPlayed,
				&playCount, &fav, &rating,
			)
			if err != nil {
				log.Printf("Error scanning search album result: %s\n", err)
				continue
			}
			if lastPlayed.Valid {
				res.LastPlayed = lastPlayed.Int64
			}
			if playCount.Valid {
				res.Plays = playCount.Int64
			}
			if fav.Valid {
				res.Favourite = fav.Int64
			}
			if rating.Valid {
				res.Rating = uint8(rating.Int16)
			}

			output = append(output, res)
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing search album db work: %s", err)
		return output
	}
	return output
}

// SearchArtists searches for and returns artists which match the search arguments.
func (lib *LocalLibrary) SearchArtists(ctx context.Context, args SearchArgs) []Artist {
	searchTerm := fmt.Sprintf("%%%s%%", args.Query)

	var output []Artist
	work := func(db *sql.DB) error {
		limitCount := int64(-1)
		if args.Count > 0 {
			limitCount = int64(args.Count)
		}

		rows, err := db.QueryContext(ctx, `
			SELECT
				ar.id,
				ar.name,
				(SELECT COUNT(DISTINCT(tr.album_id))
					FROM tracks tr
					WHERE tr.artist_id = ar.id) as albumsCount,
				ars.favourite,
				ars.user_rating
			FROM
				artists ar
				LEFT JOIN artists_stats as ars ON ars.artist_id = ar.id
			WHERE
				ar.name LIKE ?
			ORDER BY
				ar.name, ar.id
			LIMIT
				?, ?
		`, searchTerm, args.Offset, limitCount)
		if err != nil {
			log.Printf("Search artist query not successful: %s\n", err.Error())
			return nil
		}

		defer rows.Close()
		for rows.Next() {
			var (
				res    Artist
				fav    sql.NullInt64
				rating sql.NullInt16
			)

			err := rows.Scan(
				&res.ID, &res.Name, &res.AlbumCount, &fav, &rating,
			)
			if err != nil {
				log.Printf("Error scanning search artist result: %s\n", err)
				continue
			}
			if fav.Valid {
				res.Favourite = fav.Int64
			}
			if rating.Valid {
				res.Rating = uint8(rating.Int16)
			}

			output = append(output, res)
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing search artist db work: %s", err)
		return output
	}
	return output
}

// GetFilePath returns the filesystem path for a file specified by its ID.
func (lib *LocalLibrary) GetFilePath(ctx context.Context, ID int64) string {
	var filePath string
	work := func(db *sql.DB) error {
		smt, err := db.PrepareContext(ctx, `
			SELECT
				fs_path
			FROM
				tracks
			WHERE
				id = ?
		`)
		if err != nil {
			log.Printf("Error getting file path: %s\n", err)
			return nil
		}

		defer smt.Close()

		err = smt.QueryRowContext(ctx, ID).Scan(&filePath)
		if err != nil {
			log.Printf("Error file path query row: %s\n", err)
			return nil
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing get file path db work: %s", err)
		return filePath
	}
	return filePath
}

// GetAlbumFiles satisfies the Library interface
func (lib *LocalLibrary) GetAlbumFiles(ctx context.Context, albumID int64) []TrackInfo {
	var (
		output []TrackInfo

		where     = []string{"t.album_id = @albumID"}
		orderBy   = "al.name, t.number"
		queryArgs = []any{sql.Named("albumID", albumID)}
	)
	work := func(db *sql.DB) error {
		rows, err := queryTracks(ctx, db, where, orderBy, queryArgs)
		if err != nil {
			log.Printf("Query for getting albym files not successful: %s\n", err.Error())
			return nil
		}

		defer rows.Close()
		for rows.Next() {
			res, err := scanTrack(rows)
			if err != nil {
				return fmt.Errorf("scanning error: %w", err)
			}

			output = append(output, res)
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing get album files db work: %s", err)
		return output
	}
	return output
}

// GetTrack returns information for particular track identified by its
// media ID.
func (lib *LocalLibrary) GetTrack(ctx context.Context, trackID int64) (TrackInfo, error) {
	var res TrackInfo
	work := func(db *sql.DB) error {
		row := db.QueryRowContext(ctx, dbTracksQuery+`
			WHERE
				t.id = ?
		`, trackID)

		track, err := scanTrack(row)
		if err != nil && errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			log.Printf("Error getting track information: %s", err)
			return fmt.Errorf("getting track info error: %w", err)
		}

		res = track
		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		return res, err
	}
	return res, nil
}

// GetArtist returns information for particular artist.
func (lib *LocalLibrary) GetArtist(
	ctx context.Context,
	artistID int64,
) (Artist, error) {
	query := `
		SELECT
			ar.name,
			COUNT(DISTINCT(tr.album_id)) as album_count,
			ars.favourite,
			ars.user_rating
		FROM tracks tr
			LEFT JOIN artists as ar ON ar.id = tr.artist_id
			LEFT JOIN artists_stats as ars ON ars.artist_id = tr.artist_id
		WHERE
			tr.artist_id = ?
		GROUP BY
			tr.artist_id
	`
	var res Artist

	work := func(db *sql.DB) error {
		row := db.QueryRowContext(ctx, query, artistID)

		var (
			fav    sql.NullInt64
			rating sql.NullInt16
		)
		if err := row.Scan(
			&res.Name,
			&res.AlbumCount,
			&fav,
			&rating,
		); err != nil {
			return fmt.Errorf("sql query for artist info failed: %w", err)
		}
		res.ID = artistID
		if fav.Valid {
			res.Favourite = fav.Int64
		}
		if rating.Valid {
			res.Rating = uint8(rating.Int16)
		}

		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		return res, err
	}

	return res, nil
}

// GetAlbum returns information for particular album.
func (lib *LocalLibrary) GetAlbum(
	ctx context.Context,
	albumID int64,
) (Album, error) {
	query := `
		SELECT
			al.name as album_name,
			CASE WHEN COUNT(DISTINCT tr.artist_id) = 1
			THEN ar.name
			ELSE "Various Artists"
			END AS arist_name,
			COUNT(tr.id) as album_songs,
			SUM(tr.duration) as album_duration,
			SUM(us.play_count) as album_plays,
			MAX(us.last_played) as last_played,
			als.favourite,
			als.user_rating
		FROM tracks tr
			LEFT JOIN artists as ar ON ar.id = tr.artist_id
			LEFT JOIN albums_stats as als ON als.album_id = tr.album_id
			LEFT JOIN albums as al ON al.id = tr.album_id
			LEFT JOIN user_stats us ON us.track_id = tr.id
		WHERE
			tr.album_id = ?
		GROUP BY
			tr.album_id
	`
	var res Album

	work := func(db *sql.DB) error {
		row := db.QueryRowContext(ctx, query, albumID)

		var (
			fav        sql.NullInt64
			rating     sql.NullInt16
			plays      sql.NullInt64
			lastPlayed sql.NullInt64
		)
		if err := row.Scan(
			&res.Name,
			&res.Artist,
			&res.SongCount,
			&res.Duration,
			&plays,
			&lastPlayed,
			&fav,
			&rating,
		); err != nil {
			return fmt.Errorf("sql query for artist info failed: %w", err)
		}
		res.ID = albumID
		if fav.Valid {
			res.Favourite = fav.Int64
		}
		if rating.Valid {
			res.Rating = uint8(rating.Int16)
		}
		if plays.Valid {
			res.Plays = plays.Int64
		}
		if lastPlayed.Valid {
			res.LastPlayed = lastPlayed.Int64
		}

		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		return res, err
	}

	return res, nil
}

// RecordTrackPlay updates the `user_stats` table in the database.
//
// play_count and last_played are updated only if a sufficient time has
// passed since the previous value of last_played. This sufficient time is
// calculated based on the length of the media file.
func (lib *LocalLibrary) RecordTrackPlay(
	ctx context.Context,
	mediaID int64,
	atTime time.Time,
) error {
	work := func(db *sql.DB) error {
		query := `
			INSERT INTO user_stats (track_id, last_played, play_count)
			VALUES (@mediaID, @unixTime, 1)
			ON CONFLICT(track_id) DO UPDATE SET
				last_played = @unixTime,
				play_count = play_count + 1
			WHERE
				last_played + (
					SELECT
						duration / 1000 as dur
					FROM
						tracks
					WHERE
						id = @mediaID
				) / 3 < @unixTime;
		`
		unixTime := atTime.Unix()

		_, err := db.ExecContext(
			ctx, query,
			sql.Named("mediaID", mediaID),
			sql.Named("unixTime", unixTime),
		)
		return err
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		return fmt.Errorf("sql query error: %w", err)
	}

	return nil
}

// GetArtistAlbums returns all the albums which this artist has an at least
// one track in.
func (lib *LocalLibrary) GetArtistAlbums(
	ctx context.Context,
	artistID int64,
) []Album {
	var albums []Album

	work := func(db *sql.DB) error {
		var artistName string

		row := db.QueryRowContext(ctx, `
			SELECT
				name
			FROM
				artists
			WHERE
				id = ?
		`, artistID)
		if err := row.Scan(&artistName); err != nil {
			return fmt.Errorf("scanning for artist name error: %w", err)
		}

		rows, err := db.QueryContext(ctx, `
			SELECT
				t.album_id,
				a.name,
				COUNT(t.id) as songsCount,
				SUM(t.duration) as duration,
				MAX(us.last_played) as last_played,
				SUM(us.play_count) as play_count,
				als.favourite,
				als.user_rating
			FROM
				tracks t
					LEFT JOIN albums a ON a.id = t.album_id
					LEFT JOIN user_stats as us ON us.track_id = t.id
					LEFT JOIN albums_stats as als ON als.album_id = t.album_id
			WHERE
				t.artist_id = ?
			GROUP BY
				t.album_id
		`, artistID)
		if err != nil {
			log.Printf("GetArtistAlbums query not successful: %s\n", err.Error())
			return nil
		}

		defer rows.Close()
		for rows.Next() {
			res := Album{
				Artist: artistName,
			}

			var (
				lastPlayed sql.NullInt64
				playCount  sql.NullInt64
				fav        sql.NullInt64
				rating     sql.NullInt16
			)

			err := rows.Scan(
				&res.ID,
				&res.Name,
				&res.SongCount,
				&res.Duration,
				&lastPlayed,
				&playCount,
				&fav,
				&rating,
			)
			if err != nil {
				return fmt.Errorf("scanning for GetArtistAlbums error: %w", err)
			}
			if lastPlayed.Valid {
				res.LastPlayed = lastPlayed.Int64
			}
			if playCount.Valid {
				res.Plays = playCount.Int64
			}
			if fav.Valid {
				res.Favourite = fav.Int64
			}
			if rating.Valid {
				res.Rating = uint8(rating.Int16)
			}

			albums = append(albums, res)
		}

		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing get artist albums db work: %s", err)
		return albums
	}

	return albums
}

// Removes the file from the library. That means finding it in the database and
// removing it from there.
func (lib *LocalLibrary) removeFile(filePath string) {

	fullPath, err := filepath.Abs(filePath)

	if err != nil {
		log.Printf("Error removing %s: %s\n", filePath, err.Error())
		return
	}

	work := func(db *sql.DB) error {
		_, err := db.Exec(`
			DELETE FROM tracks
			WHERE fs_path = ?
		`, fullPath)
		if err != nil {
			log.Printf("Error removing %s: %s\n", fullPath, err.Error())
		}

		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing remove file db work: %s", err)
	}
}

// Removes files which belong in this directory from the library.
func (lib *LocalLibrary) removeDirectory(dirPath string) {

	// Adding slash at the end to make sure we are always removing directories
	deleteMatch := fmt.Sprintf("%s/%%", strings.TrimRight(dirPath, "/"))

	work := func(db *sql.DB) error {
		_, err := db.Exec(`
			DELETE FROM tracks
			WHERE fs_path LIKE ?
		`, deleteMatch)
		if err != nil {
			log.Printf("Error removing %s: %s\n", dirPath, err.Error())
		}

		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing remove dir db work: %s", err)
	}
}

// Determines if the file will be saved to the database. Only media files which
// jplayer can use are saved.
func (lib *LocalLibrary) isSupportedFormat(path string) bool {
	supportedFormats := []string{
		".mp3",
		".ogg",
		".oga",
		".wav",
		".fla",
		".flac",
		".m4a",
		".opus",
		".webm",
		".mp4",
	}

	base := filepath.Base(path)
	ext := filepath.Ext(base)
	if base == ext {
		// This is a file such as "path/to/.hidden". There is no point in
		// checking these. They really don't have extension. What is after
		// the dot is the actual file name. It is just hidden.
		return false
	}

	for _, format := range supportedFormats {
		if strings.EqualFold(ext, format) {
			return true
		}
	}
	return false
}

// AddMedia adds a file specified by its file system name to the library. Will create the
// needed Artist, Album if necessary.
func (lib *LocalLibrary) AddMedia(filename string) error {
	filename = filepath.Clean(filename)

	if lib.MediaExistsInLibrary(filename) {
		return nil
	}

	if _, err := fs.Stat(lib.fs, filename); err != nil {
		return err
	}

	file, err := taglib.Read(filename)

	if err != nil {
		return fmt.Errorf("Taglib error for %s: %s", filename, err.Error())
	}

	defer file.Close()

	return lib.insertMediaIntoDatabase(file, filename)
}

// insertMediaIntoDatabase accepts an already parsed media info object, its path.
// The method inserts this media into the library database.
func (lib *LocalLibrary) insertMediaIntoDatabase(file MediaFile, filePath string) error {
	artist := strings.TrimSpace(file.Artist())
	artistID, err := lib.setArtistID(artist)
	if err != nil {
		return err
	}

	fileDir := filepath.Dir(filePath)

	album := strings.TrimSpace(file.Album())
	albumID, err := lib.setAlbumID(album, fileDir)

	if err != nil {
		return err
	}

	trackNumber := int64(file.Track())
	if trackNumber == 0 {
		trackNumber = helpers.GuessTrackNumber(filePath)
	}

	title := strings.TrimSpace(file.Title())
	_, err = lib.setTrackID(
		title,
		filePath,
		trackNumber,
		artistID,
		albumID,
		file.Length().Milliseconds(),
	)
	return err
}

// MediaExistsInLibrary checks if the media file with file system path "filename" has
// been added to the library already.
func (lib *LocalLibrary) MediaExistsInLibrary(filename string) bool {
	var res bool

	work := func(db *sql.DB) error {
		smt, err := db.Prepare(`
			SELECT
				count(id)
			FROM
				tracks
			WHERE
				fs_path = ?
		`)
		if err != nil {
			res = false
			return fmt.Errorf("could not prepare sql statement: %s", err)
		}
		defer smt.Close()

		var count int
		err = smt.QueryRow(filename).Scan(&count)

		if err != nil {
			res = false
			return fmt.Errorf("error checking whether media exists already: %s", err)
		}

		res = (count >= 1)
		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error on executing db job: %s", err)
	}

	return res
}

// GetArtistID returns the id for this artist. When missing or on error
// returns that error.
func (lib *LocalLibrary) GetArtistID(artist string) (int64, error) {
	var artistID int64

	work := func(db *sql.DB) error {
		smt, err := db.Prepare(`
			SELECT
				id
			FROM
				artists
			WHERE
				name = ?
		`)
		if err != nil {
			return err
		}

		defer smt.Close()

		var id int64
		err = smt.QueryRow(artist).Scan(&id)

		if err != nil {
			return err
		}

		artistID = id
		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		return 0, err
	}

	return artistID, nil
}

// Sets a new ID for this artist if it is new to the library. If not, returns
// its current id.
func (lib *LocalLibrary) setArtistID(artist string) (int64, error) {
	if len(artist) < 1 {
		artist = UnknownLabel
	}

	id, err := lib.GetArtistID(artist)
	if err == nil {
		return id, nil
	}

	var lastInsertID int64
	work := func(db *sql.DB) error {
		stmt, err := db.Prepare(`
				INSERT INTO
					artists (name)
				VALUES
					(?)
		`)
		if err != nil {
			return err
		}

		defer stmt.Close()

		res, err := stmt.Exec(artist)
		if err != nil {
			return err
		}

		lastInsertID, _ = res.LastInsertId()
		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		return 0, err
	}

	newID, err := lib.GetArtistID(artist)
	if err != nil {
		return lastInsertID, fmt.Errorf(
			"getting the ID of inserted artist failed: %w", err)
	}

	log.Printf("Inserted artist id: %d, name: %s\n", newID, artist)
	if lastInsertID != newID {
		// In case this log is never seen for a long time it would mean that
		// the LastInsertId() bug has been fixed and it is probably safe to
		// remove the second SQL request which explicitly queries the DB for
		// the ID.
		log.Printf(
			"Wrong ID returned for artist `%s` by .LastInsertId(). "+
				"Returned: %d, actual: %d.",
			artist,
			lastInsertID,
			newID,
		)
	}

	return newID, nil
}

// GetAlbumID returns the id for this album. When missing or on error
// returns that error.
func (lib *LocalLibrary) GetAlbumID(album string, fsPath string) (int64, error) {
	var albumID int64

	work := func(db *sql.DB) error {
		smt, err := db.Prepare(`
			SELECT
				id
			FROM
				albums
			WHERE
				name = ? AND
				fs_path = ?
		`)
		if err != nil {
			return err
		}

		defer smt.Close()

		var id int64
		err = smt.QueryRow(album, fsPath).Scan(&id)
		if err != nil {
			return err
		}

		albumID = id
		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		return 0, err
	}

	return albumID, nil
}

// Sets a new ID for this album if it is new to the library. If not, returns
// its current id. Albums with the same name but by different locations need to have
// separate IDs hence the fsPath parameter.
func (lib *LocalLibrary) setAlbumID(album string, fsPath string) (int64, error) {
	if len(album) < 1 {
		album = UnknownLabel
	}

	id, err := lib.GetAlbumID(album, fsPath)
	if err == nil {
		return id, nil
	}

	var lastInsertID int64
	work := func(db *sql.DB) error {
		stmt, err := db.Prepare(`
				INSERT INTO
					albums (name, fs_path)
				VALUES
					(?, ?)
		`)
		if err != nil {
			return err
		}

		defer stmt.Close()

		res, err := stmt.Exec(album, fsPath)
		if err != nil {
			return fmt.Errorf("executing album insert: %w", err)
		}

		lastInsertID, _ = res.LastInsertId()
		return nil
	}
	if err := lib.executeDBJobAndWait(work); err != nil {
		return 0, err
	}

	// For some reason the sql.Result.LastInsertId() function does not always
	// return the correct ID. This might be a problem with the particular SQL
	// driver used. In any case, explicitly selecting it is the safest option.
	newID, err := lib.GetAlbumID(album, fsPath)
	if err != nil {
		return 0, fmt.Errorf("could not get ID of inserted album: %s", err)
	}

	log.Printf("Inserted album id: %d, name: %s, path: %s\n", newID, album, fsPath)
	if lastInsertID != newID {
		// In case this log is never seen for a long time it would mean that
		// the LastInsertId() bug has been fixed and it is probably safe to
		// remove the second SQL request which explicitly queries the DB for
		// the ID.
		log.Printf(
			"Wrong ID returned for album `%s` (%s) by .LastInsertId(). "+
				"Returned: %d, actual: %d.",
			album,
			fsPath,
			lastInsertID,
			newID,
		)
	}

	return newID, nil
}

// GetAlbumFSPathByName returns all the file paths which contain versions of an album.
func (lib *LocalLibrary) GetAlbumFSPathByName(albumName string) ([]string, error) {
	var paths []string

	work := func(db *sql.DB) error {
		row, err := db.Query(`
			SELECT
				fs_path
			FROM
				albums
			WHERE
				name = ?
		`, albumName)
		if err != nil {
			return err
		}

		defer row.Close()

		var albumPath string
		for row.Next() {
			if err := row.Scan(&albumPath); err != nil {
				return err
			}
			paths = append(paths, albumPath)
		}

		return nil
	}

	err := lib.executeDBJobAndWait(work)
	if err != nil {
		return paths, err
	}

	if len(paths) < 1 {
		return nil, ErrAlbumNotFound
	}

	return paths, nil
}

// GetAlbumFSPathByID returns the album path by its ID
func (lib *LocalLibrary) GetAlbumFSPathByID(albumID int64) (string, error) {
	var path string

	work := func(db *sql.DB) error {
		row, err := db.Query(`
			SELECT
				fs_path
			FROM
				albums
			WHERE
				id = ?
		`, albumID)
		if err != nil {
			return err
		}

		defer row.Close()

		if row.Next() {
			if err := row.Scan(&path); err != nil {
				return err
			}
			return nil
		}

		return ErrAlbumNotFound
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		return "", err
	}

	return path, nil
}

// GetTrackID returns the id for this track. When missing or on error returns that error.
func (lib *LocalLibrary) GetTrackID(
	title string,
	artistID int64,
	albumID int64,
) (int64, error) {
	var newID int64

	work := func(db *sql.DB) error {
		smt, err := db.Prepare(`
			SELECT
				id
			FROM
				tracks
			WHERE
				name = ? AND
				artist_id = ? AND
				album_id = ?
		`)
		if err != nil {
			return err
		}

		defer smt.Close()

		var id int64
		err = smt.QueryRow(title, artistID, albumID).Scan(&id)
		if err != nil {
			return err
		}

		newID = id
		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		return 0, err
	}

	return newID, nil
}

// Sets a new ID for this track if it is new to the library. If not, returns
// its current id. Tracks with the same name but by different artists and/or album
// need to have separate IDs hence the artistID and albumID parameters.
// Additionally trackNumber and file system path (fsPath) are required. They are
// used when retrieving this particular song for playing.
//
// In case the track with this file system path already exists in the library it
// is updated with new values for title, number, artist ID and album ID.
func (lib *LocalLibrary) setTrackID(title, fsPath string,
	trackNumber, artistID, albumID, duration int64) (int64, error) {

	if len(title) < 1 {
		title = filepath.Base(fsPath)
	}

	var lastInsertID int64
	work := func(db *sql.DB) error {
		stmt, err := db.Prepare(`
			INSERT INTO
				tracks (name, album_id, artist_id, fs_path, number, duration)
			VALUES
				($1, $2, $3, $4, $5, $6)
			ON CONFLICT (fs_path) DO
			UPDATE SET
				name = $1,
				album_id = $2,
				artist_id = $3,
				number = $5,
				duration = $6
		`)
		if err != nil {
			return err
		}

		defer stmt.Close()

		res, err := stmt.Exec(title, albumID, artistID, fsPath, trackNumber, duration)
		if err != nil {
			return err
		}

		lastInsertID, _ = res.LastInsertId()
		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		return 0, err
	}

	// Getting the track by its fs_path.
	var trackID int64
	work = func(db *sql.DB) error {
		smt, err := db.Prepare(`
			SELECT
				id
			FROM
				tracks
			WHERE
				fs_path = ?
		`)
		if err != nil {
			return err
		}

		defer smt.Close()

		var id int64
		err = smt.QueryRow(fsPath).Scan(&id)
		if err != nil {
			return err
		}

		trackID = id
		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		return 0, err
	}

	log.Printf("Inserted id: %d, name: %s, album ID: %d, artist ID: %d, "+
		"number: %d, dur: %d, fs_path: %s\n", trackID, title, albumID, artistID,
		trackNumber, duration, fsPath)

	if !lib.runningRescan && lastInsertID != trackID {
		// In case this log is never seen for a long time it would mean that
		// the LastInsertId() bug has been fixed and it is probably safe to
		// remove the second SQL request which explicitly queries the DB for
		// the ID.
		log.Printf(
			"Wrong ID returned for track `%s` by .LastInsertId(). "+
				"Returned: %d, actual: %d.",
			fsPath,
			lastInsertID,
			trackID,
		)
	}

	return trackID, nil
}

// Initialize should be run once every time a library is created. It checks for the
// sqlite database file and creates one if it is absent. If a file is found
// it does nothing.
func (lib *LocalLibrary) Initialize() error {
	if lib.db == nil {
		return errors.New("library is not opened, call its Open method first")
	}

	// This database is already created and populated. We could just apply the
	// migrations without executing the initial schema.
	if st, err := fs.Stat(lib.fs, lib.database); err == nil && st.Size() > 0 {
		return lib.applyMigrations()
	}

	sqlSchema, err := lib.readSchema()

	if err != nil {
		return err
	}

	queries := strings.Split(sqlSchema, ";")

	for _, query := range queries {
		query = strings.TrimSpace(query)

		if len(query) < 1 {
			continue
		}

		_, err = lib.db.Exec(query)

		if err != nil {
			return err
		}
	}

	return lib.applyMigrations()
}

// Returns the SQL schema for the library. It is stored in the project root directory
// under sqls/library_schema.sql
func (lib *LocalLibrary) readSchema() (string, error) {
	out, err := lib.sqlFilesFS.Open(sqlSchemaFile)
	if err != nil {
		return "", fmt.Errorf(
			"error opening schema file: %s",
			err,
		)
	}
	defer out.Close()

	schema, err := io.ReadAll(out)
	if err != nil {
		return "", fmt.Errorf(
			"error reading schema file `%s`: %s",
			sqlSchemaFile,
			err,
		)
	}

	if len(schema) < 1 {
		return "", fmt.Errorf("SQL schema was empty")
	}

	return string(schema), nil
}

// Truncate closes the library and removes its database file leaving no traces at all.
func (lib *LocalLibrary) Truncate() error {
	lib.Close()

	// The database is in-memory. There is no file which must be truncated.
	if lib.database == SQLiteMemoryFile {
		return nil
	}

	// The local library is not working with the actual file system. This is probably
	// a mock file system for tests. So skip removing the database.
	if _, ok := lib.fs.(*osFS); !ok {
		return nil
	}

	return os.Remove(lib.database)
}

// SetArtFinder bind a particular art.Finder to this library.
func (lib *LocalLibrary) SetArtFinder(caf art.Finder) {
	lib.artFinder = caf
}

// SetScaler bind a particular image scaler to this loca library.
func (lib *LocalLibrary) SetScaler(scl scaler.Scaler) {
	lib.imageScaler = scl
}

func (lib *LocalLibrary) scaleImage(
	ctx context.Context,
	img io.ReadCloser,
	toSize ImageSize,
) (io.ReadCloser, error) {
	if lib.imageScaler == nil {
		return nil, fmt.Errorf("no image scaler set for the local library")
	}
	if toSize != SmallImage {
		return nil, fmt.Errorf("scaling is supported only for small images atm")
	}

	res, err := lib.imageScaler.Scale(ctx, img, thumbnailWidth)
	if err != nil {
		return nil, fmt.Errorf("scaling failed: %w", err)
	}

	return newBytesReadCloser(res), nil
}

// NewLocalLibrary returns a new LocalLibrary which will use for database the file
// specified by databasePath. Also creates the database connection so you does not
// need to worry about that. It accepts the parent's context and create its own
// child context.
func NewLocalLibrary(
	ctx context.Context,
	databasePath string,
	sqlFilesFS fs.FS,
) (*LocalLibrary, error) {
	lib := new(LocalLibrary)
	lib.database = databasePath
	lib.sqlFilesFS = sqlFilesFS
	lib.fs = &osFS{}

	libContext, cancelFunc := context.WithCancel(ctx)

	lib.ctx = libContext
	lib.ctxCancelFunc = cancelFunc

	var err error

	lib.db, err = sql.Open("sqlite3", lib.database)
	if err != nil {
		return nil, err
	}

	if _, err := lib.db.ExecContext(ctx, `PRAGMA foreign_keys = ON;`); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	lib.watchLock = &sync.RWMutex{}
	lib.artworkSem = make(chan struct{}, 10)

	lib.cleanupLock = &sync.RWMutex{}

	var wg sync.WaitGroup
	wg.Add(1)
	go lib.databaseWorker(&wg)
	wg.Wait()

	return lib, nil
}

const thumbnailWidth = 60
