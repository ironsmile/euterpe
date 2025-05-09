package playlists_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ironsmile/euterpe/src/assert"
	"github.com/ironsmile/euterpe/src/helpers"
	"github.com/ironsmile/euterpe/src/library"
	"github.com/ironsmile/euterpe/src/playlists"
)

// TestPlaylistsManagerCRUD checks that the playlists manager performs all of the
// basic operations related to creating, updating and removing playlists.
func TestPlaylistsManagerCRUD(t *testing.T) {
	ctx := t.Context()

	lib := getLibrary(ctx, t)
	defer func() {
		_ = lib.Truncate()
	}()
	manager := playlists.NewManager(lib.ExecuteDBJobAndWait)

	count, err := manager.Count(ctx)
	assert.NilErr(t, err, "getting playlists count")
	assert.Equal(t, 0, count, "unexpected number of playlists")

	allPlaylists, err := manager.List(ctx, playlists.ListArgs{
		Offset: 0,
		Count:  100,
	})
	assert.NilErr(t, err, "getting all playlists")
	assert.Equal(t, 0, len(allPlaylists), "did not expect to return any playlists")

	const playlistName = "empty playlist"

	now := time.Now()
	id, err := manager.Create(ctx, playlistName, []int64{})
	assert.NilErr(t, err, "creating empty playlist")

	expected := playlists.Playlist{
		Name:      playlistName,
		ID:        id,
		Public:    true,
		CreatedAt: time.Unix(now.Unix(), 0), // seconds precision in the db
		UpdatedAt: time.Unix(now.Unix(), 0), // seconds precision in the db
	}

	playlist, err := manager.Get(ctx, id)
	assert.NilErr(t, err, "getting a single playlist")
	assertPlaylist(t, expected, playlist)

	expected.Name = "new name for empty"
	expected.Desc = "some description"
	expected.Public = false

	flse := false
	now = time.Now()

	err = manager.Update(ctx, playlist.ID, playlists.UpdateArgs{
		Name:   expected.Name,
		Desc:   expected.Desc,
		Public: &flse,
	})
	assert.NilErr(t, err, "while updating a playlist")

	expected.UpdatedAt = time.Unix(now.Unix(), 0)

	// Get it again from the database and assert it has the new values.
	playlist, err = manager.Get(ctx, playlist.ID)
	assert.NilErr(t, err, "getting a single playlist")
	assertPlaylist(t, expected, playlist)

	count, err = manager.Count(ctx)
	assert.NilErr(t, err, "getting playlists count")
	assert.Equal(t, 1, count, "number of playlists in the database")

	allPlaylists, err = manager.List(ctx, playlists.ListArgs{
		Count: 100,
	})
	assert.NilErr(t, err, "getting a list of playlists")
	assert.Equal(t, 1, len(allPlaylists), "wrong number of returned playlists")
	assertPlaylist(t, expected, allPlaylists[0])

	for _, playlist := range allPlaylists {
		assert.Equal(t, 0, len(playlist.Tracks),
			"tracks should have been omitted from lists with playlists",
		)
	}

	err = manager.Delete(ctx, playlist.ID)
	assert.NilErr(t, err, "while deleting a playlist")

	_, err = manager.Get(ctx, playlist.ID)
	assert.NotNilErr(t, err, "expected 'not found' error for deleted playlist")
}

// TestPlaylistsManagerNotFoundErrors makes sure that the playlists manager returns
// not found errors.
func TestPlaylistsManagerNotFoundErrors(t *testing.T) {
	ctx := t.Context()

	lib := getLibrary(ctx, t)
	defer func() {
		_ = lib.Truncate()
	}()
	manager := playlists.NewManager(lib.ExecuteDBJobAndWait)

	_, err := manager.Get(ctx, 123123)
	if !errors.Is(err, playlists.ErrNotFound) {
		t.Fatalf("get: expected 'not found' error but got: %s", err)
	}

	err = manager.Update(ctx, 123123, playlists.UpdateArgs{Name: "baba"})
	if !errors.Is(err, playlists.ErrNotFound) {
		t.Fatalf("update: expected 'not found' error but got: %s", err)
	}

	err = manager.Delete(ctx, 123123123)
	if !errors.Is(err, playlists.ErrNotFound) {
		t.Fatalf("delete: expected 'not found' error but got: %s", err)
	}
}

// TestPlaylistsManagerSongOperations checks that adding, moving and removing
// songs from a playlist work.
func TestPlaylistsManagerSongOperations(t *testing.T) {
	ctx := t.Context()

	lib := getLibrary(ctx, t)
	defer func() {
		_ = lib.Truncate()
	}()

	projRoot, err := helpers.ProjectRoot()
	assert.NilErr(t, err, "getting the repository root directory")

	lib.AddLibraryPath(filepath.Join(projRoot, "test_files", "library"))

	ch := make(chan struct{})
	go func() {
		lib.Scan()
		close(ch)
	}()

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out after waiting for library scan to complete")
	}

	manager := playlists.NewManager(lib.ExecuteDBJobAndWait)
	allTracks := lib.Search(ctx, library.SearchArgs{Query: "", Count: 100})
	trackIDs := make([]int64, 0, len(allTracks))
	for _, track := range allTracks {
		trackIDs = append(trackIDs, track.ID)
	}

	if len(trackIDs) < 3 {
		t.Fatalf("not enough tracks found in the library for working with playlists")
	}

	playlistID, err := manager.Create(ctx, "Testing Playlist", trackIDs)
	assert.NilErr(t, err, "failed while creating a playlist with all tracks")

	playlist, err := manager.Get(ctx, playlistID)
	assert.NilErr(t, err, "failed while getting newly inserted playlist")
	assertTracks(t, trackIDs, playlist)

	// Removing all tracks and make sure they are gone.
	err = manager.Update(ctx, playlist.ID, playlists.UpdateArgs{
		RemoveAllTracks: true,
	})
	assert.NilErr(t, err, "removing all tracks from a playlist")

	playlist, err = manager.Get(ctx, playlistID)
	assert.NilErr(t, err, "getting playlist after tracks removal")
	assert.Equal(t, 0, len(playlist.Tracks), "wrong number of tracks after removal")
	assert.Equal(t, 0, playlist.TracksCount, "inconsistent .TracksCount")

	// Add the tracks again.
	err = manager.Update(ctx, playlist.ID, playlists.UpdateArgs{
		AddTracks: trackIDs,
	})
	assert.NilErr(t, err, "adding tracks with .Update()")

	playlist, err = manager.Get(ctx, playlistID)
	assert.NilErr(t, err, "failed while getting newly updated playlist")
	assertTracks(t, trackIDs, playlist)

	// Remove tracks from the playlist.

	// currentTracks is a list of trackIDs which will represent the internal
	// state of the playlist in the database from now on.
	currentTracks := append([]int64{}, trackIDs[:1]...)
	currentTracks = append(currentTracks, trackIDs[2:]...)

	err = manager.Update(ctx, playlistID, playlists.UpdateArgs{
		RemoveTracks: []int64{1},
	})
	assert.NilErr(t, err, "removing a single track from the library")
	playlist, err = manager.Get(ctx, playlistID)
	assert.NilErr(t, err, "failed while getting playlist after removing tracks")
	assertTracks(t, currentTracks, playlist)

	// Move tracks around.
	currentTracks[0], currentTracks[1] = currentTracks[1], currentTracks[0]
	err = manager.Update(ctx, playlistID, playlists.UpdateArgs{
		MoveTracks: []playlists.MoveArgs{
			{
				FromIndex: 0,
				ToIndex:   1,
			},
		},
	})
	assert.NilErr(t, err, "while moving tracks around")
	playlist, err = manager.Get(ctx, playlistID)
	assert.NilErr(t, err, "failed while getting playlist after moving tracks")
	assertTracks(t, currentTracks, playlist)

	// Make sure empty update operation is a no-opt.
	err = manager.Update(ctx, playlistID, playlists.UpdateArgs{})
	assert.NilErr(t, err, "while doing a no-opt")
	playlist, err = manager.Get(ctx, playlistID)
	assert.NilErr(t, err, "failed while getting playlist after no-opt update")
	assertTracks(t, currentTracks, playlist)

	// Make sure moving to the same index is a no-opt.
	err = manager.Update(ctx, playlistID, playlists.UpdateArgs{
		MoveTracks: []playlists.MoveArgs{
			{
				FromIndex: 1,
				ToIndex:   1,
			},
		},
	})
	assert.NilErr(t, err, "while doing moving from index to the same index")
	playlist, err = manager.Get(ctx, playlistID)
	assert.NilErr(t, err, "failed while getting playlist after moving tracks")
	assertTracks(t, currentTracks, playlist)

	// Try appending tracks
	newTracks := []int64{currentTracks[0], trackIDs[1]}
	currentTracks = append(currentTracks, newTracks...)
	err = manager.Update(ctx, playlistID, playlists.UpdateArgs{
		AddTracks: newTracks,
	})
	assert.NilErr(t, err, "while appending a track to the end of the list")

	playlist, err = manager.Get(ctx, playlistID)
	assert.NilErr(t, err, "failed while getting playlist after moving tracks")
	assertTracks(t, currentTracks, playlist)
}

// TestPlaylistsManagerErrors checks for some expected errors when dealing with
// playlists.
func TestPlaylistsManagerErrors(t *testing.T) {
	ctx := t.Context()

	lib := getLibrary(ctx, t)
	defer func() {
		_ = lib.Truncate()
	}()
	manager := playlists.NewManager(lib.ExecuteDBJobAndWait)

	_, err := manager.Create(ctx, "", []int64{})
	assert.NotNilErr(t, err,
		"creating a playlist with empty name should have been an error",
	)
}

func assertTracks(t *testing.T, expectedIDs []int64, playlist playlists.Playlist) {
	t.Helper()

	assert.Equal(t, len(expectedIDs), len(playlist.Tracks),
		"playlist had different number of tracks than expected",
	)
	assert.Equal(t, int64(len(playlist.Tracks)), playlist.TracksCount,
		"mismatch between track count in .Tracks slice and .TracksCount",
	)

	for ind, trackID := range expectedIDs {
		assert.Equal(t, trackID, playlist.Tracks[ind].ID,
			"mismatch for track at index %d", ind,
		)
	}
}

func assertPlaylist(t *testing.T, expected, actual playlists.Playlist) {
	t.Helper()

	assert.Equal(t, expected.Name, actual.Name, "wrong newly created playlist name")
	assert.Equal(t, expected.ID, actual.ID, "wrong playlist ID returned")
	assert.Equal(t, expected.Desc, actual.Desc, "playlist description was not empty")
	assert.Equal(t, len(expected.Tracks), len(actual.Tracks), "playlist was expected to be empty")
	assert.Equal(t, expected.TracksCount, actual.TracksCount, "wrong playlist tracks count")
	assert.Equal(t, expected.CreatedAt, actual.CreatedAt, "wrong created at")
	assert.Equal(t, expected.UpdatedAt, actual.UpdatedAt, "wrong updated at")
	assert.Equal(t, expected.Public, actual.Public, "wrong public flag")
	assert.Equal(t, expected.Duration, actual.Duration, "wrong playlist duration")
}

// getTestMigrationFiles returns the SQLs directory used by the application itself
// normally. This way tests will be done with the exact same files which will be
// bundled into the binary on build.
func getTestMigrationFiles() fs.FS {
	return os.DirFS("../../sqls")
}

// It is the caller's responsibility to remove the library SQLite database file
func getLibrary(ctx context.Context, t *testing.T) *library.LocalLibrary {
	migrationsFS := getTestMigrationFiles()
	lib, err := library.NewLocalLibrary(ctx, library.SQLiteMemoryFile, migrationsFS)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = lib.Initialize()
	if err != nil {
		t.Fatalf("Initializing library: %s", err)
	}

	return lib
}
