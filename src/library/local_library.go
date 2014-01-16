package library

import (
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	taglib "github.com/landr0id/go-taglib"
	_ "github.com/mattn/go-sqlite3"

	"github.com/ironsmile/httpms/src/helpers"
)

// Implements the Library interface. Will represent files found on the local storage
type LocalLibrary struct {
	database string   // The location the library's database
	paths    []string // Filesystem locations which contain the library's media files
	db       *sql.DB
}

func (lib LocalLibrary) Close() {
	if lib.db != nil {
		lib.db.Close()
		lib.db = nil
	}
}

func (lib LocalLibrary) AddLibraryPath(path string) {
	lib.paths = append(lib.paths, path)
}

func (lib LocalLibrary) Search(searchTerm string) []SearchResult {
	return []SearchResult{}
}

func (lib LocalLibrary) GetFilePath(ID int64) string {
	return ""
}

func (lib LocalLibrary) Scan() error {
	return nil
}

func (lib LocalLibrary) AddMedia(filename string) error {
	_, err := os.Stat(filename)

	if err != nil {
		return err
	}

	file, err := taglib.Read(filename)

	if err != nil {
		return err
	}

	defer file.Close()

	artistID, err := lib.getArtistID(file.Artist())

	if err != nil {
		return err
	}

	albumID, err := lib.getAlbumID(file.Album(), artistID)

	if err != nil {
		return err
	}

	err = lib.insertTrack(file.Title(), filename, artistID, albumID)

	if err != nil {
		return err
	}

	return nil
}

func (lib LocalLibrary) getArtistID(artist string) (int64, error) {
	return 4, nil
}

func (lib LocalLibrary) getAlbumID(album string, artistID int64) (int64, error) {
	return 4, nil
}

func (lib LocalLibrary) insertTrack(title, fs_path string,
	artistID, albumID int64) error {

	stmt, err := lib.db.Prepare(`
		INSERT INTO
			tracks (name, album_id, artist_id, fs_path)
		VALUES
			(?, ?, ?, ?)
	`)

	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.Exec(title, albumID, artistID, fs_path)

	if err != nil {
		return err
	}

	return nil
}

func (lib LocalLibrary) Initialize() error {
	_, err := os.Stat(lib.database)

	if err == nil {
		return nil
	}

	sqlSchema, err := lib.readSchema()

	if err != nil {
		return err
	}

	if lib.db == nil {
		return errors.New("Library is not opened")
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

func (lib LocalLibrary) readSchema() (string, error) {
	projRoot, err := helpers.ProjectRoot()

	if err != nil {
		return "", err
	}
	sqlSchemaPath := filepath.Join(projRoot, "sqls", "library_schema.sql")

	fp, err := os.Open(sqlSchemaPath)

	if err != nil {
		return "", err
	}
	defer fp.Close()

	var out string
	buf := make([]byte, 10*1024)

	for {
		_, err := fp.Read(buf)
		if err != nil && err != io.EOF {
			return "", err
		}
		if err == io.EOF {
			break
		}
		out = out + string(buf)
	}

	out = strings.Trim(out, "\x00\n\t\r\f\v ")

	if len(out) < 1 {
		return "", errors.New("SQL schema was empty")
	}

	return out, nil
}

func (lib LocalLibrary) Truncate() error {
	lib.Close()
	return os.Remove(lib.database)
}

func NewLocalLibrary(databasePath string) (*LocalLibrary, error) {
	if len(databasePath) < 1 {
		return nil, errors.New("No database supplied to open")
	}

	lib := LocalLibrary{database: databasePath}

	var err error

	lib.db, err = sql.Open("sqlite3", lib.database)

	if err != nil {
		return nil, err
	}

	return &lib, nil
}
