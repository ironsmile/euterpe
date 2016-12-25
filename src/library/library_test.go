package library

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/ironsmile/httpms/src/helpers"
)

func contains(heystack []string, needle string) bool {
	for _, val := range heystack {
		if needle == val {
			return true
		}
	}
	return false
}

func containsInt64(heystack []int64, needle int64) bool {
	for _, val := range heystack {
		if needle == val {
			return true
		}
	}
	return false
}

func init() {
	// Will show the output from log in the console only
	// if the -v flag is passed to the tests.
	if !contains(os.Args, "-test.v=true") {
		devnull, _ := os.Create(os.DevNull)
		log.SetOutput(devnull)
	}
}

// It is the caller's resposibility to remove the library SQLite database file
func getLibrary(t *testing.T) *LocalLibrary {

	fh, err := ioutil.TempFile("", "httpms_library_test_")

	if err != nil {
		t.Fatalf("Error creating temporary library: %s", err)
	}

	lib, err := NewLocalLibrary(fh.Name())

	if err != nil {
		t.Fatalf(err.Error())
	}

	err = lib.Initialize()

	if err != nil {
		t.Fatalf("Initializing library: %s", err)
	}

	projRoot, err := helpers.ProjectRoot()

	if err != nil {
		t.Fatalf("Was not able to find test_files directory: %s", err)
	}

	testLibraryPath := filepath.Join(projRoot, "test_files", "library")

	_ = lib.AddMedia(filepath.Join(testLibraryPath, "test_file_two.mp3"))
	_ = lib.AddMedia(filepath.Join(testLibraryPath, "folder_one", "third_file.mp3"))

	return lib
}

// It is the caller's resposibility to remove the library SQLite database file
func getPathedLibrary(t *testing.T) *LocalLibrary {
	projRoot, err := helpers.ProjectRoot()

	if err != nil {
		t.Fatalf("Was not able to find test_files directory: %s", err)
	}

	testLibraryPath := filepath.Join(projRoot, "test_files", "library")

	fh, err := ioutil.TempFile("", "httpms_library_test_")

	if err != nil {
		t.Fatalf("Error creating temporary library: %s", err)
	}

	lib, err := NewLocalLibrary(fh.Name())

	if err != nil {
		t.Fatal(err)
	}

	err = lib.Initialize()

	if err != nil {
		t.Fatalf("Initializing library: %s", err)
	}

	lib.AddLibraryPath(testLibraryPath)

	return lib
}

// It is the caller's resposibility to remove the library SQLite database file
func getScannedLibrary(t *testing.T) *LocalLibrary {
	lib := getPathedLibrary(t)

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
	libDB, err := ioutil.TempFile("", "httpms_library_test_")

	if err != nil {
		t.Fatalf("Error creating temporary library: %s", err)
	}

	lib, err := NewLocalLibrary(libDB.Name())

	if err != nil {
		t.Fatal(err)
	}

	defer lib.Close()

	err = lib.Initialize()

	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		lib.Truncate()
		os.Remove(libDB.Name())
	}()

	st, err := os.Stat(libDB.Name())

	if err != nil {
		t.Fatal(err)
	}

	if st.Size() < 1 {
		t.Errorf("Library database was 0 bytes in size")
	}

	db, err := sql.Open("sqlite3", libDB.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var tables = []string{"albums", "tracks", "artists"}

	for _, table := range tables {
		row, err := db.Query(fmt.Sprintf("SELECT count(id) as cnt FROM %s", table))
		if err != nil {
			t.Fatal(err)
		}
		defer row.Close()
	}
}

func TestTruncate(t *testing.T) {
	libDB, err := ioutil.TempFile("", "httpms_library_test_")

	if err != nil {
		t.Fatalf("Error creating temporary library: %s", err)
	}

	lib, err := NewLocalLibrary(libDB.Name())

	if err != nil {
		t.Fatal(err)
	}

	err = lib.Initialize()

	if err != nil {
		t.Fatal(err)
	}

	lib.Truncate()

	_, err = os.Stat(libDB.Name())

	if err == nil {
		os.Remove(libDB.Name())
		t.Errorf("Expected database file to be missing but it is still there")
	}
}

func TestSearch(t *testing.T) {
	lib := getLibrary(t)
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

	library := getLibrary(t)
	defer library.Truncate()

	db, err := sql.Open("sqlite3", library.database)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tracksCount := func() int {
		rows, err := db.Query("SELECT count(id) as cnt FROM tracks")
		if err != nil {
			t.Fatal(err)
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
		t.Fatal(err)
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
		t.Error(err)
	}

	tracks = tracksCount()

	if tracks != 3 {
		t.Errorf("Expected to find 3 tracks but found %d", tracks)
	}

	found := library.Search("Tittled Track")

	if len(found) != 1 {
		t.Fatalf("Expected to find one track but found %d", len(found))
	}

	track := found[0]

	if track.Title != "Tittled Track" {
		t.Errorf("Found track had the wrong title: %s", track.Title)
	}
}

func TestPreAddedFiles(t *testing.T) {
	library := getLibrary(t)
	defer library.Truncate()

	_, err := library.GetArtistID("doycho")

	if err == nil {
		t.Errorf("Was not expecting to find artist doycho")
	}

	artistID, err := library.GetArtistID("Artist Testoff")

	if err != nil {
		t.Fatalf("Was not able to find Artist Testoff: %s", err)
	}

	_, err = library.GetAlbumFSPathByName("Album Of Not Being There")

	if err == nil {
		t.Errorf("Was not expecting to find Album Of Not Being There but found one")
	}

	albumPaths, err := library.GetAlbumFSPathByName("Album Of Tests")

	if err != nil {
		t.Fatalf("Was not able to find Album Of Tests: %s", err)
	}

	if len(albumPaths) != 1 {
		t.Fatalf("Expected one path for an album but found %d", len(albumPaths))
	}

	albumID, err := library.GetAlbumID("Album Of Tests", albumPaths[0])

	if err != nil {
		t.Fatalf("Error gettin album by its name and FS path: %s", err)
	}

	_, err = library.GetTrackID("404 Not Found", artistID, albumID)

	if err == nil {
		t.Errorf("Was not expecting to find 404 Not Found track but it was there")
	}

	_, err = library.GetTrackID("Another One", artistID, albumID)

	if err != nil {
		t.Fatalf("Was not able to find track Another One: %s", err)
	}
}

func TestGettingAFile(t *testing.T) {
	library := getLibrary(t)
	defer library.Truncate()

	artistID, _ := library.GetArtistID("Artist Testoff")
	albumPaths, err := library.GetAlbumFSPathByName("Album Of Tests")

	if err != nil {
		t.Fatalf("Could not find album 'Album Of Tests': %s", err)
	}

	if len(albumPaths) != 1 {
		t.Fatalf("Expected 1 path for Album Of Tests but found %d", len(albumPaths))
	}

	albumID, err := library.GetAlbumID("Album Of Tests", albumPaths[0])

	if err != nil {
		t.Fatalf("Error getting album by its name and path: %s", err)
	}

	trackID, err := library.GetTrackID("Another One", artistID, albumID)

	if err != nil {
		t.Fatalf("File not found: %s", err)
	}

	filePath := library.GetFilePath(trackID)

	suffix := "/test_files/library/test_file_two.mp3"

	if !strings.HasSuffix(filePath, filepath.FromSlash(suffix)) {
		t.Errorf("Returned track file Another One did not have the proper file path")
	}
}

func TestAddingLibraryPaths(t *testing.T) {
	lib := getPathedLibrary(t)
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
	lib := getPathedLibrary(t)
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
	lib := getScannedLibrary(t)
	defer lib.Truncate()

	found := lib.Search(`not-such-thing" OR 1=1 OR t.name="kleopatra`)

	if len(found) != 0 {
		t.Errorf("Successful sql injection in a single query")
	}

}

func TestGetAlbumFiles(t *testing.T) {
	lib := getScannedLibrary(t)
	defer lib.Truncate()

	albumPaths, err := lib.GetAlbumFSPathByName("Album Of Tests")

	if err != nil {
		t.Fatalf("Could not find fs paths for 'Album Of Tests' album: %s", err)
	}

	albumID, _ := lib.GetAlbumID("Album Of Tests", albumPaths[0])
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
	lib := getScannedLibrary(t)
	defer lib.Truncate()

	found := lib.Search("Another One")

	if len(found) != 1 {
		t.Fatalf(`Expected searching for 'Another One' to return one `+
			`result but they were %d`, len(found))
	}

	fsPath := lib.GetFilePath(found[0].ID)

	lib.removeFile(fsPath)

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

	if track.Artist != "New Artist 2" {
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
	lib := getScannedLibrary(t)
	defer lib.Truncate()
	projRoot, _ := helpers.ProjectRoot()
	testFiles := filepath.Join(projRoot, "test_files")

	testMp3 := filepath.Join(testFiles, "more_mp3s", "test_file_added.mp3")
	newFile := filepath.Join(testFiles, "library", "folder_one", "test_file_added.mp3")

	if err := helpers.Copy(testMp3, newFile); err != nil {
		t.Fatalf("Copying file to library faild: %s", err)
	}

	defer os.Remove(newFile)
	lib.WaitScan()

	checkAddedSong(lib, t)
}

func TestMovingFileIntoLibrary(t *testing.T) {
	lib := getScannedLibrary(t)
	defer lib.Truncate()
	projRoot, _ := helpers.ProjectRoot()
	testFiles := filepath.Join(projRoot, "test_files")

	testMp3 := filepath.Join(testFiles, "more_mp3s", "test_file_added.mp3")
	toBeMoved := filepath.Join(testFiles, "more_mp3s", "test_file_moved.mp3")
	newFile := filepath.Join(testFiles, "library", "test_file_added.mp3")

	if err := helpers.Copy(testMp3, toBeMoved); err != nil {
		t.Fatalf("Copying file to library faild: %s", err)
	}

	if err := os.Rename(toBeMoved, newFile); err != nil {
		os.Remove(toBeMoved)
		t.Fatalf("Was not able to move new file into library: %s", err)
	}

	defer os.Remove(newFile)
	lib.WaitScan()

	checkAddedSong(lib, t)
}

func TestAddingNonRelatedFile(t *testing.T) {
	lib := getScannedLibrary(t)
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

	lib.WaitScan()
	testLibFiles()

	os.Remove(newFile)
	lib.WaitScan()
	testLibFiles()

}

func TestRemovingFile(t *testing.T) {
	projRoot, _ := helpers.ProjectRoot()
	testFiles := filepath.Join(projRoot, "test_files")

	testMp3 := filepath.Join(testFiles, "more_mp3s", "test_file_added.mp3")
	newFile := filepath.Join(testFiles, "library", "test_file_added.mp3")

	if err := helpers.Copy(testMp3, newFile); err != nil {
		t.Fatalf("Copying file to library faild: %s", err)
	}

	lib := getScannedLibrary(t)
	defer lib.Truncate()

	results := lib.Search("")

	if len(results) != 4 {
		t.Errorf("Expected 4 files in the result set but found %d", len(results))
	}

	if err := os.Remove(newFile); err != nil {
		t.Error(err)
	}

	lib.WaitScan()

	results = lib.Search("")
	if len(results) != 3 {
		t.Errorf("Expected 3 files in the result set but found %d", len(results))
	}
}

func TestAddingAndRemovingDirectory(t *testing.T) {
	projRoot, _ := helpers.ProjectRoot()
	testFiles := filepath.Join(projRoot, "test_files")

	testDir := filepath.Join(testFiles, "more_mp3s", "to_be_moved_directory")
	movedDir := filepath.Join(testFiles, "library", "to_be_moved_directory")
	srcTestMp3 := filepath.Join(testFiles, "more_mp3s", "test_file_added.mp3")
	dstTestMp3 := filepath.Join(testFiles, "more_mp3s", "to_be_moved_directory",
		"test_file_added.mp3")

	if err := os.RemoveAll(testDir); err != nil {
		t.Fatalf("Removing directory needed for tests: %s", err)
	}

	if err := os.RemoveAll(movedDir); err != nil {
		t.Fatalf("Removing directory needed for tests: %s", err)
	}

	lib := getScannedLibrary(t)
	defer lib.Truncate()

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

	lib.WaitScan()

	checkAddedSong(lib, t)

	if err := os.RemoveAll(movedDir); err != nil {
		t.Error(err)
	}

	lib.WaitScan()

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

	_ = os.RemoveAll(testDir)
	_ = os.RemoveAll(movedDir)
	_ = os.RemoveAll(secondPlace)

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

	lib := getScannedLibrary(t)
	defer lib.Truncate()

	checkAddedSong(lib, t)

	if err := os.Rename(movedDir, secondPlace); err != nil {
		t.Error(err)
	} else {
		defer os.RemoveAll(secondPlace)
	}

	lib.WaitScan()

	checkAddedSong(lib, t)

	found := lib.Search("")

	if len(found) != 4 {
		t.Errorf("Expected to find 4 tracks but found %d", len(found))
	}

	found = lib.Search("Added Song")

	if len(found) != 1 {
		t.Fatalf("Did not find exactly one 'Added Song'. Found %d files", len(found))
	}

	foundPath := lib.GetFilePath(found[0].ID)
	expectedPath := filepath.Join(secondPlace, "test_file_added.mp3")

	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("File %s was not found: %s", expectedPath, err)
	}

	if foundPath != expectedPath {
		t.Errorf("File is in %s according to library. But it is actually in %s",
			foundPath, expectedPath)
	}

}

func checkSong(lib *LocalLibrary, song MediaFile, t *testing.T) {
	found := lib.Search(song.Title())

	if len(found) != 1 {
		t.Fatalf("Expected one result, got %d for %s", len(found), song.Title())
	}

	track := found[0]

	if track.Album != song.Album() {
		t.Errorf("Wrong track album: %s when expecting %s", track.Album, song.Album())
	}

	if track.Artist != song.Artist() {
		t.Errorf("Wrong track artist: %s when expecting %s", track.Artist, song.Artist())
	}

	if track.Title != song.Title() {
		t.Errorf("Wrong track title: %s when expecting %s", track.Title, song.Title())
	}

	if track.TrackNumber != int64(song.Track()) {
		t.Errorf("Wrong track: %d when expecting %d", track.TrackNumber, song.Track())
	}
}

func TestAddingManyFilesSimultaniously(t *testing.T) {
	lib := getPathedLibrary(t)
	defer lib.Truncate()

	numberOfFiles := 100
	mediaFiles := make([]MediaFile, 0, numberOfFiles)

	for i := 0; i < numberOfFiles; i++ {
		m := &MockMedia{
			artist: fmt.Sprintf("artist %d", i),
			album:  fmt.Sprintf("album %d", i),
			title:  fmt.Sprintf("title %d full", i),
			track:  i,
			length: 123 * time.Second,
		}
		mPath := fmt.Sprintf("/path/to/file_%d", i)

		if err := lib.insertMediaIntoDatabase(m, mPath); err != nil {
			t.Fatalf("Error adding media into the database: %s", err)
		}

		mediaFiles = append(mediaFiles, m)
	}

	lib.waitForDBWriterIdleSignal()

	for _, song := range mediaFiles {
		checkSong(lib, song, t)
	}
}

// Here an album which has different artists is simulated. This album must have the same
// album ID since all of the tracks are in the same directory and the same album name.
func TestAlbumsWithDifferentArtists(t *testing.T) {
	lib := getPathedLibrary(t)
	defer lib.Truncate()

	var err error

	tracks := []MockMedia{
		MockMedia{
			artist: "Buggy Bugoff",
			album:  "Return Of The Bugs",
			title:  "Payback",
			track:  1,
			length: 340 * time.Second,
		},
		MockMedia{
			artist: "Buggy Bugoff",
			album:  "Return Of The Bugs",
			title:  "Realization",
			track:  2,
			length: 345 * time.Second,
		},
		MockMedia{
			artist: "Off By One",
			album:  "Return Of The Bugs",
			title:  "Index By Index",
			track:  3,
			length: 244 * time.Second,
		},
	}

	for _, track := range tracks {
		err = lib.insertMediaIntoDatabase(
			&track,
			fmt.Sprintf("/media/return-of-the-bugs/%s.mp3", track.Title()),
		)

		if err != nil {
			t.Fatalf("Adding a media file %s failed: %s", track.Title(), err)
		}
	}

	lib.waitForDBWriterIdleSignal()

	found := lib.Search("Return Of The Bugs")

	if len(found) != 3 {
		t.Errorf("Expected to find 3 tracks but found %d", len(found))
	}

	albumID := found[0].AlbumID
	albumName := found[0].Album

	for _, foundTrack := range found {
		if foundTrack.AlbumID != albumID {
			t.Errorf("Track %s had a different album id in db", foundTrack.Title)
		}

		if foundTrack.Album != albumName {
			t.Errorf(
				"Track %s had a different album name: %s",
				foundTrack.Title,
				foundTrack.Album,
			)
		}
	}
}

// Albums with the same name which are for different artists should have different IDs
// when the album is in a different directory
func TestDifferentAlbumsWithTheSameName(t *testing.T) {
	lib := getPathedLibrary(t)
	defer lib.Truncate()

	tracks := []struct {
		track MockMedia
		path  string
	}{
		{
			MockMedia{
				artist: "Buggy Bugoff",
				album:  "Return Of The Bugs",
				title:  "Payback",
				track:  1,
				length: 340 * time.Second,
			},
			"/media/return-of-the-bugs/track-1.mp3",
		},
		{
			MockMedia{
				artist: "Buggy Bugoff",
				album:  "Return Of The Bugs",
				title:  "Realization",
				track:  2,
				length: 345 * time.Second,
			},
			"/media/return-of-the-bugs/track-2.mp3",
		},
		{
			MockMedia{
				artist: "Off By One",
				album:  "Return Of The Bugs",
				title:  "Index By Index",
				track:  1,
				length: 244 * time.Second,
			},
			"/media/second-return-of-the-bugs/track-1.mp3", // different directory
		},
	}

	for _, trackData := range tracks {
		err := lib.insertMediaIntoDatabase(&trackData.track, trackData.path)

		if err != nil {
			t.Fatalf("Adding a media file %s failed: %s", trackData.track.Title(), err)
		}
	}

	lib.waitForDBWriterIdleSignal()

	found := lib.Search("Return Of The Bugs")

	if len(found) != 3 {
		t.Errorf("Expected to find 3 tracks but found %d", len(found))
	}

	albumIDs := make([]int64, 0, 2)

	for _, track := range found {
		if containsInt64(albumIDs, track.AlbumID) {
			continue
		}
		albumIDs = append(albumIDs, track.AlbumID)
	}

	if len(albumIDs) != 2 {
		t.Errorf(
			"There should have been two 'Return Of The Bugs' albums but there were %d",
			len(albumIDs),
		)
	}
}
