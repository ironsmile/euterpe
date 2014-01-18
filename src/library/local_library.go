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
	paths    []string       // Filesystem locations which contain the library's media files
	db       *sql.DB        // Handler to the database file mentioned in database field
	scanWait sync.WaitGroup // Used in WaitScan method
}

// Closes the database connection. It is safe to call it as many times as you want.
func (lib *LocalLibrary) Close() {
	if lib.db != nil {
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

func (lib *LocalLibrary) Search(searchTerm string) []SearchResult {
	var output []SearchResult

	//!TODO: ESCAPE ESCAPE ESCAPE!!! OR INJECTIONS AHEAD!
	query := fmt.Sprintf(`
		SELECT
			t.id as track_id,
			t.name as track,
			al.name as album,
			at.name as artist,
			t.number as track_number
		FROM
			tracks as t
				LEFT JOIN albums as al ON al.id = t.album_id
				LEFT JOIN artists as at ON at.id = t.artist_id
		WHERE
			t.name LIKE "%%%s%%" OR
			al.name LIKE "%%%s%%" OR
			at.name LIKE "%%%s%%"
	`, searchTerm, searchTerm, searchTerm)

	rows, err := lib.db.Query(query)

	if err != nil {
		log.Printf("Query not successful: %s\n", err.Error())
		return output
	}

	defer rows.Close()
	for rows.Next() {
		var res SearchResult
		rows.Scan(&res.ID, &res.Title, &res.Album, &res.Artist, &res.TrackNumber)
		output = append(output, res)
	}

	return output
}

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

//!TODO: make scan also remove files which have been deleted since the previous scan
func (lib *LocalLibrary) Scan() {
	for _, path := range lib.paths {
		lib.scanWait.Add(1)
		go lib.scanPath(path)
	}
}

func (lib *LocalLibrary) WaitScan() {
	lib.scanWait.Wait()
}

// This is the goroutine which actually scans a library path.
// For now it ignores everything but ".mp3" and ".oga" files. It is so
// because jplayer cannot play anything else.
func (lib *LocalLibrary) scanPath(scannedPath string) {
	defer lib.scanWait.Done()

	walkFunc := func(path string, info os.FileInfo, err error) error {

		if err != nil {
			log.Println(err)
			return nil
		}

		if !strings.HasSuffix(path, ".mp3") && !strings.HasSuffix(path, ".oga") {
			return nil
		}

		lib.AddMedia(path)
		return nil
	}

	err := filepath.Walk(scannedPath, walkFunc)

	if err != nil {
		log.Println(err)
	}
}

func (lib *LocalLibrary) AddMedia(filename string) error {
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
		title = UNKNOWN_LABEL
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
//!TODO: make sure there are no race conditions. But I guess sqlite handles this
// and different connections receive their respective ids.
func (lib *LocalLibrary) lastInsertID() (int64, error) {
	var id int64

	err := lib.db.QueryRow("SELECT last_insert_rowid();").Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

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
