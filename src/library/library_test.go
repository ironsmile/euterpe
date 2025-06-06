package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	// Needed for tests as the go-sqlite3 must be imported during tests too.
	_ "github.com/mattn/go-sqlite3"

	"github.com/ironsmile/euterpe/src/assert"
	"github.com/ironsmile/euterpe/src/helpers"
)

// testTimeout is the maximum time a test is allowed to work.
var testTimeout = 40 * time.Second

func contains(heystack []string, needle string) bool {
	return slices.Contains(heystack, needle)
}

func containsInt64(heystack []int64, needle int64) bool {
	return slices.Contains(heystack, needle)
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

// It is the caller's responsibility to remove the library SQLite database file
func getLibrary(ctx context.Context, t *testing.T) *LocalLibrary {
	lib, err := NewLocalLibrary(ctx, SQLiteMemoryFile, getTestMigrationFiles())

	if err != nil {
		t.Fatal(err.Error())
	}

	err = lib.Initialize()

	if err != nil {
		t.Fatalf("Initializing library: %s", err)
	}

	testLibraryPath, err := getTestLibraryPath()

	if err != nil {
		t.Fatalf("Failed to get test library path: %s", err)
	}

	_ = lib.AddMedia(filepath.Join(testLibraryPath, "test_file_two.mp3"))
	_ = lib.AddMedia(filepath.Join(testLibraryPath, "folder_one", "third_file.mp3"))

	return lib
}

// It is the caller's responsibility to remove the library SQLite database file
func getPathedLibrary(ctx context.Context, t *testing.T) *LocalLibrary {
	projRoot, err := helpers.ProjectRoot()

	if err != nil {
		t.Fatalf("Was not able to find test_files directory: %s", err)
	}

	testLibraryPath := filepath.Join(projRoot, "test_files", "library")

	lib, err := NewLocalLibrary(ctx, SQLiteMemoryFile, getTestMigrationFiles())

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

// It is the caller's responsibility to remove the library SQLite database file
func getScannedLibrary(ctx context.Context, t *testing.T) *LocalLibrary {
	lib := getPathedLibrary(ctx, t)
	waitForLibraryScan(t, lib)
	return lib
}

func waitForLibraryScan(t *testing.T, lib *LocalLibrary) {
	ch := make(chan int)
	go func() {
		lib.Scan()
		ch <- 42
	}()

	testErrorAfter(t, 10*time.Second, ch, "Scanning library took too long")
}

func testErrorAfter(t *testing.T, dur time.Duration, done chan int, message string) {
	select {
	case <-done:
	case <-time.After(dur):
		t.Errorf("Test timed out after %s: %s", dur, message)
		t.FailNow()
	}
}

func TestInitialize(t *testing.T) {
	libDB, err := os.CreateTemp("", "httpms_library_test_")

	if err != nil {
		t.Fatalf("Error creating temporary library: %s", err)
	}

	lib, err := NewLocalLibrary(context.Background(), libDB.Name(), getTestMigrationFiles())

	if err != nil {
		t.Fatal(err)
	}

	defer lib.Close()

	err = lib.Initialize()

	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_ = lib.Truncate()
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
	libDB, err := os.CreateTemp("", "httpms_library_test_")

	if err != nil {
		t.Fatalf("Error creating temporary library: %s", err)
	}

	lib, err := NewLocalLibrary(context.TODO(), libDB.Name(), getTestMigrationFiles())

	if err != nil {
		t.Fatal(err)
	}

	err = lib.Initialize()

	if err != nil {
		t.Fatal(err)
	}

	_ = lib.Truncate()

	_, err = os.Stat(libDB.Name())

	if err == nil {
		os.Remove(libDB.Name())
		t.Errorf("Expected database file to be missing but it is still there")
	}
}

// TestSearch checks that searching for tracks, albums and artists
// populate all of their fields.
func TestSearch(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), testTimeout)
	defer cancel()
	lib := getLibrary(ctx, t)
	defer func() {
		_ = lib.Truncate()
	}()

	tracks := lib.Search(ctx, SearchArgs{
		Query: "Buggy",
	})

	if len(tracks) != 1 {
		t.Fatalf("Expected 1 result but got %d", len(tracks))
	}

	now := time.Now()
	favs := Favourites{TrackIDs: []int64{tracks[0].ID}}
	if err := lib.RecordFavourite(ctx, favs); err != nil {
		t.Fatalf("Cannot set track favourite: %s", err)
	}

	if err := lib.RecordTrackPlay(ctx, tracks[0].ID, now); err != nil {
		t.Fatalf("Failed to record playing time: %s", err)
	}

	if err := lib.SetTrackRating(ctx, tracks[0].ID, 3); err != nil {
		t.Fatalf("Failed to set track rating: %s", err)
	}

	// Search for the track again once its attributes have been updated.
	tracks = lib.Search(ctx, SearchArgs{
		Query: "Buggy",
	})

	if len(tracks) != 1 {
		t.Fatalf("Expected 1 result but got %d", len(tracks))
	}

	expected := SearchResult{
		Artist:      "Buggy Bugoff",
		Album:       "Return Of The Bugs",
		Title:       "Payback",
		TrackNumber: 1,
		Year:        2013,
		Format:      "mp3",
		Duration:    1000,
		Bitrate:     0x20c00,
		Size:        17314,
		Favourite:   now.Unix(),
		LastPlayed:  now.Unix(),
		Rating:      3,
		Plays:       1,
	}

	assertTrack(t, expected, tracks[0])

	// Now test the SearchAlbums function.
	albums := lib.SearchAlbums(ctx, SearchArgs{Query: "Return Of The Bugs"})
	if len(albums) != 1 {
		t.Fatalf("Expected one album but got %d", len(albums))
	}

	expectedAlbum := Album{
		Name:       expected.Album,
		ID:         tracks[0].AlbumID,
		Artist:     expected.Artist,
		SongCount:  1,
		Duration:   expected.Duration,
		Plays:      expected.Plays,
		LastPlayed: expected.LastPlayed,
		Year:       expected.Year,
		Favourite:  0,
		Rating:     0,
	}

	assertAlbum(t, expectedAlbum, albums[0])

	// Rate album and add it to favourites and try again.
	now = time.Now()
	favs = Favourites{AlbumIDs: []int64{albums[0].ID}}
	if err := lib.RecordFavourite(ctx, favs); err != nil {
		t.Errorf("Failed to mark album as favourite: %s", err)
	}
	expectedAlbum.Favourite = now.Unix()

	expectedAlbum.Rating = 4
	if err := lib.SetAlbumRating(ctx, albums[0].ID, expectedAlbum.Rating); err != nil {
		t.Errorf("Failed to set album rating: %s", err)
	}

	// Get the latest data for the album from the database.
	albums = lib.SearchAlbums(ctx, SearchArgs{Query: "Return Of The Bugs"})
	if len(albums) != 1 {
		t.Fatalf("Expected one album but got %d", len(albums))
	}

	assertAlbum(t, expectedAlbum, albums[0])

	// Now search for the artist.
	artists := lib.SearchArtists(ctx, SearchArgs{Query: "Bugoff"})
	if len(artists) != 1 {
		t.Fatalf("Expected to get one artist but got %d", len(artists))
	}

	expectedArtist := Artist{
		ID:         tracks[0].ArtistID,
		Name:       tracks[0].Artist,
		AlbumCount: 1,
		Favourite:  0,
		Rating:     0,
	}

	assertArtist(t, expectedArtist, artists[0])

	// Set artist ratings and mark as favourite and then search again.
	now = time.Now()
	favs = Favourites{ArtistIDs: []int64{artists[0].ID}}
	if err := lib.RecordFavourite(ctx, favs); err != nil {
		t.Errorf("Failed to mark artist as favourite: %s", err)
	}
	expectedArtist.Favourite = now.Unix()

	expectedArtist.Rating = 2
	if err := lib.SetArtistRating(ctx, artists[0].ID, expectedArtist.Rating); err != nil {
		t.Errorf("Failed to set artist rating: %s", err)
	}

	artists = lib.SearchArtists(ctx, SearchArgs{Query: artists[0].Name})
	if len(artists) != 1 {
		t.Fatalf("Expected to get one artist but got %d", len(artists))
	}

	assertArtist(t, expectedArtist, artists[0])
}

// assertTrack checks that expected is the same as actual but skips
// checking for the actual track ID as it may not be known beforehand.
func assertTrack(t *testing.T, expected, actual TrackInfo) {
	t.Helper()

	if actual.Artist != expected.Artist {
		t.Errorf("Expected Artist `%s` but found `%s`",
			expected.Artist, actual.Artist)
	}

	if actual.Title != expected.Title {
		t.Errorf("Expected Title `%s` but found `%s`",
			expected.Title, actual.Title)
	}

	if actual.Album != expected.Album {
		t.Errorf("Expected Album `%s` but found `%s`",
			expected.Album, actual.Album)
	}

	if actual.TrackNumber != expected.TrackNumber {
		t.Errorf("Expected TrackNumber `%d` but found `%d`",
			expected.TrackNumber, actual.TrackNumber)
	}

	if actual.AlbumID < 1 {
		t.Errorf("AlbumID was below 1: `%d`", actual.AlbumID)
	}

	assert.Equal(t, expected.Year, actual.Year, "track year")
	assert.Equal(t, expected.Format, actual.Format, "file format")
	assert.Equal(t, expected.Duration, actual.Duration, "track duration")
	assert.Equal(t, expected.LastPlayed, actual.LastPlayed, "track last played at")
	assert.Equal(t, expected.Plays, actual.Plays, "track plays")
	assert.Equal(t, expected.Favourite, actual.Favourite, "track favourite date")
	assert.Equal(t, expected.Bitrate, actual.Bitrate, "track bitrate")
	assert.Equal(t, expected.Rating, actual.Rating, "track rating")
	assert.Equal(t, expected.Size, actual.Size, "track file size")
}

// assertAlbum asserts that `actual` is the same as `expected`.
func assertAlbum(t *testing.T, expected, actual Album) {
	t.Helper()

	assert.Equal(t, expected.Name, actual.Name, "album name")
	assert.Equal(t, expected.ID, actual.ID, "album ID")
	assert.Equal(t, expected.Artist, actual.Artist, "album artist")
	assert.Equal(t, expected.SongCount, actual.SongCount, "album tracks count")
	assert.Equal(t, expected.Duration, actual.Duration, "album duration")
	assert.Equal(t, expected.Plays, actual.Plays, "album play count")
	assert.Equal(t, expected.LastPlayed, actual.LastPlayed, "album last played")
	assert.Equal(t, expected.Year, actual.Year, "album year")
	assert.Equal(t, expected.Favourite, actual.Favourite, "album favourite status")
	assert.Equal(t, expected.Rating, actual.Rating, "album rating")
	assert.Equal(t, expected.AvgBitrate, actual.AvgBitrate, "album avg bitrate")
}

// assertArtist asserts that `actual` is the same as `expected`.
func assertArtist(t *testing.T, expected, actual Artist) {
	t.Helper()

	assert.Equal(t, expected.Name, actual.Name, "artist name")
	assert.Equal(t, expected.ID, actual.ID, "artist ID")
	assert.Equal(t, expected.AlbumCount, actual.AlbumCount, "albums count")
	assert.Equal(t, expected.Favourite, actual.Favourite, "artist favourites status")
	assert.Equal(t, expected.Rating, actual.Rating, "artist ratings")
}

// TestNotFoundErrors checks that "not found" errors are returned while trying
// to get albums, artists and tracks which do not exist.
func TestNotFoundErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), testTimeout)
	defer cancel()
	lib := getLibrary(ctx, t)
	defer func() {
		_ = lib.Truncate()
	}()

	if _, err := lib.GetTrack(ctx, 98234); !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected 'not found' error for track but got: %s", err)
	}

	if _, err := lib.GetAlbum(ctx, 827362); !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected 'not found' error for album but got: %s", err)
	}

	if _, err := lib.GetArtist(ctx, 8173); !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected 'not found' error for artist but got: %s", err)
	}
}

// TestGettingSpecificData checks that getting tracks, albums and artists
// by their ID is populating all of their properties.
func TestGettingSpecificData(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), testTimeout)
	defer cancel()
	lib := getLibrary(ctx, t)
	defer func() {
		_ = lib.Truncate()
	}()

	tracks := lib.Search(ctx, SearchArgs{
		Query: "Buggy",
	})

	if len(tracks) != 1 {
		t.Fatalf("Expected 1 result but got %d", len(tracks))
	}

	now := time.Now()
	favs := Favourites{
		TrackIDs:  []int64{tracks[0].ID},
		ArtistIDs: []int64{tracks[0].ArtistID},
		AlbumIDs:  []int64{tracks[0].AlbumID},
	}
	if err := lib.RecordFavourite(ctx, favs); err != nil {
		t.Fatalf("Cannot set track favourite: %s", err)
	}

	if err := lib.RecordTrackPlay(ctx, tracks[0].ID, now); err != nil {
		t.Fatalf("Failed to record playing time: %s", err)
	}

	if err := lib.SetTrackRating(ctx, tracks[0].ID, 3); err != nil {
		t.Fatalf("Failed to set track rating: %s", err)
	}

	if err := lib.SetAlbumRating(ctx, tracks[0].AlbumID, 4); err != nil {
		t.Fatalf("Failed to set album rating: %s", err)
	}

	if err := lib.SetArtistRating(ctx, tracks[0].ArtistID, 1); err != nil {
		t.Fatalf("Failed to set artist rating: %s", err)
	}

	expectedTrack := tracks[0]
	expectedTrack.Rating = 3
	expectedTrack.Favourite = now.Unix()
	expectedTrack.Plays = 1
	expectedTrack.LastPlayed = now.Unix()

	track, err := lib.GetTrack(ctx, tracks[0].ID)
	if err != nil {
		t.Fatalf("Failed to get track by ID: %s", err)
	}
	assertTrack(t, expectedTrack, track)

	expectedAlbum := Album{
		ID:         track.AlbumID,
		Name:       track.Album,
		Artist:     track.Artist,
		SongCount:  1,
		Duration:   track.Duration,
		Plays:      1,
		Favourite:  now.Unix(),
		LastPlayed: now.Unix(),
		Rating:     4,
		Year:       track.Year,
		AvgBitrate: 134144,
	}

	album, err := lib.GetAlbum(ctx, track.AlbumID)
	if err != nil {
		t.Errorf("Failed to get album by ID: %s", err)
	}
	assertAlbum(t, expectedAlbum, album)

	expectedArtist := Artist{
		ID:         track.ArtistID,
		Name:       track.Artist,
		AlbumCount: 1,
		Favourite:  now.Unix(),
		Rating:     1,
	}

	artist, err := lib.GetArtist(ctx, track.ArtistID)
	if err != nil {
		t.Errorf("Failed to get artist by ID: %s", err)
	}
	assertArtist(t, expectedArtist, artist)
}

func TestAddingNewFiles(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	library := getLibrary(ctx, t)
	defer func() {
		_ = library.Truncate()
	}()

	tracksCount := func() int {
		rows, err := library.db.Query("SELECT count(id) as cnt FROM tracks")
		if err != nil {
			t.Fatal(err)
			return 0
		}
		defer rows.Close()

		var count int

		for rows.Next() {
			if err := rows.Scan(&count); err != nil {
				t.Errorf("error counting tracks: %s", err)
			}
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

	found := library.Search(ctx, SearchArgs{Query: "Tittled Track"})

	if len(found) != 1 {
		t.Fatalf("Expected to find one track but found %d", len(found))
	}

	track := found[0]

	if track.Title != "Tittled Track" {
		t.Errorf("Found track had the wrong title: %s", track.Title)
	}
}

func TestAlbumFSPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	library := getLibrary(ctx, t)
	defer func() { _ = library.Truncate() }()

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
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	library := getLibrary(ctx, t)
	defer func() { _ = library.Truncate() }()

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
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	library := getLibrary(ctx, t)
	defer func() { _ = library.Truncate() }()

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

	filePath := library.GetFilePath(ctx, trackID)

	suffix := "/test_files/library/test_file_two.mp3"

	if !strings.HasSuffix(filePath, filepath.FromSlash(suffix)) {
		t.Errorf("Returned track file Another One did not have the proper file path")
	}
}

func TestAddingLibraryPaths(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getPathedLibrary(ctx, t)
	defer func() {
		_ = lib.Truncate()
	}()

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
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getPathedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()

	ch := make(chan int)
	go func() {
		lib.Scan()
		ch <- 42
	}()
	testErrorAfter(t, 10*time.Second, ch, "Scanning library took too long")

	for _, track := range []string{"Another One", "Payback", "Tittled Track"} {
		found := lib.Search(ctx, SearchArgs{Query: track})

		if len(found) != 1 {
			t.Errorf("%s was not found after the scan", track)
		}
	}
}

// TestRescanning alters a file in the database and then does a rescan, expecting
// its data to be synchronized back to what is on the filesystem.
func TestRescanning(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getPathedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()

	ch := make(chan int)
	go func() {
		lib.Scan()
		ch <- 42
	}()
	testErrorAfter(t, 10*time.Second, ch, "Scanning library took too long")

	const alterTrackQuery = `
		UPDATE tracks
		SET
			name = 'Broken File'
		WHERE
			name = 'Another One'
	`
	if _, err := lib.db.Exec(alterTrackQuery); err != nil {
		t.Fatalf("altering track in the database failed")
	}

	var rescanErr error
	go func() {
		rescanErr = lib.Rescan(ctx)
		ch <- 42
	}()
	testErrorAfter(t, 10*time.Second, ch, "Rescanning library took too long")

	if rescanErr != nil {
		t.Fatalf("rescan returned an error: %s", rescanErr)
	}

	for _, track := range []string{"Another One", "Payback", "Tittled Track"} {
		found := lib.Search(ctx, SearchArgs{Query: track})

		if len(found) != 1 {
			t.Errorf("%s was not found after the scan", track)
		}
	}
}

func TestSQLInjections(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getScannedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()

	found := lib.Search(ctx, SearchArgs{
		Query: `not-such-thing" OR 1=1 OR t.name="kleopatra`,
	})

	if len(found) != 0 {
		t.Errorf("Successful sql injection in a single query")
	}
}

func TestGetAlbumFiles(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getScannedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()

	albumPaths, err := lib.GetAlbumFSPathByName("Album Of Tests")

	if err != nil {
		t.Fatalf("Could not find fs paths for 'Album Of Tests' album: %s", err)
	}

	albumID, _ := lib.GetAlbumID("Album Of Tests", albumPaths[0])
	albumFiles := lib.GetAlbumFiles(ctx, albumID)

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
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getScannedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()

	found := lib.Search(ctx, SearchArgs{Query: "Another One"})

	if len(found) != 1 {
		t.Fatalf(`Expected searching for 'Another One' to return one `+
			`result but they were %d`, len(found))
	}

	fsPath := lib.GetFilePath(ctx, found[0].ID)

	lib.removeFile(fsPath)

	found = lib.Search(ctx, SearchArgs{Query: "Another One"})

	if len(found) != 0 {
		t.Error(`Did not expect to find Another One but it was there.`)
	}
}

func checkAddedSong(ctx context.Context, lib *LocalLibrary, t *testing.T) {
	found := lib.Search(ctx, SearchArgs{Query: "Added Song"})

	if len(found) != 1 {
		filePaths := []string{}
		for _, track := range found {
			filePath := lib.GetFilePath(ctx, track.ID)
			filePaths = append(filePaths, fmt.Sprintf("%d: %s", track.ID, filePath))
		}
		t.Fatalf("Expected one result, got %d for Added Song: %+v. Paths:\n%s",
			len(found), found, strings.Join(filePaths, "\n"))
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

func checkSong(ctx context.Context, lib *LocalLibrary, song MediaFile, t *testing.T) {
	found := lib.Search(ctx, SearchArgs{Query: song.Title()})

	if len(found) != 1 {
		t.Fatalf("Expected one result, got %d for %s: %+v",
			len(found), song.Title(), found)
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

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getPathedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()

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
		mInfo := fileInfo{
			Size:     int64(m.Length().Seconds()) * 256000,
			FilePath: fmt.Sprintf("/path/to/file_%d", i),
			Modified: time.Now(),
		}

		if err := lib.insertMediaIntoDatabase(m, mInfo); err != nil {
			t.Fatalf("Error adding media into the database: %s", err)
		}

		mediaFiles = append(mediaFiles, m)
	}

	for _, song := range mediaFiles {
		checkSong(ctx, lib, song, t)
	}
}

// TestAlbumsWithDifferentArtists simulates an album which has different artists.
// This album must have the same album ID since all of the tracks are in the same
// directory and the same album name.
func TestAlbumsWithDifferentArtists(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getPathedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()

	var err error

	tracks := []MockMedia{
		{
			artist: "Buggy Bugoff",
			album:  "Return Of The Bugs",
			title:  "Payback",
			track:  1,
			length: 340 * time.Second,
		},
		{
			artist: "Buggy Bugoff",
			album:  "Return Of The Bugs",
			title:  "Realization",
			track:  2,
			length: 345 * time.Second,
		},
		{
			artist: "Off By One",
			album:  "Return Of The Bugs",
			title:  "Index By Index",
			track:  3,
			length: 244 * time.Second,
		},
	}

	for _, track := range tracks {
		trackInfo := fileInfo{
			Size:     int64(track.Length().Seconds()) * 256000,
			FilePath: fmt.Sprintf("/media/return-of-the-bugs/%s.mp3", track.Title()),
			Modified: time.Now(),
		}
		err = lib.insertMediaIntoDatabase(&track, trackInfo)
		if err != nil {
			t.Fatalf("Adding a media file %s failed: %s", track.Title(), err)
		}
	}

	found := lib.Search(ctx, SearchArgs{Query: "Return Of The Bugs"})

	if len(found) != 3 {
		t.Fatalf("Expected to find 3 tracks but found %d", len(found))
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

// TestDifferentAlbumsWithTheSameName makes sure that albums with the same name which
// are for different artists should have different IDs when the album is in a different
// directory.
func TestDifferentAlbumsWithTheSameName(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getPathedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()

	tracks := []struct {
		track MockMedia
		path  string
	}{
		{
			track: MockMedia{
				artist: "Buggy Bugoff",
				album:  "Return Of The Bugs",
				title:  "Payback",
				track:  1,
				length: 340 * time.Second,
			},
			path: "/media/return-of-the-bugs/track-1.mp3",
		},
		{
			track: MockMedia{
				artist: "Buggy Bugoff",
				album:  "Return Of The Bugs",
				title:  "Realization",
				track:  2,
				length: 345 * time.Second,
			},
			path: "/media/return-of-the-bugs/track-2.mp3",
		},
		{
			track: MockMedia{
				artist: "Off By One",
				album:  "Return Of The Bugs",
				title:  "Index By Index",
				track:  1,
				length: 244 * time.Second,
			},
			path: "/media/second-return-of-the-bugs/track-1.mp3", // different directory
		},
	}

	for _, trackData := range tracks {
		fileInfo := fileInfo{
			Size:     int64(trackData.track.Length().Seconds()) * 256000,
			FilePath: trackData.path,
			Modified: time.Now(),
		}
		err := lib.insertMediaIntoDatabase(&trackData.track, fileInfo)

		if err != nil {
			t.Fatalf("Adding a media file %s failed: %s", trackData.track.Title(), err)
		}
	}

	found := lib.Search(ctx, SearchArgs{Query: "Return Of The Bugs"})

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

// TestLocalLibrarySupportedFormats makes sure that format recognition from file name
// does return true only for supported formats.
func TestLocalLibrarySupportedFormats(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{
			path:     filepath.FromSlash("some/path.mp3"),
			expected: true,
		},
		{
			path:     filepath.FromSlash("path.mp3"),
			expected: true,
		},
		{
			path:     filepath.FromSlash("some/path.ogg"),
			expected: true,
		},
		{
			path:     filepath.FromSlash("some/path.wav"),
			expected: true,
		},
		{
			path:     filepath.FromSlash("some/path.fla"),
			expected: true,
		},
		{
			path:     filepath.FromSlash("some/path.flac"),
			expected: true,
		},
		{
			path:     filepath.FromSlash("path.flac"),
			expected: true,
		},
		{
			path:     filepath.FromSlash("some/.mp3"),
			expected: false,
		},
		{
			path:     filepath.FromSlash("file.MP3"),
			expected: true,
		},
		{
			path:     filepath.FromSlash("some/file.pdf"),
			expected: false,
		},
		{
			path:     filepath.FromSlash("some/mp3"),
			expected: false,
		},
		{
			path:     filepath.FromSlash("mp3"),
			expected: false,
		},
		{
			path:     filepath.FromSlash("somewhere/file.opus"),
			expected: true,
		},
		{
			path:     filepath.FromSlash("somewhere/FILE.webm"),
			expected: true,
		},
		{
			path:     filepath.FromSlash("somewhere/other.WEbm"),
			expected: true,
		},
		{
			path:     filepath.FromSlash("/proc/cpuinfo"),
			expected: false,
		},
	}

	// lib does not need to be initialized. The isSupportedFormat method does not
	// touch any of its properties.
	lib := LocalLibrary{}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			actual := lib.isSupportedFormat(test.path)
			if test.expected != actual {
				t.Errorf("Support for %s is wrong. Expected %t but got %t.",
					test.path, test.expected, actual)
			}
		})
	}
}

// TestLocalLibraryGetArtistAlbums makes sure that the LocalLibrary's GetArtistAlbums
// returns the expected results.
func TestLocalLibraryGetArtistAlbums(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getPathedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()

	const (
		albumName       = "Return Of The Bugs"
		secondAlbumName = "Return Of The Bugs II Deluxe 3000"
		artistName      = "Buggy Bugoff"
		albumYear       = 2014
	)

	tracks := []struct {
		track MockMedia
		path  string
	}{
		{
			track: MockMedia{
				artist: artistName,
				album:  albumName,
				title:  "Payback",
				track:  1,
				length: 340 * time.Second,
				year:   albumYear,
			},
			path: "/media/return-of-the-bugs/track-1.mp3",
		},
		{
			track: MockMedia{
				artist: artistName,
				album:  albumName,
				title:  "Realization",
				track:  2,
				length: 345 * time.Second,
				year:   albumYear,
			},
			path: "/media/return-of-the-bugs/track-2.mp3",
		},
		{
			track: MockMedia{
				artist: artistName,
				album:  secondAlbumName,
				title:  "Index By Index",
				track:  1,
				length: 244 * time.Second,
			},
			path: "/media/second-return-of-the-bugs/track-1.mp3",
		},
		{
			track: MockMedia{
				artist: "Nothing To Do With The Rest",
				album:  "Maybe Some Less Bugs Please",
				title:  "Test By Test",
				track:  1,
				length: 523 * time.Second,
			},
			path: "/media/maybe-some-less-bugs/track-1.mp3",
		},
	}

	expected := map[string]Album{
		albumName: {
			Name:   albumName,
			Artist: artistName,
			Year:   albumYear,
		},
		secondAlbumName: {
			Name:   secondAlbumName,
			Artist: artistName,
		},
	}

	for _, trackData := range tracks {
		trackInfo := fileInfo{
			Size:     int64(trackData.track.Length().Seconds()) * 256000,
			FilePath: trackData.path,
			Modified: time.Now(),
		}
		err := lib.insertMediaIntoDatabase(&trackData.track, trackInfo)

		if err != nil {
			t.Fatalf("Adding a media file %s failed: %s", trackData.track.Title(), err)
		}

		al, ok := expected[trackData.track.album]
		if !ok {
			continue
		}

		al.Duration += trackData.track.length.Milliseconds()
		al.SongCount++

		expected[al.Name] = al
	}

	// Record some plays, set favourite and rating for an album.
	toRecordTracks := lib.Search(ctx, SearchArgs{Query: "Index By Index"})
	if len(toRecordTracks) != 1 {
		t.Fatalf("could not find album to record plays to")
	}

	recordTime := time.Now()
	favs := Favourites{
		AlbumIDs: []int64{toRecordTracks[0].AlbumID},
	}
	if err := lib.RecordFavourite(ctx, favs); err != nil {
		t.Fatalf("failed to set album '%s' as favourite: %s",
			toRecordTracks[0].Album, err,
		)
	}
	if err := lib.RecordTrackPlay(ctx, toRecordTracks[0].ID, recordTime); err != nil {
		t.Fatalf("failed to record play for track: %s", err)
	}
	if err := lib.SetAlbumRating(ctx, toRecordTracks[0].AlbumID, 3); err != nil {
		t.Fatalf("failed to set album rating: %s", err)
	}

	secAlbum, ok := expected[secondAlbumName]
	if !ok {
		t.Fatalf("wrong test, `%s` is missing from expected", secondAlbumName)
	}

	secAlbum.Favourite = recordTime.Unix()
	secAlbum.LastPlayed = recordTime.Unix()
	secAlbum.Plays = 1
	secAlbum.Rating = 3
	expected[secondAlbumName] = secAlbum

	var artistID int64
	results := lib.Search(ctx, SearchArgs{Query: artistName})
	for _, track := range results {
		if track.Artist == artistName {
			artistID = track.ArtistID
			break
		}
	}

	if artistID == 0 {
		t.Fatalf("could not find artist `%s`", artistName)
	}

	artistAlbums := lib.GetArtistAlbums(ctx, artistID)
	if len(expected) != len(artistAlbums) {
		t.Errorf("expected %d albums but got %d", len(expected), len(artistAlbums))
	}

	for _, expectedAlbum := range expected {
		var found bool
		for _, album := range artistAlbums {
			if album.Name != expectedAlbum.Name || album.Artist != expectedAlbum.Artist {
				continue
			}
			found = true

			if expectedAlbum.SongCount != album.SongCount {
				t.Errorf("album `%s`: expected %d songs but got %d",
					album.Name,
					expectedAlbum.SongCount,
					album.SongCount,
				)
			}

			if expectedAlbum.Duration != album.Duration {
				t.Errorf("album `%s`: expected %dms duration but got %dms",
					album.Name,
					expectedAlbum.Duration,
					album.Duration,
				)
			}
			assert.Equal(t, expectedAlbum.LastPlayed, album.LastPlayed,
				"album `%s` last played timestamp", album.Name,
			)
			assert.Equal(t, expectedAlbum.Plays, album.Plays,
				"album `%s` plays count", album.Name,
			)
			assert.Equal(t, expectedAlbum.Favourite, album.Favourite,
				"album `%s` favourite timestamp", album.Name,
			)
			assert.Equal(t, expectedAlbum.Rating, album.Rating,
				"album `%s` rating", album.Name,
			)
			assert.Equal(t, expectedAlbum.Year, album.Year,
				"album `%s` year", album.Name,
			)
		}

		if !found {
			t.Errorf("Album `%s` was not found among the artist albums",
				expectedAlbum.Name,
			)
		}
	}
}

// getTestMigrationFiles returns the SQLs directory used by the application itself
// normally. This way tests will be done with the exact same files which will be
// bundled into the binary on build.
func getTestMigrationFiles() fs.FS {
	return os.DirFS("../../sqls")
}
