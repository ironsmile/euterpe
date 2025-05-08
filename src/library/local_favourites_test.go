package library

import (
	"context"
	"testing"
	"time"

	"github.com/ironsmile/euterpe/src/art/artfakes"
	"github.com/ironsmile/euterpe/src/scaler/scalerfakes"
)

// TestFavouritesTracks checks that adding and removing tracks from the list of
// favourites works.
func TestFavouritesTracks(t *testing.T) {
	ctx := context.Background()
	lib := setUpLibForFavoutites(t)
	defer func() { _ = lib.Truncate() }()

	classicalSearch := SearchArgs{Query: "Classical Bugs", Count: 10}

	foundTracks := lib.Search(ctx, classicalSearch)
	if len(foundTracks) != 2 {
		logTracksFavs(t, foundTracks)
		t.Fatalf("expected two songs to be returned but got %d", len(foundTracks))
	}

	favTracks := Favourites{}
	for _, track := range foundTracks {
		if track.Favourite != 0 {
			t.Errorf("track %d is not supposed to be part of favourites yet", track.ID)
		}

		favTracks.TrackIDs = append(favTracks.TrackIDs, track.ID)
	}
	err := lib.RecordFavourite(ctx, favTracks)
	if err != nil {
		t.Fatalf("recording favourites failed: %s", err)
	}

	foundTracks = lib.Search(ctx, classicalSearch)
	if len(foundTracks) != 2 {
		logTracksFavs(t, foundTracks)
		t.Fatalf("expected two songs to be returned again but got %d", len(foundTracks))
	}

	for _, track := range foundTracks {
		if !recentTimestamp(time.Unix(track.Favourite, 0), 10*time.Second) {
			t.Errorf("track %d favourite timestamp is not set: %d",
				track.ID, track.Favourite,
			)
		}
	}

	found, _ := lib.BrowseTracks(BrowseArgs{
		OrderBy: OrderByFavourites,
		Order:   OrderDesc,
		PerPage: 10,
	})
	if len(found) != 2 {
		t.Errorf("expected 2 tracks when browsing by favourites but got %d", len(found))
	}

	for _, track := range found {
		if track.Album != classicalSearch.Query {
			t.Errorf("expected track from `%s` but got one from `%s`",
				classicalSearch.Query,
				track.Album,
			)
		}

		if !recentTimestamp(time.Unix(track.Favourite, 0), 10*time.Second) {
			t.Errorf("track %d favourite timestamp is not set when browsing: %d",
				track.ID, track.Favourite,
			)
		}
	}

	err = lib.RemoveFavourite(ctx, favTracks)
	if err != nil {
		t.Fatalf("removing favourites failed: %s", err)
	}

	found, _ = lib.BrowseTracks(BrowseArgs{
		OrderBy: OrderByFavourites,
		Order:   OrderDesc,
		PerPage: 10,
	})
	if len(found) != 0 {
		t.Errorf("expected no tracks when browsing by favourites but got %d", len(found))
	}
}

// TestFavouritesAlbums checks that adding albums to favourites work.
func TestFavouritesAlbums(t *testing.T) {
	ctx := context.Background()
	lib := setUpLibForFavoutites(t)
	defer func() { _ = lib.Truncate() }()

	classicalSearch := SearchArgs{Query: "Classical Bugs", Count: 10}

	albums := lib.SearchAlbums(ctx, classicalSearch)
	if len(albums) != 1 {
		logAlbumsFavs(t, albums)
		t.Fatalf("expected one album but got %d", len(albums))
	}

	favAlbums := Favourites{}
	for _, album := range albums {
		if album.Favourite != 0 {
			t.Errorf("did no expect the album %d to be part of favourites", album.ID)
		}

		favAlbums.AlbumIDs = append(favAlbums.AlbumIDs, album.ID)
	}

	err := lib.RecordFavourite(ctx, favAlbums)
	if err != nil {
		t.Fatalf("failed to record the favourite albums: %s", err)
	}

	foundAlbums := lib.SearchAlbums(ctx, classicalSearch)
	if len(foundAlbums) != 1 {
		logAlbumsFavs(t, foundAlbums)
		t.Fatalf("expected one album again but got %d", len(albums))
	}

	for _, album := range foundAlbums {
		if !recentTimestamp(time.Unix(album.Favourite, 0), 10*time.Second) {
			t.Errorf("album %d favourite timestamp is not set: %d",
				album.ID,
				album.Favourite,
			)
		}
	}

	found, _ := lib.BrowseAlbums(BrowseArgs{
		OrderBy: OrderByFavourites,
		Order:   OrderDesc,
		PerPage: 10,
	})
	if len(found) != 1 {
		t.Errorf("expected one album when browsing by favourites but got %d", len(found))
	}

	if found[0].Name != classicalSearch.Query {
		t.Errorf("expected alubm `%s` but got `%s`", classicalSearch.Query, found[0].Name)
	}

	if !recentTimestamp(time.Unix(found[0].Favourite, 0), 10*time.Second) {
		t.Errorf("album %d favourite timestamp is not set when browsing: %d",
			found[0].ID, found[0].Favourite,
		)
	}

	err = lib.RemoveFavourite(ctx, favAlbums)
	if err != nil {
		t.Fatalf("removing favourites failed: %s", err)
	}

	found, _ = lib.BrowseAlbums(BrowseArgs{
		OrderBy: OrderByFavourites,
		Order:   OrderDesc,
		PerPage: 10,
	})
	if len(found) != 0 {
		t.Errorf("expected no albums when browsing by favourites but got %d", len(found))
	}
}

// TestFavouritesArtists checks that adding artists to the favourites works.
func TestFavouritesArtists(t *testing.T) {
	ctx := context.Background()
	lib := setUpLibForFavoutites(t)
	defer func() { _ = lib.Truncate() }()

	stackSearchArgs := SearchArgs{Query: "Stack Overflow", Count: 10}
	artists := lib.SearchArtists(ctx, stackSearchArgs)
	if len(artists) != 1 {
		logArtistsFavs(t, artists)
		t.Fatalf("expected one artist but got %d", len(artists))
	}

	favArtists := Favourites{}
	for _, artist := range artists {
		if artist.Favourite != 0 {
			t.Errorf("did not expect the artist %d tu be part of favourites", artist.ID)
		}
		favArtists.ArtistIDs = append(favArtists.ArtistIDs, artist.ID)
	}

	err := lib.RecordFavourite(ctx, favArtists)
	if err != nil {
		t.Fatalf("error while storing favourite artists: %s", err)
	}

	foundArtists := lib.SearchArtists(ctx, stackSearchArgs)
	if len(foundArtists) != 1 {
		logArtistsFavs(t, foundArtists)
		t.Fatalf("expected one artist again but got %d", len(foundArtists))
	}

	for _, artist := range foundArtists {
		if !recentTimestamp(time.Unix(artist.Favourite, 0), 10*time.Second) {
			t.Errorf("artist %d favourite timestamp is not set: %d",
				artist.ID, artist.Favourite,
			)
		}
	}

	found, _ := lib.BrowseArtists(BrowseArgs{
		OrderBy: OrderByFavourites,
		Order:   OrderDesc,
		PerPage: 10,
	})
	if len(found) != 1 {
		t.Errorf("expected one artist when browsing by favourites but got %d", len(found))
	}

	if found[0].Name != stackSearchArgs.Query {
		t.Errorf("expected artist `%s` but got `%s`", stackSearchArgs.Query, found[0].Name)
	}

	if !recentTimestamp(time.Unix(found[0].Favourite, 0), 10*time.Second) {
		t.Errorf("artist %d favourite timestamp is not set when browsing: %d",
			found[0].ID, found[0].Favourite,
		)
	}

	err = lib.RemoveFavourite(ctx, favArtists)
	if err != nil {
		t.Fatalf("removing favourites failed: %s", err)
	}

	found, _ = lib.BrowseArtists(BrowseArgs{
		OrderBy: OrderByFavourites,
		Order:   OrderDesc,
		PerPage: 10,
	})
	if len(found) != 0 {
		t.Errorf("expected no artists when browsing by favourites but got %d", len(found))
	}
}

func setUpLibForFavoutites(t *testing.T) *LocalLibrary {
	ctx := context.Background()
	lib, err := NewLocalLibrary(ctx, SQLiteMemoryFile, getTestMigrationFiles())
	if err != nil {
		t.Fatal(err.Error())
	}

	if err := lib.Initialize(); err != nil {
		t.Fatalf("Initializing library: %s", err)
	}

	lib.SetArtFinder(&artfakes.FakeFinder{})
	lib.SetScaler(&scalerfakes.FakeScaler{})

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
		{
			track: MockMedia{
				artist: "Stack Overflow",
				album:  "Classical Bugs",
				title:  "To Cast You Or Not",
				track:  1,
				length: 151 * time.Second,
			},
			path: "/media/classical-bugs/track-1.mp3",
		},
		{
			track: MockMedia{
				artist: "Stack Overflow",
				album:  "Classical Bugs",
				title:  "Double Free",
				track:  2,
				length: 984 * time.Second,
			},
			path: "/media/classical-bugs/track-2.mp3",
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
			lib.Truncate()
			t.Fatalf("Adding a media file %s failed: %s", trackData.track.Title(), err)
		}
	}

	return lib
}

func recentTimestamp(timestamp time.Time, tolerance time.Duration) bool {
	now := time.Now()
	return now.Add(tolerance).After(timestamp) && now.Add(-tolerance).Before(timestamp)
}

func logArtistsFavs(t *testing.T, artists []Artist) {
	for _, artist := range artists {
		t.Logf("found artist %s id(%d): fav(%d)", artist.Name, artist.ID, artist.Favourite)
	}
}

func logAlbumsFavs(t *testing.T, albums []Album) {
	for _, album := range albums {
		t.Logf("found album %s id(%d): fav(%d)", album.Name, album.ID, album.Favourite)
	}
}

func logTracksFavs(t *testing.T, tracks []TrackInfo) {
	for _, track := range tracks {
		t.Logf("found track %s id(%d): fav(%d)", track.Title, track.ID, track.Favourite)
	}
}
