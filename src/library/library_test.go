package library

import (
	"context"
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

func getTestLibraryPath() (string, error) {
	projRoot, err := helpers.ProjectRoot()

	if err != nil {
		return "", err
	}

	return filepath.Join(projRoot, "test_files", "library"), nil
}

// It is the caller's resposibility to remove the library SQLite database file
func getLibrary(ctx context.Context, t *testing.T) *LocalLibrary {
	lib, err := NewLocalLibrary(ctx, SQLiteMemoryFile)

	if err != nil {
		t.Fatalf(err.Error())
	}

	err = lib.Initialize()

	if err != nil {
		t.Fatalf("Initializing library: %s", err)
	}

	testLibraryPath, err := getTestLibraryPath()

	_ = lib.AddMedia(filepath.Join(testLibraryPath, "test_file_two.mp3"))
	_ = lib.AddMedia(filepath.Join(testLibraryPath, "folder_one", "third_file.mp3"))

	return lib
}

// It is the caller's resposibility to remove the library SQLite database file
func getPathedLibrary(ctx context.Context, t *testing.T) *LocalLibrary {
	projRoot, err := helpers.ProjectRoot()

	if err != nil {
		t.Fatalf("Was not able to find test_files directory: %s", err)
	}

	testLibraryPath := filepath.Join(projRoot, "test_files", "library")

	lib, err := NewLocalLibrary(ctx, SQLiteMemoryFile)

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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	lib := getPathedLibrary(ctx, t)
	defer lib.Truncate()

	ch := testErrorAfter(t, 10*time.Second, "Scanning library took too long")
	lib.Scan()
	ch <- 42

	return lib
}

func testErrorAfter(t *testing.T, dur time.Duration, message string) chan int {
	ch := make(chan int)

	go func() {
		select {
		case <-ch:
			close(ch)
			return
		case <-time.After(dur):
			log.Printf("Test timed out: %s", message)
			close(ch)
			t.Errorf(message)
			t.FailNow()
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

	lib, err := NewLocalLibrary(context.Background(), libDB.Name())

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

	var queries = []string{
		"SELECT count(id) as cnt FROM albums",
		"SELECT count(id) as cnt FROM tracks",
		"SELECT count(id) as cnt FROM artists",
	}

	for _, query := range queries {
		row, err := db.Query(query)
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

	lib, err := NewLocalLibrary(context.TODO(), libDB.Name())

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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	lib := getLibrary(ctx, t)
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

func TestAddingNewFiles(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	library := getLibrary(ctx, t)
	defer library.Truncate()

	tracksCount := func() int {
		rows, err := library.db.Query("SELECT count(id) as cnt FROM tracks")
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

func TestAlbumFSPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	library := getLibrary(ctx, t)
	defer library.Truncate()

	testLibraryPath, err := getTestLibraryPath()

	if err != nil {
		t.Fatalf("Cannot get test library path: %s", testLibraryPath)
	}

	albumPaths, err := library.GetAlbumFSPathByName("Album Of Tests")

	if err != nil {
		t.Fatalf("Was not able to find Album Of Tests: %s", err)
	}

	if len(albumPaths) != 1 {
		t.Fatalf("Expected one path for an album but found %d", len(albumPaths))
	}

	if testLibraryPath != albumPaths[0] {
		t.Errorf("Album path mismatch. Expected `%s` but got `%s`", testLibraryPath,
			albumPaths[0])
	}
}

func TestPreAddedFiles(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	library := getLibrary(ctx, t)
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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	library := getLibrary(ctx, t)
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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	lib := getPathedLibrary(ctx, t)
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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	lib := getPathedLibrary(ctx, t)
	defer lib.Truncate()

	ch := testErrorAfter(t, 10*time.Second, "Scanning library took too long")
	lib.Scan()
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
		filePaths := []string{}
		for _, track := range found {
			filePath := lib.GetFilePath(track.ID)
			filePaths = append(filePaths, fmt.Sprintf("%d: %s", track.ID, filePath))
		}
		t.Fatalf("Expected one result, got %d for Added Song: %+v. Paths:\n%s", len(found), found,
			strings.Join(filePaths, "\n"))
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

func checkSong(lib *LocalLibrary, song MediaFile, t *testing.T) {
	found := lib.Search(song.Title())

	if len(found) != 1 {
		t.Fatalf("Expected one result, got %d for %s: %+v", len(found), song.Title(), found)
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
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	lib := getPathedLibrary(ctx, t)
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

	for _, song := range mediaFiles {
		checkSong(lib, song, t)
	}
}

// Here an album which has different artists is simulated. This album must have the same
// album ID since all of the tracks are in the same directory and the same album name.
func TestAlbumsWithDifferentArtists(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	lib := getPathedLibrary(ctx, t)
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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	lib := getPathedLibrary(ctx, t)
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
