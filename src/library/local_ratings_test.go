package library

import (
	"context"
	"testing"

	"github.com/ironsmile/euterpe/src/assert"
)

// TestRatingOutOfBounds checks that setting ratings outside of the [0-5] interval causes
// an error.
func TestRatingOutOfBounds(t *testing.T) {
	ctx := context.Background()
	lib := setUpLibForFavRatingsTesting(t)
	defer func() { _ = lib.Truncate() }()

	tracks := lib.Search(ctx, SearchArgs{
		Count: 1,
	})
	if len(tracks) < 1 {
		t.Errorf("expected at least one track to be returned")
	}

	err := lib.SetTrackRating(ctx, tracks[0].ID, 12)
	if err == nil {
		t.Errorf("expected error when setting out of bounds track rating")
	}

	err = lib.SetAlbumRating(ctx, tracks[0].AlbumID, 12)
	if err == nil {
		t.Errorf("expected error when setting out of bounds album rating")
	}

	err = lib.SetArtistRating(ctx, tracks[0].ArtistID, 12)
	if err == nil {
		t.Errorf("expected error when setting out of bounds artist rating")
	}
}

// TestRatingTracks checks that setting and removing ratings for tracks does work.
func TestRatingTracks(t *testing.T) {
	ctx := context.Background()
	lib := setUpLibForFavRatingsTesting(t)
	defer func() { _ = lib.Truncate() }()

	tracks := lib.Search(ctx, SearchArgs{
		Count: 1,
	})
	if len(tracks) < 1 {
		t.Errorf("expected at least one track to be returned")
	}

	track := tracks[0]
	if err := lib.SetTrackRating(ctx, track.ID, 3); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	found, err := lib.GetTrack(ctx, track.ID)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	assert.Equal(t, track.ID, found.ID)
	assert.Equal(t, 3, found.Rating, "wrong track rating")

	if err := lib.SetTrackRating(ctx, track.ID, 0); err != nil {
		t.Errorf("error removing track rating: %s", err)
	}

	found, err = lib.GetTrack(ctx, track.ID)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	assert.Equal(t, track.ID, found.ID)
	assert.Equal(t, 0, found.Rating, "wrong track rating after removal")
}

// TestRatingsAlbums checks that setting and removing ratings from albums work.
func TestRatingsAlbums(t *testing.T) {
	ctx := context.Background()
	lib := setUpLibForFavRatingsTesting(t)
	defer func() { _ = lib.Truncate() }()

	albums := lib.SearchAlbums(ctx, SearchArgs{
		Count: 1,
	})
	if len(albums) < 1 {
		t.Errorf("expected at least one album to be returned")
	}

	album := albums[0]
	if err := lib.SetAlbumRating(ctx, album.ID, 4); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	found, err := lib.GetAlbum(ctx, album.ID)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	assert.Equal(t, album.ID, found.ID)
	assert.Equal(t, 4, found.Rating, "wrong album rating")

	if err := lib.SetAlbumRating(ctx, album.ID, 0); err != nil {
		t.Errorf("error removing album rating: %s", err)
	}

	found, err = lib.GetAlbum(ctx, album.ID)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	assert.Equal(t, album.ID, found.ID)
	assert.Equal(t, 0, found.Rating, "wrong album rating after removal")
}

// TestRatingsArtists checks that setting and removing ratings for artists work.
func TestRatingsArtists(t *testing.T) {
	ctx := context.Background()
	lib := setUpLibForFavRatingsTesting(t)
	defer func() { _ = lib.Truncate() }()

	artists := lib.SearchArtists(ctx, SearchArgs{
		Count: 1,
	})
	if len(artists) < 1 {
		t.Errorf("expected at least one artist to be returned")
	}

	artist := artists[0]
	if err := lib.SetArtistRating(ctx, artist.ID, 4); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	found, err := lib.GetArtist(ctx, artist.ID)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	assert.Equal(t, artist.ID, found.ID)
	assert.Equal(t, 4, found.Rating, "wrong artist rating")

	if err := lib.SetArtistRating(ctx, artist.ID, 0); err != nil {
		t.Errorf("error removing artist rating: %s", err)
	}

	found, err = lib.GetArtist(ctx, artist.ID)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	assert.Equal(t, artist.ID, found.ID)
	assert.Equal(t, 0, found.Rating, "wrong artist rating after removal")
}
