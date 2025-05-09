package playlists_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/ironsmile/euterpe/src/assert"
	"github.com/ironsmile/euterpe/src/library"
	"github.com/ironsmile/euterpe/src/playlists"
)

// TestPlaylistsManager checks that the playlists manager performs all of the
// basic operations it is designed to do.
func TestPlaylistsManager(t *testing.T) {
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
	assertPlaylist(t, expected, playlist)

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
}

// getTestMigrationFiles returns the SQLs directory used by the application itself
// normally. This way tests will be done with the exact same files which will be
// bundled into the binary on build.
func getTestMigrationFiles() fs.FS {
	return os.DirFS("../../sqls")
}

// It is the caller's responsibility to remove the library SQLite database file
func getLibrary(ctx context.Context, t *testing.T) *library.LocalLibrary {
	lib, err := library.NewLocalLibrary(ctx, library.SQLiteMemoryFile, getTestMigrationFiles())
	if err != nil {
		t.Fatal(err.Error())
	}

	err = lib.Initialize()
	if err != nil {
		t.Fatalf("Initializing library: %s", err)
	}

	return lib
}
