//!TODO: Use a random temp file. Someone may be using /tmp/test.db for something
// already
package library

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/ironsmile/httpms/src/helpers"
)

func init() {
	devnull, _ := os.Create(os.DevNull)
	log.SetOutput(devnull)
}

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

func getPathedLibrary(t *testing.T, dbFile string) *LocalLibrary {
	projRoot, err := helpers.ProjectRoot()

	if err != nil {
		t.Fatalf("Was not able to find test_files directory.", err.Error())
	}

	testLibraryPath := filepath.Join(projRoot, "test_files", "library")

	lib, err := NewLocalLibrary(dbFile)

	if err != nil {
		t.Fatalf(err.Error())
	}

	err = lib.Initialize()

	if err != nil {
		t.Fatalf("Initializing library: %s", err.Error())
	}

	lib.AddLibraryPath(testLibraryPath)

	return lib
}

func getScannedLibrary(t *testing.T, dbFile string) *LocalLibrary {
	lib := getPathedLibrary(t, dbFile)

	lib.Scan()

	ch := testErrorAfter(10, "Scanning library took too long")
	lib.WaitScan()
	ch <- 42

	return lib
}

func testErrorAfter(seconds time.Duration, message string) chan int {
	ch := make(chan int)

	go func() {
		select {
		case _ = <-ch:
			close(ch)
			return
		case <-time.After(seconds * time.Second):
			close(ch)
			println(message)
			os.Exit(1)
		}
	}()

	return ch
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

	if found[0].AlbumID < 1 {
		t.Errorf("AlbumID was below 1: `%d`", found[0].AlbumID)
	}

}

func TestAddigNewFiles(t *testing.T) {
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

	projRoot, err := helpers.ProjectRoot()

	if err != nil {
		t.Fatalf(err.Error())
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

	found := library.Search("Tittled Track")

	if len(found) != 1 {
		t.Errorf("Expected to find one track but found %d", len(found))
	}

	track := found[0]

	if track.Title != "Tittled Track" {
		t.Errorf("Found track had the wrong title: %s", track.Title)
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

func TestAddingLibraryPaths(t *testing.T) {
	lib := getPathedLibrary(t, "/tmp/test-library-paths.db")
	defer lib.Truncate()

	if len(lib.paths) != 1 {
		t.Fatalf("Expected 1 library path but found %d", len(lib.paths))
	}

	notExistingPath := filepath.FromSlash("/hopefully/not/existing/path/")

	lib.AddLibraryPath(notExistingPath)
	lib.AddLibraryPath(filepath.FromSlash("/"))

	if len(lib.paths) != 2 {
		t.Fatalf("Expected 2 library path but found %d", len(lib.paths))
	}
}

func TestScaning(t *testing.T) {
	lib := getPathedLibrary(t, "/tmp/test-library-paths.db")
	defer lib.Truncate()

	lib.Scan()

	ch := testErrorAfter(10, "Scanning library took too long")
	lib.WaitScan()
	ch <- 42

	for _, track := range []string{"Another One", "Payback", "Tittled Track"} {
		found := lib.Search(track)

		if len(found) != 1 {
			t.Errorf("%s was not found after the scan", track)
		}
	}

}

func TestSQLInjections(t *testing.T) {
	lib := getScannedLibrary(t, "/tmp/test-sql-injections.db")
	defer lib.Truncate()

	found := lib.Search(`not-such-thing" OR 1=1 OR t.name="kleopatra`)

	if len(found) != 0 {
		t.Errorf("Successful sql injection in a single query")
	}

}

func TestGetAlbumFiles(t *testing.T) {
	lib := getScannedLibrary(t, "/tmp/test-album-download.db")
	defer lib.Truncate()

	artistID, _ := lib.GetArtistID("Artist Testoff")
	albumID, _ := lib.GetAlbumID("Album Of Tests", artistID)

	albumFiles := lib.GetAlbumFiles(albumID)

	if len(albumFiles) != 2 {
		t.Errorf("Expected 2 files in the album but found %d", len(albumFiles))
	}

	for _, track := range albumFiles {
		if track.Album != "Album Of Tests" {
			t.Errorf("GetAlbumFiles returned file in album `%s`", track.Album)
		}

		if track.Artist != "Artist Testoff" {
			t.Errorf("GetAlbumFiles returned file from artist `%s`", track.Artist)
		}
	}

	trackNames := []string{"Tittled Track", "Another One"}

	for _, trackName := range trackNames {
		found := false
		for _, track := range albumFiles {
			if track.Title == trackName {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Track `%s` was not among the results", trackName)
		}
	}
}

func TestRemoveFileFunction(t *testing.T) {
	lib := getScannedLibrary(t, "/tmp/test-moving-files.db")
	defer lib.Truncate()

	found := lib.Search("Another One")

	if len(found) != 1 {
		t.Fatalf(`Expected searching for 'Another One' to return one `+
			`result but they were %d`, len(found))
	}

	fs_path := lib.GetFilePath(found[0].ID)

	lib.removeFile(fs_path)

	found = lib.Search("Another One")

	if len(found) != 0 {
		t.Error(`Did not expect to find Another One but it was there.`)
	}
}

func checkAddedSong(lib *LocalLibrary, t *testing.T) {
	found := lib.Search("Added Song")

	if len(found) != 1 {
		t.Fatalf("Expected one result, got %d for Added Song", len(found))
	}

	track := found[0]

	if track.Album != "Unexpected Album" {
		t.Errorf("Wrong track album: %s", track.Album)
	}

	if track.Artist != "New Artist" {
		t.Errorf("Wrong track artist: %s", track.Artist)
	}

	if track.Title != "Added Song" {
		t.Errorf("Wrong track title: %s", track.Title)
	}

	if track.TrackNumber != 1 {
		t.Errorf("Wrong track number: %d", track.TrackNumber)
	}
}

func TestAddingNewFile(t *testing.T) {
	lib := getScannedLibrary(t, "/tmp/test-new-files.db")
	defer lib.Truncate()
	projRoot, _ := helpers.ProjectRoot()
	testFiles := filepath.Join(projRoot, "test_files")

	testMp3 := filepath.Join(testFiles, "more_mp3s", "test_file_added.mp3")
	newFile := filepath.Join(testFiles, "library", "folder_one", "test_file_added.mp3")

	if err := helpers.Copy(testMp3, newFile); err != nil {
		t.Fatalf("Copying file to library faild: %s", err.Error())
	}

	defer os.Remove(newFile)
	time.Sleep(10 * time.Millisecond)

	checkAddedSong(lib, t)
}

func TestMovingFileIntoLibrary(t *testing.T) {
	lib := getScannedLibrary(t, "/tmp/test-moving-files.db")
	defer lib.Truncate()
	projRoot, _ := helpers.ProjectRoot()
	testFiles := filepath.Join(projRoot, "test_files")

	testMp3 := filepath.Join(testFiles, "more_mp3s", "test_file_added.mp3")
	toBeMoved := filepath.Join(testFiles, "more_mp3s", "test_file_moved.mp3")
	newFile := filepath.Join(testFiles, "library", "test_file_added.mp3")

	if err := helpers.Copy(testMp3, toBeMoved); err != nil {
		t.Fatalf("Copying file to library faild: %s", err.Error())
	}

	if err := os.Rename(toBeMoved, newFile); err != nil {
		os.Remove(toBeMoved)
		t.Fatalf("Was not able to move new file into library: %s", err.Error())
	}

	defer os.Remove(newFile)
	time.Sleep(10 * time.Millisecond)

	checkAddedSong(lib, t)
}

func TestAddingNonRelatedFile(t *testing.T) {
	lib := getScannedLibrary(t, "/tmp/test-adding-nonrelated-files.db")
	defer lib.Truncate()
	projRoot, _ := helpers.ProjectRoot()
	testFiles := filepath.Join(projRoot, "test_files")
	newFile := filepath.Join(testFiles, "library", "not_related")

	testLibFiles := func() {
		results := lib.Search("")
		if len(results) != 3 {
			t.Errorf("Expected 3 files in the library but found %d", len(results))
		}
	}

	fh, err := os.Create(newFile)

	if err != nil {
		t.Fatal(err)
	}

	fh.WriteString("Some contents")
	fh.Close()

	time.Sleep(10 * time.Millisecond)
	testLibFiles()

	os.Remove(newFile)
	time.Sleep(10 * time.Millisecond)
	testLibFiles()

}

func TestRemovingFile(t *testing.T) {
	projRoot, _ := helpers.ProjectRoot()
	testFiles := filepath.Join(projRoot, "test_files")

	testMp3 := filepath.Join(testFiles, "more_mp3s", "test_file_added.mp3")
	newFile := filepath.Join(testFiles, "library", "test_file_added.mp3")

	if err := helpers.Copy(testMp3, newFile); err != nil {
		t.Fatalf("Copying file to library faild: %s", err.Error())
	}

	lib := getScannedLibrary(t, "/tmp/test-removing-files.db")
	defer lib.Truncate()

	results := lib.Search("")

	if len(results) != 4 {
		t.Errorf("Expected 4 files in the result set but found %d", len(results))
	}

	if err := os.Remove(newFile); err != nil {
		t.Error(err)
	}

	time.Sleep(10 * time.Millisecond)

	results = lib.Search("")
	if len(results) != 3 {
		t.Errorf("Expected 3 files in the result set but found %d", len(results))
	}
}

func TestAddingAndRemovingDirectory(t *testing.T) {
	lib := getScannedLibrary(t, "/tmp/test-moving-directory.db")
	defer lib.Truncate()

	projRoot, _ := helpers.ProjectRoot()
	testFiles := filepath.Join(projRoot, "test_files")

	testDir := filepath.Join(testFiles, "more_mp3s", "to_be_moved_directory")
	movedDir := filepath.Join(testFiles, "library", "to_be_moved_directory")
	srcTestMp3 := filepath.Join(testFiles, "more_mp3s", "test_file_added.mp3")
	dstTestMp3 := filepath.Join(testFiles, "more_mp3s", "to_be_moved_directory",
		"test_file_added.mp3")

	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(testDir)

	if err := helpers.Copy(srcTestMp3, dstTestMp3); err != nil {
		t.Fatal(err)
	}

	if err := os.Rename(testDir, movedDir); err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)

	checkAddedSong(lib, t)

	if err := os.RemoveAll(movedDir); err != nil {
		t.Error(err)
	}

	results := lib.Search("")

	if len(results) != 3 {
		t.Errorf("Expected 3 songs but found %d", len(results))
	}

}

func TestMovingDirectory(t *testing.T) {
	projRoot, _ := helpers.ProjectRoot()
	testFiles := filepath.Join(projRoot, "test_files")

	testDir := filepath.Join(testFiles, "more_mp3s", "to_be_moved_directory")
	movedDir := filepath.Join(testFiles, "library", "to_be_moved_directory")
	secondPlace := filepath.Join(testFiles, "library", "second_place")
	srcTestMp3 := filepath.Join(testFiles, "more_mp3s", "test_file_added.mp3")
	dstTestMp3 := filepath.Join(testFiles, "more_mp3s", "to_be_moved_directory",
		"test_file_added.mp3")

	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(testDir)

	if err := helpers.Copy(srcTestMp3, dstTestMp3); err != nil {
		t.Fatal(err)
	}

	if err := os.Rename(testDir, movedDir); err != nil {
		t.Fatal(err)
	} else {
		defer os.RemoveAll(movedDir)
	}

	lib := getScannedLibrary(t, "/tmp/test-moving-inner-directory.db")
	defer lib.Truncate()

	checkAddedSong(lib, t)

	if err := os.Rename(movedDir, secondPlace); err != nil {
		t.Error(err)
	} else {
		defer os.RemoveAll(secondPlace)
	}

	time.Sleep(10 * time.Millisecond)

	checkAddedSong(lib, t)

	found := lib.Search("")

	if len(found) != 4 {
		t.Errorf("Expected to find 4 tracks but found %s", len(found))
	}

	found = lib.Search("Added Song")

	if len(found) != 1 {
		t.Fatalf("Did not find exactly 'Added Song'. Found %d files", len(found))
	}

	foundPath := lib.GetFilePath(found[0].ID)
	expectedPath := filepath.Join(secondPlace, "test_file_added.mp3")

	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("File %s was not found: %s", expectedPath, err.Error())
	}

	if foundPath != expectedPath {
		t.Errorf("File is in %s according library. But it is actually in %s",
			foundPath, expectedPath)
	}

}
