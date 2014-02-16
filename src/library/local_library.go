package library

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	taglib "github.com/landr0id/go-taglib"
	_ "github.com/mattn/go-sqlite3"

	"github.com/ironsmile/httpms/src/helpers"
)

// Will be used in case some media tag is missing. As a consequence
// if there are many files with missing title, artist and album only
// one of them will be saved in the library.
const UNKNOWN_LABEL = "Unknown"

// Implements the Library interface. Will represent files found on the local storage
type LocalLibrary struct {
	database string         // The location of the library's database
	paths    []string       // FS locations which contain the library's media files
	db       *sql.DB        // Handler to the database file mentioned in database field
	scanWait sync.WaitGroup // Used in WaitScan method
	dbWait   sync.WaitGroup
}

// Closes the database connection. It is safe to call it as many times as you want.
func (lib *LocalLibrary) Close() {
	if lib.db != nil {
		lib.WaitScan()
		lib.db.Close()
		lib.db = nil
	}
}

func (lib *LocalLibrary) AddLibraryPath(path string) {
	_, err := os.Stat(path)

	if err != nil {
		log.Println(err)
		return
	}

	lib.paths = append(lib.paths, path)
}

// Does a search in the library. Will match against the track's name, artist and album.
func (lib *LocalLibrary) Search(searchTerm string) []SearchResult {
	var output []SearchResult

	searchTerm = fmt.Sprintf("%%%s%%", searchTerm)

	rows, err := lib.db.Query(`
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
			t.name LIKE ? OR
			al.name LIKE ? OR
			at.name LIKE ?
		ORDER BY
			al.name, t.number
	`, searchTerm, searchTerm, searchTerm)

	if err != nil {
		log.Printf("Query not successful: %s\n", err.Error())
		return output
	}

	defer rows.Close()
	for rows.Next() {
		var res SearchResult
		rows.Scan(&res.ID, &res.Title, &res.Album, &res.Artist,
			&res.TrackNumber, &res.AlbumID)
		output = append(output, res)
	}

	return output
}

// Returns the filsystem path for a file specified by its ID.
func (lib *LocalLibrary) GetFilePath(ID int64) string {

	smt, err := lib.db.Prepare(`
		SELECT
			fs_path
		FROM
			tracks
		WHERE
			id = ?
	`)

	if err != nil {
		log.Println(err)
		return ""
	}

	defer smt.Close()

	var filePath string
	err = smt.QueryRow(ID).Scan(&filePath)

	if err != nil {
		log.Println(err)
		return ""
	}

	return filePath
}

// Satisfies the Library interface
func (lib *LocalLibrary) GetAlbumFiles(albumID int64) []SearchResult {
	var output []SearchResult

	rows, err := lib.db.Query(`
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
		return output
	}

	defer rows.Close()
	for rows.Next() {
		var res SearchResult
		rows.Scan(&res.ID, &res.Title, &res.Album, &res.Artist,
			&res.TrackNumber, &res.AlbumID)
		output = append(output, res)
	}

	return output
}

//!TODO: make scan also remove files which have been deleted since the previous scan
// Scans all of the folders in paths for media files. New files will be added to the
// database.
func (lib *LocalLibrary) Scan() {
	// Make sure there are no other scans working at the moment
	lib.WaitScan()

	start := time.Now()
	mediaChan := make(chan string, 100)

	lib.dbWait.Add(1)
	go lib.databaseWriter(mediaChan)

	for _, path := range lib.paths {
		lib.scanWait.Add(1)
		go lib.scanPath(path, mediaChan)
	}

	lib.dbWait.Add(1)
	go func() {
		defer func() {
			log.Printf("Walking took %s", time.Since(start))
			lib.dbWait.Done()
		}()
		lib.scanWait.Wait()
		close(mediaChan)
	}()

	go func() {
		lib.dbWait.Wait()
		log.Printf("Scaning took %s", time.Since(start))
	}()
}

// Reads from the media channel and saves into the database every file
// received.
func (lib *LocalLibrary) databaseWriter(media <-chan string) {
	defer lib.dbWait.Done()
	for filename := range media {
		lib.AddMedia(filename)
	}
}

// Blocks the current goroutine until the scan has been finished
func (lib *LocalLibrary) WaitScan() {
	lib.dbWait.Wait()
}

// This is the goroutine which actually scans a library path.
// For now it ignores everything but ".mp3" and ".oga" files. It is so
// because jplayer cannot play anything else. Sends every suitable
// file into the media channel
func (lib *LocalLibrary) scanPath(scannedPath string, media chan<- string) {
	start := time.Now()

	defer func() {
		log.Printf("Walking %s took %s", scannedPath, time.Since(start))
		lib.scanWait.Done()
	}()

	supportedFormats := []string{
		".mp3",
		".ogg",
		".oga",
		".wav",
		".fla",
		".flac",
		".m4a",
	}

	walkFunc := func(path string, info os.FileInfo, err error) error {

		if err != nil {
			log.Println(err)
			return nil
		}

		for _, format := range supportedFormats {
			if !strings.HasSuffix(path, format) {
				continue
			}
			media <- path
			break
		}

		return nil
	}

	err := filepath.Walk(scannedPath, walkFunc)

	if err != nil {
		log.Println(err)
	}
}

// Adds a file specified by its filesystem name to the library. Will create the
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
		return err
	}

	defer file.Close()

	artistID, err := lib.setArtistID(file.Artist())

	if err != nil {
		return err
	}

	albumID, err := lib.setAlbumID(file.Album(), artistID)

	if err != nil {
		return err
	}

	_, err = lib.setTrackID(file.Title(), filename, int64(file.Track()),
		artistID, albumID)

	if err != nil {
		return err
	}

	return nil
}

// Checks if the media file with file system path "filename" has been added to the
// library already.
func (lib *LocalLibrary) MediaExistsInLibrary(filename string) bool {
	smt, err := lib.db.Prepare(`
		SELECT
			count(id)
		FROM
			tracks
		WHERE
			fs_path = ?
	`)

	if err != nil {
		log.Println(err)
		return false
	}
	defer smt.Close()

	var count int
	err = smt.QueryRow(filename).Scan(&count)

	if err != nil {
		log.Println(err)
		return false
	}

	return count >= 1
}

// Returns the id for this artist. When missing or on error returns that error.
func (lib *LocalLibrary) GetArtistID(artist string) (int64, error) {
	smt, err := lib.db.Prepare(`
		SELECT
			id
		FROM
			artists
		WHERE
			name = ?
	`)

	if err != nil {
		return 0, err
	}

	defer smt.Close()

	var id int64
	err = smt.QueryRow(artist).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

// Sets a new ID for this artist if it is new to the library. If not, returns
// its current id.
func (lib *LocalLibrary) setArtistID(artist string) (int64, error) {
	if len(artist) < 1 {
		artist = UNKNOWN_LABEL
	}

	id, err := lib.GetArtistID(artist)

	if err == nil {
		return id, nil
	}

	stmt, err := lib.db.Prepare(`
			INSERT INTO
				artists (name)
			VALUES
				(?)
	`)

	if err != nil {
		return 0, err
	}

	defer stmt.Close()

	_, err = stmt.Exec(artist)

	if err != nil {
		return 0, err
	}

	return lib.lastInsertID()
}

// Returns the id for this artist's album. When missing or on error returns that error.
func (lib *LocalLibrary) GetAlbumID(album string, artistID int64) (int64, error) {
	smt, err := lib.db.Prepare(`
		SELECT
			id
		FROM
			albums
		WHERE
			name = ? AND
			artist_id = ?
	`)

	if err != nil {
		return 0, err
	}

	defer smt.Close()

	var id int64
	err = smt.QueryRow(album, artistID).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

// Sets a new ID for this album if it is new to the library. If not, returns
// its current id. Albums with the same name but by different artists need to have
// separate IDs hence the artistID parameter.
func (lib *LocalLibrary) setAlbumID(album string, artistID int64) (int64, error) {
	if len(album) < 1 {
		album = UNKNOWN_LABEL
	}

	id, err := lib.GetAlbumID(album, artistID)

	if err == nil {
		return id, nil
	}

	stmt, err := lib.db.Prepare(`
			INSERT INTO
				albums (name, artist_id)
			VALUES
				(?, ?)
	`)

	if err != nil {
		return 0, err
	}

	defer stmt.Close()

	_, err = stmt.Exec(album, artistID)

	if err != nil {
		return 0, err
	}

	return lib.lastInsertID()
}

// Returns the id for this track. When missing or on error returns that error.
func (lib *LocalLibrary) GetTrackID(title string,
	artistID, albumID int64) (int64, error) {
	smt, err := lib.db.Prepare(`
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
		return 0, err
	}

	defer smt.Close()

	var id int64
	err = smt.QueryRow(title, artistID, albumID).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

// Sets a new ID for this track if it is new to the library. If not, returns
// its current id. Tracks with the same name but by different artists and/or album
// need to have separate IDs hence the artistID and albumID parameters.
// Additionally trackNumber and filesystem path (fs_path) are required. They are
// used when retreiving this particular song for playing.
func (lib *LocalLibrary) setTrackID(title, fs_path string,
	trackNumber, artistID, albumID int64) (int64, error) {

	if len(title) < 1 {
		title = filepath.Base(fs_path)
	}

	id, err := lib.GetTrackID(title, artistID, albumID)

	if err == nil {
		return id, nil
	}

	stmt, err := lib.db.Prepare(`
		INSERT INTO
			tracks (name, album_id, artist_id, fs_path, number)
		VALUES
			(?, ?, ?, ?, ?)
	`)

	if err != nil {
		return 0, err
	}

	defer stmt.Close()

	_, err = stmt.Exec(title, albumID, artistID, fs_path, trackNumber)

	if err != nil {
		return 0, err
	}

	return lib.lastInsertID()
}

// Returns the last ID insert in the database.
func (lib *LocalLibrary) lastInsertID() (int64, error) {
	var id int64

	err := lib.db.QueryRow("SELECT last_insert_rowid();").Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

// Should be run once every time a library is created. It checks for the
// sqlite database file and creates one if it is absent. If a file is found
// it does nothing.
func (lib *LocalLibrary) Initialize() error {
	_, err := os.Stat(lib.database)

	if err == nil {
		return nil
	}

	sqlSchema, err := lib.readSchema()

	if err != nil {
		return err
	}

	if lib.db == nil {
		return errors.New("Library is not opened. Call its Open method first.")
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

	return nil
}

// Returns the SQL schema for the library. It is stored in the project root directory
// under sqls/library_schema.sql
func (lib *LocalLibrary) readSchema() (string, error) {
	projRoot, err := helpers.ProjectRoot()

	if err != nil {
		return "", err
	}
	sqlSchemaPath := filepath.Join(projRoot, "sqls", "library_schema.sql")

	outBytes, err := ioutil.ReadFile(sqlSchemaPath)

	if err != nil {
		return "", err
	}

	out := string(outBytes)

	if len(out) < 1 {
		return "", errors.New("SQL schema was empty")
	}

	return out, nil
}

func (lib *LocalLibrary) Truncate() error {
	lib.Close()
	return os.Remove(lib.database)
}

// Returns a new LocalLibrary which will use for database the file specified by
// databasePath. Also creates the database connection so you does not need to
// worry about that.
func NewLocalLibrary(databasePath string) (*LocalLibrary, error) {
	lib := new(LocalLibrary)
	lib.database = databasePath

	var err error

	lib.db, err = sql.Open("sqlite3", lib.database)

	if err != nil {
		return nil, err
	}

	return lib, nil
}
