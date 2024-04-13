package library

import (
	"context"
	"testing"
	"time"
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
		err := lib.insertMediaIntoDatabase(&trackData.track, trackData.path)

		if err != nil {
			t.Fatalf("Adding a media file %s failed: %s", trackData.track.Title(), err)
		}
	}

	tests := []struct {
		search   BrowseArgs
		expected []string
	}{
		{
			search: BrowseArgs{
				Page:    0,
				PerPage: 3,
				Order:   OrderDesc,
				OrderBy: OrderByName,
			},
			expected: []string{"Unit Tests", "Two By Two", "Code Review"},
		},
		{
			search: BrowseArgs{
				Page:    1,
				PerPage: 3,
				Order:   OrderDesc,
				OrderBy: OrderByName,
			},
			expected: []string{"Buggy Bugoff"},
		},
		{
			search: BrowseArgs{
				Page:    1,
				PerPage: 2,
				Order:   OrderAsc,
				OrderBy: OrderByName,
			},
			expected: []string{"Two By Two", "Unit Tests"},
		},
		{
			search: BrowseArgs{
				Page:    3,
				PerPage: 2,
				Order:   OrderAsc,
				OrderBy: OrderByName,
			},
			expected: []string{},
		},
	}

	for _, test := range tests {
		browseArgs := test.search
		expectedArtists := test.expected

		foundArtists, count := lib.BrowseArtists(browseArgs)

		if count != allArtistsCount {
			t.Fatalf("Expected all artists to be %d but found %d with search %+v",
				allArtistsCount, count, browseArgs)
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
		{
			MockMedia{
				artist: "Eriney",
				album:  "Never To Be",
				title:  "Totally Going To Release It",
				track:  1,
				length: 912 * time.Second,
			},
			"/media/never-to-be/track-1.mp3",
		},
		{
			MockMedia{
				artist: "Eriney",
				album:  "Never To Be",
				title:  "Pinky Promise",
				track:  2,
				length: 211 * time.Second,
			},
			"/media/never-to-be/track-2.mp3",
		},
		{
			MockMedia{
				artist: "Eriney",
				album:  "Definitely Never Happening",
				title:  "No Way",
				track:  1,
				length: 127 * time.Second,
			},
			"/media/definitely-never-happening/track-1.mp3",
		},
	}

	insertedAlbums := map[string]struct{}{}

	for _, trackData := range tracks {
		err := lib.insertMediaIntoDatabase(&trackData.track, trackData.path)

		if err != nil {
			t.Fatalf("Adding a media file %s failed: %s", trackData.track.Title(), err)
		}

		insertedAlbums[trackData.track.Album()] = struct{}{}
	}

	allAlbumsCount := len(insertedAlbums)

	tests := []struct {
		search   BrowseArgs
		expected []string
	}{
		{
			search: BrowseArgs{
				Page:    0,
				PerPage: 3,
				Order:   OrderAsc,
				OrderBy: OrderByName,
			},
			expected: []string{"Definitely Never Happening", "Hands In Blue", "Never To Be"},
		},
		{
			search: BrowseArgs{
				Page:    1,
				PerPage: 3,
				Order:   OrderAsc,
				OrderBy: OrderByName,
			},
			expected: []string{"The Return Of The Bugs"},
		},
		{
			search: BrowseArgs{
				Page:    1,
				PerPage: 2,
				Order:   OrderDesc,
				OrderBy: OrderByName,
			},
			expected: []string{"Hands In Blue", "Definitely Never Happening"},
		},
		{
			search: BrowseArgs{
				Page:    3,
				PerPage: 2,
				Order:   OrderDesc,
				OrderBy: OrderByName,
			},
			expected: []string{},
		},
	}

	for _, test := range tests {
		browseArgs := test.search
		expectedAlbums := test.expected

		foundAlbums, count := lib.BrowseAlbums(browseArgs)

		if count != allAlbumsCount {
			t.Fatalf("Expected all albums to be %d but found %d with search %+v",
				allAlbumsCount, count, browseArgs)
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
}
