package library

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestBrowsingArtists(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
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
			"/media/return-of-the-bugs/track-2.mp3",
		},
		{
			MockMedia{
				artist: "Unit Tests",
				album:  "The Return Of The Bugs",
				title:  "Cyclomatic Complexity",
				track:  4,
				length: 602 * time.Second,
			},
			"/media/return-of-the-bugs/track-2.mp3",
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

}
