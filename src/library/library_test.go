//!TODO: Use a random temp file. Someone may be using /tmp/test.db for something
// already
package library

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/ironsmile/httpms/src/helpers"
)

func getLibrary(t *testing.T, dbFile string) *LocalLibrary {

	lib, err := NewLocalLibrary(dbFile)

	if err != nil {
		t.Fatalf(err.Error())
	}

	err = lib.Initialize()

	if err != nil {
		t.Fatalf("Initializing library: %s", err.Error())
	}

	projRoot, err := helpers.ProjectRoot()

	if err != nil {
		t.Fatalf("Was not able to find test_files directory.", err.Error())
	}

	testLibraryPath := filepath.Join(projRoot, "test_files", "library")

	_ = lib.AddMedia(filepath.Join(testLibraryPath, "test_file_two.mp3"))
	_ = lib.AddMedia(filepath.Join(testLibraryPath, "folder_one", "third_file.mp3"))

	return lib
}

func TestInitialize(t *testing.T) {
	lib, err := NewLocalLibrary("/tmp/test-init.db")

	if err != nil {
		t.Fatalf(err.Error())
	}

	defer lib.Close()

	err = lib.Initialize()

	if err != nil {
		t.Fatalf(err.Error())
	}

	defer lib.Truncate()
	defer func() {
		os.Remove("/tmp/test-init.db")
	}()

	st, err := os.Stat("/tmp/test-init.db")

	if err != nil {
		t.Fatalf(err.Error())
	}

	if st.Size() < 1 {
		t.Errorf("Library database was 0 bytes in size")
	}

	db, err := sql.Open("sqlite3", "/tmp/test-init.db")
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer db.Close()

	var tables = []string{"albums", "tracks", "artists"}

	for _, table := range tables {
		row, err := db.Query(fmt.Sprintf("SELECT count(id) as cnt FROM %s", table))
		if err != nil {
			t.Fatalf(err.Error())
		}
		defer row.Close()
	}
}

func TestTruncate(t *testing.T) {
	lib, err := NewLocalLibrary("/tmp/test-truncate.db")

	if err != nil {
		t.Fatalf(err.Error())
	}

	err = lib.Initialize()

	if err != nil {
		t.Fatalf(err.Error())
	}

	lib.Truncate()

	_, err = os.Stat("/tmp/test-truncate.db")

	if err == nil {
		os.Remove("/tmp/test-truncate.db")
		t.Errorf("Expected database file to be missing but it is still there")
	}
}

func TestSearch(t *testing.T) {
	lib := getLibrary(t, "/tmp/test-search.db")
	defer lib.Truncate()

	found := lib.Search("Buggy")

	if len(found) != 1 {
		t.Fatalf("Expected 1 result but got %d", len(found))
	}

	expected := SearchResult{
		Artist:      "Buggy Bugoff",
		Album:       "Return Of The Bugs",
		Title:       "Payback",
		TrackNumber: 1,
	}

	if found[0].Artist != expected.Artist {
		t.Errorf("Expected Artist `%s` but found `%s`",
			expected.Artist, found[0].Artist)
	}

	if found[0].Title != expected.Title {
		t.Errorf("Expected Title `%s` but found `%s`",
			expected.Title, found[0].Title)
	}

	if found[0].Album != expected.Album {
		t.Errorf("Expected Album `%s` but found `%s`",
			expected.Album, found[0].Album)
	}

	if found[0].TrackNumber != expected.TrackNumber {
		t.Errorf("Expected TrackNumber `%d` but found `%d`",
			expected.TrackNumber, found[0].TrackNumber)
	}

}

func TestDirectoryScan(t *testing.T) {

}

func TestAddigNewFiles(t *testing.T) {
	projRoot, err := helpers.ProjectRoot()

	if err != nil {
		t.Fatalf(err.Error())
	}

	library := getLibrary(t, "/tmp/test-new-files.db")
	defer library.Truncate()

	db, err := sql.Open("sqlite3", "/tmp/test-new-files.db")
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer db.Close()

	tracksCount := func() int {
		rows, err := db.Query("SELECT count(id) as cnt FROM tracks")
		if err != nil {
			t.Fatalf(err.Error())
			return 0
		}
		defer rows.Close()

		var count int

		for rows.Next() {
			rows.Scan(&count)
		}

		return count
	}

	tracks := tracksCount()

	if tracks != 2 {
		t.Errorf("Expected to find 2 tracks but found %d", tracks)
	}

	testLibraryPath := filepath.Join(projRoot, "test_files", "library")
	absentFile := filepath.Join(testLibraryPath, "not_there")

	err = library.AddMedia(absentFile)

	if err == nil {
		t.Fatalf("Expected a 'not found' error but got no error at all")
	}

	realFile := filepath.Join(testLibraryPath, "test_file_one.mp3")

	err = library.AddMedia(realFile)

	if err != nil {
		t.Errorf(err.Error())
	}

	tracks = tracksCount()

	if tracks != 3 {
		t.Errorf("Expected to find 3 tracks but found %d", tracks)
	}

}

func TestPreAddedFiles(t *testing.T) {
	library := getLibrary(t, "/tmp/test-preadded-files.db")
	defer library.Truncate()

	_, err := library.GetArtistID("doycho")

	if err == nil {
		t.Errorf("Was not expecting to find artist doycho")
	}

	artistID, err := library.GetArtistID("Artist Testoff")

	if err != nil {
		t.Fatalf("Was not able to find Artist Testoff: %s", err.Error())
	}

	_, err = library.GetAlbumID("Album Of Not Being There", artistID)

	if err == nil {
		t.Errorf("Was not expecting to find Album Of Not Being There but found one")
	}

	albumID, err := library.GetAlbumID("Album Of Tests", artistID)

	if err != nil {
		t.Fatalf("Was not able to find Album Of Tests: %d", err.Error())
	}

	_, err = library.GetTrackID("404 Not Found", artistID, albumID)

	if err == nil {
		t.Errorf("Was not expecting to find 404 Not Found track but it was there")
	}

	_, err = library.GetTrackID("Another One", artistID, albumID)

	if err != nil {
		t.Fatalf("Was not able to find track Another One: %s", err.Error())
	}
}

func TestGettingAFile(t *testing.T) {
	library := getLibrary(t, "/tmp/test-getting-a-file.db")
	defer library.Truncate()

	artistID, _ := library.GetArtistID("Artist Testoff")
	albumID, _ := library.GetAlbumID("Album Of Tests", artistID)
	trackID, err := library.GetTrackID("Another One", artistID, albumID)

	if err != nil {
		t.Fatalf("File not found: %S", err.Error())
	}

	filePath := library.GetFilePath(trackID)

	suffix := "/test_files/library/test_file_two.mp3"

	if !strings.HasSuffix(filePath, filepath.FromSlash(suffix)) {
		t.Errorf("Returned track file Another One did not have the proper file path")
	}
}

func TestScaning(t *testing.T) {
	//!TODO
}
