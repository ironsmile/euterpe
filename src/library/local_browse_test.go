package library

import (
	"context"
	"testing"
	"time"

	"github.com/ironsmile/euterpe/src/assert"
)

// TestBrowsingArtists adds a bunch of tracks into the database and tries
// browsing them by different criteria.
func TestBrowsingArtists(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getPathedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()

	tracks := []struct {
		track MockMedia
		path  string
	}{
		{
			MockMedia{
				artist: "Buggy Bugoff",
				album:  "The Return Of The Bugs",
				title:  "Payback",
				track:  1,
				length: 340 * time.Second,
			},
			"/media/return-of-the-bugs/track-1.mp3",
		},
		{
			MockMedia{
				artist: "Buggy Bugoff",
				album:  "The Return Of The Bugs",
				title:  "Realization",
				track:  2,
				length: 345 * time.Second,
			},
			"/media/return-of-the-bugs/track-2.mp3",
		},
		{
			MockMedia{
				artist: "Code Review",
				album:  "The Return Of The Bugs",
				title:  "Regression Testing",
				track:  3,
				length: 218 * time.Second,
			},
			"/media/return-of-the-bugs/track-3.mp3",
		},
		{
			MockMedia{
				artist: "Unit Tests",
				album:  "The Return Of The Bugs",
				title:  "Cyclomatic Complexity",
				track:  4,
				length: 602 * time.Second,
			},
			"/media/return-of-the-bugs/track-4.mp3",
		},
		{
			MockMedia{
				artist: "Two By Two",
				album:  "Hands In Blue",
				title:  "They Will Never Stop Coming",
				track:  1,
				length: 244 * time.Second,
			},
			"/media/two-by-two/track-3.mp3",
		},
	}

	allArtistsCount := 4

	for _, trackData := range tracks {
		trackInfo := fileInfo{
			Size:     int64(trackData.track.Length().Seconds()) * 128000,
			FilePath: trackData.path,
			Modified: time.Now(),
		}
		err := lib.insertMediaIntoDatabase(&trackData.track, trackInfo)

		if err != nil {
			t.Fatalf("Adding a media file %s failed: %s", trackData.track.Title(), err)
		}
	}

	artistFilter := lib.SearchArtists(ctx, SearchArgs{Query: "Two By Two", Count: 1})
	if len(artistFilter) != 1 {
		t.Fatalf("Could not find the 'Two By Two' artist.")
	}

	tests := []struct {
		desc     string
		search   BrowseArgs
		expected []string
		total    int
	}{
		{
			desc: "descending order first page",
			search: BrowseArgs{
				Page:    0,
				PerPage: 3,
				Order:   OrderDesc,
				OrderBy: OrderByName,
			},
			expected: []string{"Unit Tests", "Two By Two", "Code Review"},
			total:    allArtistsCount,
		},
		{
			desc: "second page",
			search: BrowseArgs{
				Page:    1,
				PerPage: 3,
				Order:   OrderDesc,
				OrderBy: OrderByName,
			},
			expected: []string{"Buggy Bugoff"},
			total:    allArtistsCount,
		},
		{
			desc: "offset overrides page",
			search: BrowseArgs{
				Page:    1, // making sure this value is ignored
				Offset:  2,
				PerPage: 1,
				Order:   OrderDesc,
				OrderBy: OrderByName,
			},
			expected: []string{"Code Review"},
			total:    allArtistsCount,
		},
		{
			desc: "ascending order",
			search: BrowseArgs{
				Page:    1,
				PerPage: 2,
				Order:   OrderAsc,
				OrderBy: OrderByName,
			},
			expected: []string{"Two By Two", "Unit Tests"},
			total:    allArtistsCount,
		},
		{
			desc: "artist id filter",
			search: BrowseArgs{
				ArtistID: artistFilter[0].ID,
				PerPage:  10,
				Order:    OrderAsc,
				OrderBy:  OrderByName,
			},
			expected: []string{artistFilter[0].Name},
			total:    1,
		},
		{
			desc: "order by ID",
			search: BrowseArgs{
				Offset:  0,
				PerPage: 2,
				Order:   OrderAsc,
				OrderBy: OrderByID,
			},
			expected: []string{"Buggy Bugoff", "Code Review"},
			total:    allArtistsCount,
		},
		{
			desc: "page out of bounds",
			search: BrowseArgs{
				Page:    3,
				PerPage: 2,
				Order:   OrderAsc,
				OrderBy: OrderByName,
			},
			expected: []string{},
			total:    allArtistsCount,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			browseArgs := test.search
			expectedArtists := test.expected

			foundArtists, count := lib.BrowseArtists(browseArgs)

			if count != test.total {
				t.Fatalf("Expected all artists to be %d but found %d with search %+v",
					test.total, count, browseArgs)
			}

			if len(foundArtists) != len(expectedArtists) {
				t.Fatalf("Expected returned artists to be %d but found %d for search %+v",
					len(expectedArtists), len(foundArtists), browseArgs)
			}

			for ind, expectedName := range expectedArtists {
				if foundArtists[ind].Name != expectedName {
					t.Errorf("Expected artist[%d] to be '%s' for search %+v but it was '%s'",
						ind, expectedName, browseArgs, foundArtists[ind].Name)
				}
			}
		})
	}

	// Try random browsing. Here only the number of returned elements is tested since
	// the actual order is not deterministic.
	browseArgs := BrowseArgs{
		Page:    5,
		PerPage: 3,
		OrderBy: OrderByRandom,
	}
	foundArtists, count := lib.BrowseArtists(browseArgs)

	if count != allArtistsCount {
		t.Errorf("Expected all artists to be %d but found %d with search %+v",
			allArtistsCount, count, browseArgs)
	}

	if len(foundArtists) != int(browseArgs.PerPage) {
		t.Errorf("Expected %d artists to be returned but got %d",
			browseArgs.PerPage, len(foundArtists))
	}
}

// TestBrowsingAlbums adds a bunch of tracks into the database and tries
// browsing them by different criteria.
func TestBrowsingAlbums(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getPathedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()

	const neverToBe = "Never To Be"

	tracks := []struct {
		track     MockMedia
		path      string
		plays     int
		favourite bool
	}{
		{
			track: MockMedia{
				artist: "Buggy Bugoff",
				album:  "The Return Of The Bugs",
				title:  "Payback",
				track:  1,
				length: 340 * time.Second,
				year:   2013,
			},
			path: "/media/return-of-the-bugs/track-1.mp3",
		},
		{
			track: MockMedia{
				artist: "Buggy Bugoff",
				album:  "The Return Of The Bugs",
				title:  "Realization",
				track:  2,
				length: 345 * time.Second,
				year:   2013,
			},
			path: "/media/return-of-the-bugs/track-2.mp3",
		},
		{
			track: MockMedia{
				artist: "Code Review",
				album:  "The Return Of The Bugs",
				title:  "Regression Testing",
				track:  3,
				length: 218 * time.Second,
				year:   2013,
			},
			path:  "/media/return-of-the-bugs/track-3.mp3",
			plays: 1,
		},
		{
			track: MockMedia{
				artist: "Unit Tests",
				album:  "The Return Of The Bugs",
				title:  "Cyclomatic Complexity",
				track:  4,
				length: 602 * time.Second,
				year:   2013,
			},
			path: "/media/return-of-the-bugs/track-4.mp3",
		},
		{
			track: MockMedia{
				artist: "Two By Two",
				album:  "Hands In Blue",
				title:  "They Will Never Stop Coming",
				track:  1,
				length: 244 * time.Second,
				year:   2016,
			},
			path: "/media/two-by-two/track-3.mp3",
		},
		{
			track: MockMedia{
				artist:  "Eriney",
				album:   neverToBe,
				title:   "Totally Going To Release It",
				track:   1,
				length:  912 * time.Second,
				year:    2022,
				bitrate: 128,
			},
			path:      "/media/never-to-be/track-1.mp3",
			plays:     5,
			favourite: true,
		},
		{
			track: MockMedia{
				artist:  "Eriney",
				album:   neverToBe,
				title:   "Pinky Promise",
				track:   2,
				length:  211 * time.Second,
				year:    2022,
				bitrate: 320,
			},
			path:      "/media/never-to-be/track-2.mp3",
			plays:     5,
			favourite: true,
		},
		{
			track: MockMedia{
				artist: "Eriney",
				album:  "Definitely Never Happening",
				title:  "No Way",
				track:  1,
				length: 127 * time.Second,
				year:   2017,
			},
			path:  "/media/definitely-never-happening/track-1.mp3",
			plays: 2,
		},
	}

	insertedAlbums := map[string]struct{}{}
	favs := Favourites{}

	for _, trackData := range tracks {
		trackInfo := fileInfo{
			Size:     int64(trackData.track.Length().Seconds()) * 128000,
			FilePath: trackData.path,
			Modified: time.Now(),
		}
		err := lib.insertMediaIntoDatabase(&trackData.track, trackInfo)

		if err != nil {
			t.Fatalf("Adding a media file %s failed: %s", trackData.track.Title(), err)
		}

		insertedAlbums[trackData.track.Album()] = struct{}{}

		inserted := lib.Search(ctx, SearchArgs{
			Query: trackData.track.Title(),
			Count: 1,
		})
		if len(inserted) != 1 {
			t.Fatalf("Could not find the inserted track `%s`", trackData.track.Title())
		}

		for i := 0; i < trackData.plays; i++ {
			playedAt := time.Now().Add(-time.Duration(trackData.plays-i) * 10 * time.Minute)
			err := lib.RecordTrackPlay(ctx, inserted[0].ID, playedAt)
			if err != nil {
				t.Fatalf("Failed to record track playing: %s", err)
			}
		}

		if trackData.favourite {
			favs.TrackIDs = append(favs.TrackIDs, inserted[0].ID)
			favs.AlbumIDs = append(favs.AlbumIDs, inserted[0].AlbumID)
		}
	}

	if err := lib.RecordFavourite(ctx, favs); err != nil {
		t.Errorf("could not store favourites: %s", err)
	}

	allAlbumsCount := len(insertedAlbums)

	artistFilter := lib.SearchArtists(ctx, SearchArgs{Query: "Two By Two", Count: 1})
	if len(artistFilter) != 1 {
		t.Fatalf("Could not find the 'Two By Two' artist.")
	}

	tests := []struct {
		desc     string
		search   BrowseArgs
		expected []string
		total    int
	}{
		{
			desc: "first page ascending order by name",
			search: BrowseArgs{
				Page:    0,
				PerPage: 3,
				Order:   OrderAsc,
				OrderBy: OrderByName,
			},
			expected: []string{"Definitely Never Happening", "Hands In Blue", neverToBe},
			total:    allAlbumsCount,
		},
		{
			desc: "second page ascending by name",
			search: BrowseArgs{
				Page:    1,
				PerPage: 3,
				Order:   OrderAsc,
				OrderBy: OrderByName,
			},
			expected: []string{"The Return Of The Bugs"},
			total:    allAlbumsCount,
		},
		{
			desc: "offset overrides page",
			search: BrowseArgs{
				Page:    1, // making sure this value is ignored
				Offset:  2,
				PerPage: 1,
				Order:   OrderAsc,
				OrderBy: OrderByName,
			},
			expected: []string{neverToBe},
			total:    allAlbumsCount,
		},
		{
			desc: "descending order",
			search: BrowseArgs{
				Page:    1,
				PerPage: 2,
				Order:   OrderDesc,
				OrderBy: OrderByName,
			},
			expected: []string{"Hands In Blue", "Definitely Never Happening"},
			total:    allAlbumsCount,
		},
		{
			desc: "artist.ID filter",
			search: BrowseArgs{
				ArtistID: artistFilter[0].ID,
				Offset:   0,
				PerPage:  20,
			},
			expected: []string{"Hands In Blue"},
			total:    1,
		},
		{
			desc: "page out of bounds",
			search: BrowseArgs{
				Page:    3,
				PerPage: 2,
				Order:   OrderDesc,
				OrderBy: OrderByName,
			},
			expected: []string{},
			total:    allAlbumsCount,
		},
		{
			desc: "year range",
			search: BrowseArgs{
				Offset:   0,
				PerPage:  10,
				Order:    OrderAsc,
				OrderBy:  OrderByName,
				FromYear: ptr(int64(2015)),
				ToYear:   ptr(int64(2019)),
			},
			expected: []string{"Definitely Never Happening", "Hands In Blue"},
			total:    2,
		},
		{
			desc: "year lower bound only",
			search: BrowseArgs{
				Offset:   1,
				PerPage:  2,
				Order:    OrderAsc,
				OrderBy:  OrderByName,
				FromYear: ptr(int64(2015)),
			},
			expected: []string{"Hands In Blue", neverToBe},
			total:    3,
		},
		{
			desc: "year upper bound only",
			search: BrowseArgs{
				Offset:  0,
				PerPage: 100,
				Order:   OrderAsc,
				OrderBy: OrderByName,
				ToYear:  ptr(int64(2015)),
			},
			expected: []string{"The Return Of The Bugs"},
			total:    1,
		},
		{
			desc: "order by id",
			search: BrowseArgs{
				Offset:  0,
				PerPage: 2,
				Order:   OrderAsc,
				OrderBy: OrderByID,
			},
			expected: []string{"The Return Of The Bugs", "Hands In Blue"},
			total:    allAlbumsCount,
		},
		{
			desc: "order by artist name",
			search: BrowseArgs{
				Offset:  0,
				PerPage: 2,
				Order:   OrderAsc,
				OrderBy: OrderByArtistName,
			},
			expected: []string{neverToBe, "Definitely Never Happening"},
			total:    allAlbumsCount,
		},
		{
			desc: "order by year",
			search: BrowseArgs{
				Offset:  2,
				PerPage: 2,
				Order:   OrderAsc,
				OrderBy: OrderByYear,
			},
			expected: []string{"Definitely Never Happening", neverToBe},
			total:    allAlbumsCount,
		},
		{
			desc: "order by frequently played",
			search: BrowseArgs{
				Offset:  0,
				PerPage: 2,
				Order:   OrderDesc,
				OrderBy: OrderByFrequentlyPlayed,
			},
			expected: []string{neverToBe, "Definitely Never Happening"},
			total:    allAlbumsCount,
		},
		{
			desc: "order by favourites",
			search: BrowseArgs{
				Offset:  0,
				PerPage: 10,
				Order:   OrderDesc,
				OrderBy: OrderByFavourites,
			},
			expected: []string{neverToBe},
			total:    1,
		},
		{
			desc: "order by recency",
			search: BrowseArgs{
				Offset:  0,
				PerPage: 10,
				Order:   OrderDesc,
				OrderBy: OrderByRecentlyPlayed,
			},
			expected: []string{
				"Definitely Never Happening", neverToBe,
				"The Return Of The Bugs", "Hands In Blue",
			},
			total: 4,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			browseArgs := test.search
			expectedAlbums := test.expected

			foundAlbums, count := lib.BrowseAlbums(browseArgs)

			if count != test.total {
				t.Fatalf("Expected all albums to be %d but found %d with search %+v",
					test.total, count, browseArgs)
			}

			if len(foundAlbums) != len(expectedAlbums) {
				t.Fatalf("Expected returned albums to be %d but found %d for search %+v",
					len(expectedAlbums), len(foundAlbums), browseArgs)
			}

			for ind, expectedName := range expectedAlbums {
				if foundAlbums[ind].Name != expectedName {
					t.Errorf("Expected album[%d] to be '%s' for search %+v but it was '%s'",
						ind, expectedName, browseArgs, foundAlbums[ind].Name)
				}
			}
		})
	}

	// Try random browsing. Here only the number of returned elements is tested since
	// the actual order is not deterministic.
	browseArgs := BrowseArgs{
		Page:    5,
		PerPage: 3,
		OrderBy: OrderByRandom,
	}
	foundAlbums, count := lib.BrowseAlbums(browseArgs)

	if count != allAlbumsCount {
		t.Errorf("Expected all albums to be %d but found %d with search %+v",
			allAlbumsCount, count, browseArgs)
	}

	if len(foundAlbums) != int(browseArgs.PerPage) {
		t.Errorf("Expected %d albums to be returned but got %d",
			browseArgs.PerPage, len(foundAlbums))
	}

	// Make sure song count, album duration and average bitrate are properly set.
	browseArgs = BrowseArgs{
		Page:    0,
		PerPage: uint(allAlbumsCount),
	}
	allAlbums, _ := lib.BrowseAlbums(browseArgs)
	var notGonnaHappen Album
	for _, found := range allAlbums {
		if found.Name == neverToBe {
			notGonnaHappen = found
			break
		}
	}

	if notGonnaHappen.ID == 0 {
		t.Fatalf("The album `%s` was not found", neverToBe)
	}

	var (
		neverToBeBitrate  uint64
		neverToBeDuration time.Duration
		neverToBeTracks   int
	)
	for _, mockTrack := range tracks {
		if mockTrack.track.album != neverToBe {
			continue
		}
		neverToBeDuration += mockTrack.track.length
		neverToBeBitrate += uint64(mockTrack.track.bitrate) * 1024
		neverToBeTracks++
	}
	neverToBeBitrate = uint64(float64(neverToBeBitrate) / float64(neverToBeTracks))

	if notGonnaHappen.SongCount != int64(neverToBeTracks) {
		t.Errorf("wrong number of tracks. Expected %d but got %d",
			neverToBeTracks,
			notGonnaHappen.SongCount,
		)
	}

	if notGonnaHappen.Duration != neverToBeDuration.Milliseconds() {
		t.Errorf("wrong duration. Expected  %dms but got %dms",
			neverToBeDuration.Milliseconds(),
			notGonnaHappen.Duration,
		)
	}

	if notGonnaHappen.AvgBitrate != neverToBeBitrate {
		t.Errorf("wrong average bitrate. Expected %d but got %d",
			neverToBeBitrate,
			notGonnaHappen.AvgBitrate,
		)
	}
}

// TestBrowsingTracks checks that browsing by songs works for the local library. It
// adds a few songs into the library and then tries different types of queries against
// it through the BrowseTracks method.
func TestBrowsingTracks(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	lib := getPathedLibrary(ctx, t)
	defer func() { _ = lib.Truncate() }()

	tracks := []struct {
		track     MockMedia
		path      string
		plays     int
		favourite bool
	}{
		{
			track: MockMedia{
				artist: "Buggy Bugoff",
				album:  "The Return Of The Bugs",
				title:  "Payback",
				track:  1,
				length: 340 * time.Second,
				year:   2013,
			},
			path: "/media/return-of-the-bugs/track-1.mp3",
		},
		{
			track: MockMedia{
				artist: "Buggy Bugoff",
				album:  "The Return Of The Bugs",
				title:  "Realization",
				track:  2,
				length: 345 * time.Second,
				year:   2013,
			},
			path: "/media/return-of-the-bugs/track-2.mp3",
		},
		{
			track: MockMedia{
				artist: "Code Review",
				album:  "The Return Of The Bugs",
				title:  "Regression Testing",
				track:  3,
				length: 218 * time.Second,
				year:   2013,
			},
			path: "/media/return-of-the-bugs/track-3.mp3",
		},
		{
			track: MockMedia{
				artist: "Unit Tests",
				album:  "The Return Of The Bugs",
				title:  "Cyclomatic Complexity",
				track:  4,
				length: 602 * time.Second,
				year:   2013,
			},
			path: "/media/return-of-the-bugs/track-4.mp3",
		},
		{
			track: MockMedia{
				artist: "Two By Two",
				album:  "Hands In Blue",
				title:  "They Will Never Stop Coming",
				track:  1,
				length: 244 * time.Second,
			},
			path: "/media/two-by-two/track-3.mp3",
		},
		{
			track: MockMedia{
				artist: "Eriney",
				album:  "Never To Be",
				title:  "Totally Going To Release It",
				track:  1,
				length: 912 * time.Second,
				year:   2016,
			},
			path: "/media/never-to-be/track-1.mp3",
		},
		{
			track: MockMedia{
				artist: "Eriney",
				album:  "Never To Be",
				title:  "Pinky Promise",
				track:  2,
				length: 211 * time.Second,
				year:   2016,
			},
			path:      "/media/never-to-be/track-2.mp3",
			plays:     33,
			favourite: true,
		},
		{
			track: MockMedia{
				artist: "Eriney",
				album:  "Definitely Never Happening",
				title:  "No Way",
				track:  1,
				length: 127 * time.Second,
				year:   2018,
			},
			path:      "/media/definitely-never-happening/track-1.mp3",
			plays:     100,
			favourite: true,
		},
	}

	tracksInfo := make(map[string]MockMedia)
	favs := Favourites{}

	for _, trackData := range tracks {
		trackInfo := fileInfo{
			Size:     int64(trackData.track.Length().Seconds()) * 128000,
			FilePath: trackData.path,
			Modified: time.Now(),
		}
		err := lib.insertMediaIntoDatabase(&trackData.track, trackInfo)

		if err != nil {
			t.Fatalf("Adding a media file %s failed: %s", trackData.track.Title(), err)
		}

		tracksInfo[trackData.track.title] = trackData.track

		inserted := lib.Search(ctx, SearchArgs{
			Query: trackData.track.Title(),
			Count: 1,
		})
		if len(inserted) != 1 {
			t.Fatalf("Could not find the inserted track `%s`", trackData.track.Title())
		}

		for i := 0; i < trackData.plays; i++ {
			playedAt := time.Now().Add(-time.Duration(trackData.plays-i) * 10 * time.Minute)
			err := lib.RecordTrackPlay(ctx, inserted[0].ID, playedAt)
			if err != nil {
				t.Fatalf("Failed to record track playing: %s", err)
			}
		}

		if trackData.favourite {
			favs.TrackIDs = append(favs.TrackIDs, inserted[0].ID)
			favs.AlbumIDs = append(favs.AlbumIDs, inserted[0].AlbumID)
		}
	}

	if err := lib.RecordFavourite(ctx, favs); err != nil {
		t.Errorf("Could not store favourites: %s", err)
	}

	allTracksCount := len(tracks)

	artists := lib.SearchArtists(ctx, SearchArgs{Query: "Eriney", Count: 1})
	if len(artists) != 1 {
		t.Fatalf("Cannot find Eriney in inserted artists")
	}
	eriney := artists[0]

	tests := []struct {
		desc     string
		search   BrowseArgs
		expected []string
		total    int
	}{
		{
			desc: "order by name",
			search: BrowseArgs{
				Offset:  3,
				PerPage: 3,
				Order:   OrderAsc,
				OrderBy: OrderByName,
			},
			expected: []string{
				"Pinky Promise",
				"Realization",
				"Regression Testing",
			},
			total: allTracksCount,
		},
		{
			desc: "order by name desc",
			search: BrowseArgs{
				Offset:  0,
				PerPage: 2,
				Order:   OrderDesc,
				OrderBy: OrderByName,
			},
			expected: []string{
				"Totally Going To Release It",
				"They Will Never Stop Coming",
			},
			total: allTracksCount,
		},
		{
			desc: "order by artist name",
			search: BrowseArgs{
				Offset:  0,
				PerPage: 2,
				Order:   OrderAsc,
				OrderBy: OrderByArtistName,
			},
			expected: []string{
				"Payback",
				"Realization",
			},
			total: allTracksCount,
		},
		{
			desc: "default order is id ascending",
			search: BrowseArgs{
				Offset:  0,
				PerPage: 2,
			},
			expected: []string{
				"Payback",
				"Realization",
			},
			total: allTracksCount,
		},
		{
			desc: "filter with artist ID",
			search: BrowseArgs{
				Offset:   1,
				PerPage:  2,
				ArtistID: eriney.ID,
				OrderBy:  OrderByName,
				Order:    OrderAsc,
			},
			expected: []string{
				"Pinky Promise",
				"Totally Going To Release It",
			},
			total: 3,
		},
		{
			desc: "filter year range",
			search: BrowseArgs{
				Offset:   0,
				PerPage:  10,
				OrderBy:  OrderByID,
				Order:    OrderAsc,
				FromYear: ptr(int64(2014)),
				ToYear:   ptr(int64(2017)),
			},
			expected: []string{
				"Totally Going To Release It",
				"Pinky Promise",
			},
			total: 2,
		},
		{
			desc: "order by frequently played",
			search: BrowseArgs{
				Offset:  0,
				PerPage: 2,
				OrderBy: OrderByFrequentlyPlayed,
				Order:   OrderDesc,
			},
			expected: []string{
				"No Way",
				"Pinky Promise",
			},
			total: allTracksCount,
		},
		{
			desc: "order by recenlty played",
			search: BrowseArgs{
				Offset:  0,
				PerPage: 1,
				OrderBy: OrderByRecentlyPlayed,
				Order:   OrderDesc,
			},
			expected: []string{
				"Pinky Promise",
			},
			total: allTracksCount,
		},
		{
			desc: "order by artist name",
			search: BrowseArgs{
				Offset:  0,
				PerPage: 1,
				OrderBy: OrderByArtistName,
				Order:   OrderAsc,
			},
			expected: []string{
				"Payback",
			},
			total: allTracksCount,
		},
		{
			desc: "order by year desc",
			search: BrowseArgs{
				Offset:  0,
				PerPage: 1,
				OrderBy: OrderByYear,
				Order:   OrderDesc,
			},
			expected: []string{
				"No Way",
			},
			total: 7, // one track does not have an year
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			browseArgs := test.search
			expectedTracks := test.expected

			foundTracks, count := lib.BrowseTracks(browseArgs)

			if count != test.total {
				t.Fatalf("Expected all track to be %d but found %d with search %+v",
					test.total, count, browseArgs)
			}

			if len(foundTracks) != len(expectedTracks) {
				t.Fatalf("Expected returned tracks to be %d but found %d for search %+v",
					len(expectedTracks), len(foundTracks), browseArgs)
			}

			for ind, expectedName := range expectedTracks {
				track := foundTracks[ind]

				if track.Title != expectedName {
					t.Errorf("Expected track[%d] to be '%s' for search %+v but it was '%s'",
						ind, expectedName, browseArgs, track.Title)
				}

				trackInfo, ok := tracksInfo[expectedName]
				if !ok {
					t.Fatalf("track info for track '%s' not found", track.Title)
				}

				if trackInfo.album != track.Album {
					t.Errorf("%d: expected album `%s` but got `%s`",
						ind, trackInfo.album, track.Album,
					)
				}

				if trackInfo.artist != track.Artist {
					t.Errorf("%d: expected artist `%s` but got `%s`",
						ind, trackInfo.artist, track.Artist,
					)
				}

				if int64(trackInfo.track) != track.TrackNumber {
					t.Errorf("%d: expected number `%d` but got `%d`",
						ind, trackInfo.track, track.TrackNumber,
					)
				}

				if trackInfo.length.Milliseconds() != track.Duration {
					t.Errorf("%d: expected duration (ms) `%d` but got `%d`",
						ind, trackInfo.length.Milliseconds(), track.Duration,
					)
				}

				assert.Equal(t, int32(trackInfo.Year()), track.Year,
					"%d: wrong track year", ind)
			}
		})
	}

	// Try random browsing. Here only the number of returned elements is tested since
	// the actual order is not deterministic.
	browseArgs := BrowseArgs{
		Page:    5,
		PerPage: 3,
		OrderBy: OrderByRandom,
	}
	foundTracks, count := lib.BrowseTracks(browseArgs)

	if count != allTracksCount {
		t.Errorf("Expected all tracks to be %d but found %d with search %+v",
			allTracksCount, count, browseArgs)
	}

	if len(foundTracks) != int(browseArgs.PerPage) {
		t.Errorf("Expected %d tracks to be returned but got %d",
			browseArgs.PerPage, len(foundTracks))
	}
}

func ptr[T any](v T) *T {
	return &v
}
