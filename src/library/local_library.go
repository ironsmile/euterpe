package library

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gobuffalo/packr"
	"github.com/howeyc/fsnotify"
	taglib "github.com/wtolson/go-taglib"

	// Blind import is the way a SQL driver is imported. This is the proposed way
	// from the golang documentation.
	_ "github.com/mattn/go-sqlite3"

	"github.com/ironsmile/httpms/ca"
	"github.com/ironsmile/httpms/src/config"
	"github.com/ironsmile/httpms/src/helpers"
)

const (
	// UnknownLabel will be used in case some media tag is missing. As a consequence
	// if there are many files with missing title, artist and album only
	// one of them will be saved in the library.
	UnknownLabel = "Unknown"

	// SQLiteMemoryFile can be used as a database path for the sqlite's Open method.
	// When using it, one owuld create a memory database which does not write
	// anything on disk. See https://www.sqlite.org/inmemorydb.html for more info
	// on the subject of in-memory databases. We are using a shared cache because
	// this causes all the different connections in the database/sql pool to be
	// connected to the same "memory file". Without this. every new connection
	// would end up creating a new memory database.
	SQLiteMemoryFile = "file::memory:?cache=shared"

	// sqlFilesPath is path to the directory which contains the `.sql` schme and
	// migrations files. It must be relative to this package.
	sqlFilesPath = "../../sqls/"

	// sqlSchemaFile is the file which contains the initial SQL Schema for the
	// media library. It must be relateive to `sqlFilesPath`.
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

	// ErrAlbumNotFound is returned when no album could be found fr partilcuar operation.
	ErrAlbumNotFound = errors.New("Album Not Found")

	// ErrArtworkNotFound is returned when no artwork can be found for particular album.
	ErrArtworkNotFound = NewArtworkError("Artwork Not Found")

	// ErrCachedArtworkNotFound is returned when the database has been queried and
	// its cache says the artwork was not found in the recent past. No need to continue
	// searching further once you receive this error.
	ErrCachedArtworkNotFound = NewArtworkError("Artwork Not Found (Cached)")

	// ErrArtworkTooBig is returned from operation when the artwork is too big for it to
	// handle.
	ErrArtworkTooBig = NewArtworkError("Artwork Is Too Big")
)

// ArtworkError represents some kind of artwork error.
type ArtworkError struct {
	Err string
}

// Error implements the error interface.
func (a *ArtworkError) Error() string {
	return a.Err
}

// NewArtworkError returns a new artwork error which will have `err` as message.
func NewArtworkError(err string) *ArtworkError {
	return &ArtworkError{Err: err}
}

func init() {
	flag.BoolVar(&LibraryFastScan, "fast-library-scan", false, "Do not honour"+
		" the configuration set in 'library_scan'. With this flag,"+
		" scanning will be done as fast as possible. This may be useful when"+
		" running the daemon for the fists time with big libraries.")
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

	coverArtFinder ca.CovertArtFinder

	sqlFiles packr.Box
}

// Close closes the database connection. It is safe to call it as many times as you want.
func (lib *LocalLibrary) Close() {
	lib.ctxCancelFunc()
	lib.db.Close()
}

// AddLibraryPath adds a library directory to the list of libraries which will be
// scanned and consequently watched.
func (lib *LocalLibrary) AddLibraryPath(path string) {
	_, err := os.Stat(path)

	if err != nil {
		log.Printf("error adding path: %s", err)
		return
	}

	lib.paths = append(lib.paths, path)
}

// Search searches in the library. Will match against the track's name, artist and album.
func (lib *LocalLibrary) Search(searchTerm string) []SearchResult {
	searchTerm = fmt.Sprintf("%%%s%%", searchTerm)

	var output []SearchResult
	done := make(chan struct{})
	defer close(done)

	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()

		rows, err := db.Query(`
			SELECT
				t.id as track_id,
				t.name as track,
				al.name as album,
				at.name as artist,
				t.number as track_number,
				t.album_id as album_id,
				t.fs_path as fs_path
			FROM
				tracks as t
					LEFT JOIN albums as al ON al.id = t.album_id
					LEFT JOIN artists as at ON at.id = t.artist_id
			WHERE
				t.name LIKE ? OR
				al.name LIKE ? OR
				at.name LIKE ?
			ORDER BY
				al.name, t.number
		`, searchTerm, searchTerm, searchTerm)
		if err != nil {
			log.Printf("Query not successful: %s\n", err.Error())
			return nil
		}

		defer rows.Close()
		for rows.Next() {
			var res SearchResult
			rows.Scan(&res.ID, &res.Title, &res.Album, &res.Artist,
				&res.TrackNumber, &res.AlbumID, &res.Format)

			res.Format = strings.TrimLeft(filepath.Ext(res.Format), ".")
			if res.Format == "" {
				res.Format = "mp3"
			}

			output = append(output, res)
		}

		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		log.Printf("Error executing search db work: %s", err)
		return output
	}

	<-done
	return output
}

// GetFilePath returns the filsystem path for a file specified by its ID.
func (lib *LocalLibrary) GetFilePath(ID int64) string {
	var filePath string
	done := make(chan struct{})
	defer close(done)

	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()

		smt, err := db.Prepare(`
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

		err = smt.QueryRow(ID).Scan(&filePath)
		if err != nil {
			log.Printf("Error file path query row: %s\n", err)
			return nil
		}

		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		log.Printf("Error executing get file path db work: %s", err)
		return filePath
	}

	<-done
	return filePath
}

// GetAlbumFiles satisfies the Library interface
func (lib *LocalLibrary) GetAlbumFiles(albumID int64) []SearchResult {
	var output []SearchResult
	done := make(chan struct{})
	defer close(done)

	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()

		rows, err := db.Query(`
			SELECT
				t.id as track_id,
				t.name as track,
				al.name as album,
				at.name as artist,
				t.number as track_number,
				t.album_id as album_id
			FROM
				tracks as t
					LEFT JOIN albums as al ON al.id = t.album_id
					LEFT JOIN artists as at ON at.id = t.artist_id
			WHERE
				t.album_id = ?
			ORDER BY
				al.name, t.number
		`, albumID)
		if err != nil {
			log.Printf("Query not successful: %s\n", err.Error())
			return nil
		}

		defer rows.Close()
		for rows.Next() {
			var res SearchResult
			rows.Scan(&res.ID, &res.Title, &res.Album, &res.Artist,
				&res.TrackNumber, &res.AlbumID)
			output = append(output, res)
		}

		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		log.Printf("Error executing get album files db work: %s", err)
		return output
	}

	<-done
	return output
}

// Removes the file from the library. That means finding it in the database and
// removing it from there.
func (lib *LocalLibrary) removeFile(filePath string) {

	fullPath, err := filepath.Abs(filePath)

	if err != nil {
		log.Printf("Error removing %s: %s\n", filePath, err.Error())
		return
	}

	done := make(chan struct{})
	defer close(done)

	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()

		_, err = db.Exec(`
			DELETE FROM tracks
			WHERE fs_path = ?
		`, fullPath)
		if err != nil {
			log.Printf("Error removing %s: %s\n", fullPath, err.Error())
		}

		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		log.Printf("Error executing remove file db work: %s", err)
		return
	}

	<-done
}

// Removes files which belong in this directory from the library.
func (lib *LocalLibrary) removeDirectory(dirPath string) {

	// Adding slash at the end to make sure we are always removing directories
	deleteMatch := fmt.Sprintf("%s/%%", strings.TrimRight(dirPath, "/"))

	done := make(chan struct{})
	defer close(done)

	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()

		_, err := db.Exec(`
			DELETE FROM tracks
			WHERE fs_path LIKE ?
		`, deleteMatch)
		if err != nil {
			log.Printf("Error removing %s: %s\n", dirPath, err.Error())
		}

		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		log.Printf("Error executing remove dir db work: %s", err)
		return
	}

	<-done
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
	}

	for _, format := range supportedFormats {
		if !strings.HasSuffix(path, format) {
			continue
		}
		return true
	}
	return false
}

// AddMedia adds a file specified by its filesystem name to the library. Will create the
// needed Artist, Album if neccessery.
func (lib *LocalLibrary) AddMedia(filename string) error {
	if lib.MediaExistsInLibrary(filename) {
		return nil
	}

	_, err := os.Stat(filename)

	if err != nil {
		return err
	}

	file, err := taglib.Read(filename)

	if err != nil {
		return fmt.Errorf("Taglib error for %s: %s", filename, err.Error())
	}

	defer file.Close()

	// log.Printf("New Song:\nArtist: %s\nAlbum: %s\nTitle: %s\nTrack: %d\n",
	// 	file.Artist(), file.Album(), file.Title(), int(file.Track()))

	return lib.insertMediaIntoDatabase(file, filename)
}

// insertMediaIntoDatabase accepts an already parsed media info object, its path.
// The method inserts this media into the library database.
func (lib *LocalLibrary) insertMediaIntoDatabase(file MediaFile, filePath string) error {
	artistID, err := lib.setArtistID(file.Artist())

	if err != nil {
		return err
	}

	fileDir := filepath.Dir(filePath)

	albumID, err := lib.setAlbumID(file.Album(), fileDir)

	if err != nil {
		return err
	}

	trackNumber := int64(file.Track())

	if trackNumber == 0 {
		trackNumber = helpers.GuessTrackNumber(filePath)
	}

	_, err = lib.setTrackID(file.Title(), filePath, trackNumber, artistID, albumID)

	return err
}

// MediaExistsInLibrary checks if the media file with file system path "filename" has
// been added to the library already.
func (lib *LocalLibrary) MediaExistsInLibrary(filename string) bool {
	res := make(chan bool)
	defer close(res)

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
			res <- false
			return fmt.Errorf("could not prepare sql statement: %s", err)
		}
		defer smt.Close()

		var count int
		err = smt.QueryRow(filename).Scan(&count)

		if err != nil {
			res <- false
			return fmt.Errorf("error checking whether media exists already: %s", err)
		}

		res <- (count >= 1)
		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		log.Printf("Error on executing db job: %s", err)
		return false
	}

	return <-res
}

// GetArtistID returns the id for this artist. When missing or on error
// returns that error.
func (lib *LocalLibrary) GetArtistID(artist string) (int64, error) {
	var (
		artistID int64
		outErr   error
	)
	done := make(chan struct{})
	defer close(done)

	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()

		smt, err := db.Prepare(`
			SELECT
				id
			FROM
				artists
			WHERE
				name = ?
		`)
		if err != nil {
			outErr = err
			return nil
		}

		defer smt.Close()

		var id int64
		err = smt.QueryRow(artist).Scan(&id)

		if err != nil {
			outErr = err
			return nil
		}

		artistID = id
		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		return 0, err
	}

	<-done
	return artistID, outErr
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

	var (
		newID  int64
		outErr error
	)
	done := make(chan struct{})
	defer close(done)
	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()

		stmt, err := db.Prepare(`
				INSERT INTO
					artists (name)
				VALUES
					(?)
		`)
		if err != nil {
			outErr = err
			return nil
		}

		defer stmt.Close()

		_, err = stmt.Exec(artist)
		if err != nil {
			outErr = err
			return nil
		}

		newID, outErr = lastInsertID(db)
		log.Printf("Inserted artist id: %d, name: %s\n", newID, artist)
		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		return 0, err
	}

	<-done
	return newID, outErr
}

// GetAlbumID returns the id for this album. When missing or on error
// returns that error.
func (lib *LocalLibrary) GetAlbumID(album string, fsPath string) (int64, error) {
	var (
		albumID int64
		outErr  error
	)
	done := make(chan struct{})
	defer close(done)

	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()

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
			outErr = err
			return nil
		}

		defer smt.Close()

		var id int64
		err = smt.QueryRow(album, fsPath).Scan(&id)
		if err != nil {
			outErr = err
			return nil
		}

		albumID = id
		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		return 0, err
	}

	<-done
	return albumID, outErr
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

	var (
		newID  int64
		outErr error
	)
	done := make(chan struct{})
	defer close(done)

	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()

		stmt, err := db.Prepare(`
				INSERT INTO
					albums (name, fs_path)
				VALUES
					(?, ?)
		`)
		if err != nil {
			outErr = err
			return nil
		}

		defer stmt.Close()

		_, err = stmt.Exec(album, fsPath)
		if err != nil {
			outErr = err
			return nil
		}

		newID, outErr = lastInsertID(db)
		log.Printf("Inserted album id: %d, name: %s, path: %s\n", newID, album, fsPath)

		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		return 0, err
	}

	<-done
	return newID, outErr
}

// GetAlbumFSPathByName returns all the file paths which contain versions of an album.
func (lib *LocalLibrary) GetAlbumFSPathByName(albumName string) ([]string, error) {
	var (
		paths  []string
		outErr error
	)
	done := make(chan struct{})
	defer close(done)

	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()

		row, err := db.Query(`
			SELECT
				fs_path
			FROM
				albums
			WHERE
				name = ?
		`, albumName)
		if err != nil {
			outErr = err
			return nil
		}

		defer row.Close()

		var albumPath string
		for row.Next() {
			if err := row.Scan(&albumPath); err != nil {
				outErr = err
				return nil
			}
			paths = append(paths, albumPath)
		}

		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		return paths, err
	}

	<-done
	if outErr == nil && len(paths) < 1 {
		return nil, ErrAlbumNotFound
	}
	return paths, outErr
}

// GetAlbumFSPathByID returns the album path by its ID
func (lib *LocalLibrary) GetAlbumFSPathByID(albumID int64) (string, error) {
	var (
		path   string
		outErr error
	)
	done := make(chan struct{})
	defer close(done)

	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()

		row, err := db.Query(`
			SELECT
				fs_path
			FROM
				albums
			WHERE
				id = ?
		`, albumID)
		if err != nil {
			outErr = err
			return nil
		}

		defer row.Close()

		for row.Next() {
			if err := row.Scan(&path); err != nil {
				outErr = err
				return nil
			}
			return nil
		}

		outErr = ErrAlbumNotFound
		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		return "", err
	}

	<-done
	return path, outErr
}

// GetTrackID returns the id for this track. When missing or on error returns that error.
func (lib *LocalLibrary) GetTrackID(title string,
	artistID, albumID int64) (int64, error) {

	var (
		newID  int64
		outErr error
	)
	done := make(chan struct{})
	defer close(done)

	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()

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
			outErr = err
			return nil
		}

		defer smt.Close()

		var id int64
		err = smt.QueryRow(title, artistID, albumID).Scan(&id)
		if err != nil {
			outErr = err
			return nil
		}

		newID = id
		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		return 0, err
	}

	<-done
	return newID, outErr
}

// Sets a new ID for this track if it is new to the library. If not, returns
// its current id. Tracks with the same name but by different artists and/or album
// need to have separate IDs hence the artistID and albumID parameters.
// Additionally trackNumber and filesystem path (fsPath) are required. They are
// used when retreiving this particular song for playing.
func (lib *LocalLibrary) setTrackID(title, fsPath string,
	trackNumber, artistID, albumID int64) (int64, error) {

	if len(title) < 1 {
		title = filepath.Base(fsPath)
	}

	id, err := lib.GetTrackID(title, artistID, albumID)
	if err == nil {
		return id, nil
	}

	var (
		newID  int64
		outErr error
	)
	done := make(chan struct{})
	defer close(done)

	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()

		stmt, err := db.Prepare(`
			INSERT INTO
				tracks (name, album_id, artist_id, fs_path, number)
			VALUES
				(?, ?, ?, ?, ?)
		`)
		if err != nil {
			outErr = err
			return nil
		}

		defer stmt.Close()

		_, err = stmt.Exec(title, albumID, artistID, fsPath, trackNumber)
		if err != nil {
			outErr = err
			return nil
		}

		newID, outErr = lastInsertID(db)
		log.Printf("Inserted id: %d, name: %s, album ID: %d, artist ID: %d, number: %d, fs_path: %s\n",
			newID, title, albumID, artistID, trackNumber, fsPath)

		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		return 0, err
	}

	<-done
	return newID, outErr
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
	if st, err := os.Stat(lib.database); err == nil && st.Size() > 0 {
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
	out, err := lib.sqlFiles.MustString(sqlSchemaFile)
	if err != nil {
		return "", fmt.Errorf(
			"error reading schema file `%s`: %s",
			sqlSchemaFile,
			err,
		)
	}

	if len(out) < 1 {
		return "", fmt.Errorf("SQL schema was empty")
	}

	return out, nil
}

// Truncate Closes the library and removes its database file leaving no traces at all.
func (lib *LocalLibrary) Truncate() error {
	lib.Close()

	// The database is in-memory. There is no file which must be truncated.
	if lib.database == SQLiteMemoryFile {
		return nil
	}

	return os.Remove(lib.database)
}

// SetCoverArtFinder bind a particular ca.CoverArtFinder to this library.
func (lib *LocalLibrary) SetCoverArtFinder(caf ca.CovertArtFinder) {
	lib.coverArtFinder = caf
}

// NewLocalLibrary returns a new LocalLibrary which will use for database the file
// specified by databasePath. Also creates the database connection so you does not
// need to worry about that. It accepts the parent's context and create its own
// child context.
func NewLocalLibrary(ctx context.Context, databasePath string) (*LocalLibrary, error) {
	lib := new(LocalLibrary)
	lib.database = databasePath
	lib.sqlFiles = packr.NewBox(sqlFilesPath)

	libContext, cancelFunc := context.WithCancel(ctx)

	lib.ctx = libContext
	lib.ctxCancelFunc = cancelFunc

	var err error

	lib.db, err = sql.Open("sqlite3", lib.database)

	if err != nil {
		return nil, err
	}

	lib.watchLock = &sync.RWMutex{}
	lib.artworkSem = make(chan struct{}, 10)

	var wg sync.WaitGroup
	wg.Add(1)
	go lib.databaseWorker(&wg)
	wg.Wait()

	return lib, nil
}
