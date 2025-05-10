package webserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/ironsmile/euterpe/src/assert"
	"github.com/ironsmile/euterpe/src/library"
	"github.com/ironsmile/euterpe/src/playlists"
	"github.com/ironsmile/euterpe/src/playlists/playlistsfakes"
	"github.com/ironsmile/euterpe/src/webserver"
)

// TestPlaylistHandlerErrors checks how the playlist handler reacts to errors
// returned by the playlist manager.
func TestPlaylistHandlerErrors(t *testing.T) {
	tests := []struct {
		desc         string
		method       string
		url          string
		body         string
		playlistsErr error

		expectedCode int
	}{
		{
			desc:         "malformed playlist ID for get",
			method:       http.MethodGet,
			url:          "/v1/playlist/totally-not-an-int",
			expectedCode: http.StatusNotFound,
		},
		{
			desc:         "malformed playlist ID for patch",
			method:       http.MethodPatch,
			url:          "/v1/playlist/totally-not-an-int",
			expectedCode: http.StatusNotFound,
		},
		{
			desc:         "malformed playlist ID for delete",
			method:       http.MethodDelete,
			url:          "/v1/playlist/totally-not-an-int",
			expectedCode: http.StatusNotFound,
		},
		{
			desc:         "malformed playlist ID for put",
			method:       http.MethodPut,
			url:          "/v1/playlist/totally-not-an-int",
			expectedCode: http.StatusNotFound,
		},
		{
			desc:         "trace method",
			method:       http.MethodTrace,
			url:          "/v1/playlist/5",
			expectedCode: http.StatusMethodNotAllowed,
		},
		{
			desc:         "post method",
			method:       http.MethodPost,
			url:          "/v1/playlist/5",
			expectedCode: http.StatusMethodNotAllowed,
		},
		{
			desc:         "replace playlist which does not exist",
			method:       http.MethodPut,
			url:          "/v1/playlist/5",
			body:         `{"name": "does not matter"}`,
			playlistsErr: playlists.ErrNotFound,
			expectedCode: http.StatusNotFound,
		},
		{
			desc:         "replace playlist which does not exist wrapped",
			method:       http.MethodPut,
			url:          "/v1/playlist/5",
			body:         `{"name": "does not matter"}`,
			playlistsErr: fmt.Errorf("wrapped: %w", playlists.ErrNotFound),
			expectedCode: http.StatusNotFound,
		},
		{
			desc:         "replace playlist general error",
			method:       http.MethodPut,
			url:          "/v1/playlist/5",
			body:         `{"name": "does not matter"}`,
			playlistsErr: fmt.Errorf("some error"),
			expectedCode: http.StatusInternalServerError,
		},
		{
			desc:         "replace playlist wrong request JSON",
			method:       http.MethodPut,
			url:          "/v1/playlist/5",
			body:         `totally not a JSON`,
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "change playlist which does not exist",
			method:       http.MethodPatch,
			url:          "/v1/playlist/5",
			body:         `{"name": "does not matter"}`,
			playlistsErr: playlists.ErrNotFound,
			expectedCode: http.StatusNotFound,
		},
		{
			desc:         "change playlist which does not exist wrapped",
			method:       http.MethodPatch,
			url:          "/v1/playlist/5",
			body:         `{"name": "does not matter"}`,
			playlistsErr: fmt.Errorf("wrapped: %w", playlists.ErrNotFound),
			expectedCode: http.StatusNotFound,
		},
		{
			desc:         "change playlist general error",
			method:       http.MethodPatch,
			url:          "/v1/playlist/5",
			body:         `{"name": "does not matter"}`,
			playlistsErr: fmt.Errorf("some error"),
			expectedCode: http.StatusInternalServerError,
		},
		{
			desc:         "change playlist wrong request JSON",
			method:       http.MethodPatch,
			url:          "/v1/playlist/5",
			body:         `totally not a JSON`,
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "delete playlist which does not exist",
			method:       http.MethodDelete,
			url:          "/v1/playlist/5",
			playlistsErr: playlists.ErrNotFound,
			expectedCode: http.StatusNotFound,
		},
		{
			desc:         "delete playlist which does not exist wrapped",
			method:       http.MethodDelete,
			url:          "/v1/playlist/5",
			playlistsErr: fmt.Errorf("wrapped: %w", playlists.ErrNotFound),
			expectedCode: http.StatusNotFound,
		},
		{
			desc:         "delete playlist general error",
			method:       http.MethodDelete,
			url:          "/v1/playlist/5",
			playlistsErr: fmt.Errorf("some error"),
			expectedCode: http.StatusInternalServerError,
		},
		{
			desc:         "get playlist which does not exist",
			method:       http.MethodGet,
			url:          "/v1/playlist/5",
			playlistsErr: playlists.ErrNotFound,
			expectedCode: http.StatusNotFound,
		},
		{
			desc:         "get playlist which does not exist wrapped",
			method:       http.MethodGet,
			url:          "/v1/playlist/5",
			playlistsErr: fmt.Errorf("wrapped: %w", playlists.ErrNotFound),
			expectedCode: http.StatusNotFound,
		},
		{
			desc:         "get playlist general error",
			method:       http.MethodGet,
			url:          "/v1/playlist/5",
			playlistsErr: fmt.Errorf("some error"),
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			fakeplay := &playlistsfakes.FakePlaylister{}
			fakeplay.CreateReturns(0, test.playlistsErr)
			fakeplay.DeleteReturns(test.playlistsErr)
			fakeplay.GetReturns(playlists.Playlist{}, test.playlistsErr)
			fakeplay.UpdateReturns(test.playlistsErr)

			handler := routePlaylistHandler(
				webserver.NewSinglePlaylistHandler(fakeplay),
			)

			var body io.Reader
			if test.body != "" {
				body = bytes.NewBufferString(test.body)
			}

			req := httptest.NewRequest(test.method, test.url, body)
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			result := resp.Result()
			defer result.Body.Close()

			assert.Equal(t, test.expectedCode, result.StatusCode,
				"HTTP error response mismatch",
			)

			if result.StatusCode == http.StatusMethodNotAllowed {
				// The gorilla mux does not use JSON responses just yet.
				return
			}

			assertJSONContentType(t, result)

			errResp := struct {
				Error string `json:"error"`
			}{}

			dec := json.NewDecoder(result.Body)
			if err := dec.Decode(&errResp); err != nil {
				t.Logf("handler response:\n---\n%s\n---\n", resp.Body)
				t.Fatalf("failed decode JSON response: %s", err)
			}

			if errResp.Error == "" {
				t.Fatalf("the `error` property of the JSON response was not set")
			}
		})
	}
}

// TestPlaylistGettingSingle checks that getting a single playlist works.
func TestPlaylistGettingSingle(t *testing.T) {
	now := time.Now()

	expected := playlists.Playlist{
		ID:          5,
		Name:        "some name",
		Desc:        "some description",
		Public:      true,
		Duration:    5 * time.Second,
		CreatedAt:   now,
		UpdatedAt:   now,
		TracksCount: 2,
		Tracks: []library.TrackInfo{
			{
				ID:          1,
				Title:       "track 1",
				Artist:      "some artist",
				ArtistID:    1,
				Album:       "some album",
				AlbumID:     1,
				TrackNumber: 1,
				Format:      "mp3",
				Duration:    (51 * time.Second).Milliseconds(),
				Plays:       3,
				Favourite:   now.Unix(),
				LastPlayed:  now.Unix(),
				Rating:      3,
				Year:        2014,
				Bitrate:     256 * 1024,
				Size:        67 * 1024,
			},
			{
				ID:       2,
				Title:    "track 2",
				Artist:   "second artist",
				ArtistID: 2,
				Album:    "second album",
				AlbumID:  2,
				Format:   "ogg",
			},
		},
	}

	fakeplay := &playlistsfakes.FakePlaylister{}
	fakeplay.GetReturns(expected, nil)

	handler := routePlaylistHandler(
		webserver.NewSinglePlaylistHandler(fakeplay),
	)

	req := httptest.NewRequest(http.MethodGet, "/v1/playlist/5", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	result := resp.Result()
	assert.Equal(t, http.StatusOK, result.StatusCode, "HTTP response code")
	assertJSONContentType(t, result)

	assert.Equal(t, 1, fakeplay.GetCallCount(), "handler did not request the playlist")
	_, callID := fakeplay.GetArgsForCall(0)
	assert.Equal(t, 5, callID, "getting playlist called with wrong ID")

	var actual apiPlaylist
	dec := json.NewDecoder(result.Body)
	if err := dec.Decode(&actual); err != nil {
		t.Logf("handler response:\n---\n%s\n---\n", resp.Body)
		t.Fatalf("failed to decode playlist response: %s", err)
	}

	assertPlaylist(t, expected, actual)
	for ind, expectedTrack := range expected.Tracks {
		assertTrack(t, expectedTrack, actual.Tracks[ind])
	}
}

// TestPlaylistChange checks that the HTTP methods for changing a playlist work.
func TestPlaylistChange(t *testing.T) {
	tests := []struct {
		desc       string
		httpMethod string
		body       string
		expected   playlists.UpdateArgs
	}{
		{
			desc:       "replacing a playlist",
			httpMethod: http.MethodPut,
			// The "move_indeces" property is a trap. We're
			// making sure it is ignored by the handler.
			body: `{
				"name": "some name",
				"description": "some description",
				"add_tracks_by_id": [2,7,4],
				"move_indeces": [{"from": 1, "to": 2}]
			}`,
			expected: playlists.UpdateArgs{
				Name:            "some name",
				Desc:            "some description",
				AddTracks:       []int64{2, 7, 4},
				RemoveAllTracks: true,
			},
		},
		{
			desc:       "changing a playlist",
			httpMethod: http.MethodPatch,
			// The "move_indeces" property is a trap. We're
			// making sure it is ignored by the handler.
			body: `{
				"name": "some name",
				"description": "some description",
				"add_tracks_by_id": [2,7,4],
				"move_indeces": [{"from": 1, "to": 2}],
				"remove_indeces": [4,2,1]
			}`,
			expected: playlists.UpdateArgs{
				Name:         "some name",
				Desc:         "some description",
				AddTracks:    []int64{2, 7, 4},
				RemoveTracks: []int64{4, 2, 1},
				MoveTracks: []playlists.MoveArgs{
					{FromIndex: 1, ToIndex: 2},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			fakeplay := &playlistsfakes.FakePlaylister{}

			handler := routePlaylistHandler(
				webserver.NewSinglePlaylistHandler(fakeplay),
			)

			expected := test.expected
			body := bytes.NewBufferString(test.body)
			req := httptest.NewRequest(test.httpMethod, "/v1/playlist/123562", body)
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			result := resp.Result()
			assert.Equal(t, http.StatusNoContent, result.StatusCode, "unexpected HTTP response")

			assert.Equal(t, 1, fakeplay.UpdateCallCount(), "unexpected number of Update calls")
			_, actualID, actual := fakeplay.UpdateArgsForCall(0)
			assert.Equal(t, 123562, actualID, "mismatch for updated playlist ID")
			assert.Equal(t, expected.Name, actual.Name, "wrong update name used")
			assert.Equal(t, expected.Desc, actual.Desc, "wrong update description used")
			assert.Equal(t, expected.RemoveAllTracks, actual.RemoveAllTracks, "wrong remove all flag")
			assert.Equal(t, expected.Public, actual.Public, "public status should've been set")
			assert.Equal(t, len(expected.RemoveTracks), len(actual.RemoveTracks), "no tracks should've been removed explicitly")
			assert.Equal(t, len(expected.MoveTracks), len(actual.MoveTracks), "no tracks should've been moved")
			assert.Equal(t, len(expected.AddTracks), len(actual.AddTracks), "added tracks count")

			for ind, trackID := range expected.AddTracks {
				assert.Equal(t, trackID, actual.AddTracks[ind], "track ID at index %d mismatch", ind)
			}

			for ind, trackID := range expected.RemoveTracks {
				assert.Equal(t, trackID, actual.RemoveTracks[ind], "removed track index (at %d) mismatch", ind)
			}

			for ind, move := range expected.MoveTracks {
				assert.Equal(t, move, actual.MoveTracks[ind], "move at index %d mismatch", ind)
			}
		})
	}
}

// TestPlaylistRemoving checks that requests for deleting a playlist are properly
// forwarded to the playlists manager.
func TestPlaylistRemoving(t *testing.T) {
	fakeplay := &playlistsfakes.FakePlaylister{}

	handler := routePlaylistHandler(
		webserver.NewSinglePlaylistHandler(fakeplay),
	)

	req := httptest.NewRequest(http.MethodDelete, "/v1/playlist/5521", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	result := resp.Result()
	assert.Equal(t, http.StatusNoContent, result.StatusCode, "unexpected HTTP response")

	assert.Equal(t, 1, fakeplay.DeleteCallCount(), "unexpected number of Delete calls")
	_, actualID := fakeplay.DeleteArgsForCall(0)
	assert.Equal(t, 5521, actualID, "wrong playlist deleted wow!")
}

func assertJSONContentType(t *testing.T, result *http.Response) {
	t.Helper()

	contentType := result.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected JSON response but it was `%s`", contentType)
	}
}

// routePlaylistHandler wraps a handler the same way the web server will do when
// constructing the main application router. This is needed for tests so that the
// Gorilla mux variables will be parsed.
func routePlaylistHandler(h http.Handler) http.Handler {
	router := mux.NewRouter()
	router.StrictSlash(true)
	router.UseEncodedPath()
	router.Handle(webserver.APIv1EndpointPlaylist, h).Methods(
		webserver.APIv1Methods[webserver.APIv1EndpointPlaylist]...,
	)

	return router
}

type apiPlaylist struct {
	ID          int64               `json:"id"`
	Name        string              `json:"name"`
	Desc        string              `json:"description,omitempty"`
	TracksCount int64               `json:"tracks_count"`
	Duration    int64               `json:"duration"`   // Playlist duration in millisecs.
	CreatedAt   int64               `json:"created_at"` // Unix timestamp in seconds.
	UpdatedAt   int64               `json:"updated_at"` // Unix timestamp in seconds.
	Tracks      []library.TrackInfo `json:"tracks,omitempty"`
}

// assertTrack checks that expected is the same as actual.
func assertTrack(t *testing.T, expected, actual library.TrackInfo) {
	t.Helper()

	assert.Equal(t, expected.ID, actual.ID, "track ID")
	assert.Equal(t, expected.Title, actual.Title, "track title")
	assert.Equal(t, expected.Album, actual.Album, "track album")
	assert.Equal(t, expected.TrackNumber, actual.TrackNumber, "track track number")
	assert.Equal(t, expected.AlbumID, actual.AlbumID, "track album ID")
	assert.Equal(t, expected.Artist, actual.Artist, "track artist")
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

func assertPlaylist(t *testing.T, expected playlists.Playlist, actual apiPlaylist) {
	t.Helper()

	assert.Equal(t, expected.ID, actual.ID, "playlist ID")
	assert.Equal(t, expected.Name, actual.Name, "playlist name")
	assert.Equal(t, expected.Desc, actual.Desc, "playlist description")
	assert.Equal(t, expected.TracksCount, actual.TracksCount, "tracks count")
	assert.Equal(t, expected.Duration.Milliseconds(), actual.Duration, "playlist duration")
	assert.Equal(t, expected.CreatedAt.Unix(), actual.CreatedAt, "created timestamp")
	assert.Equal(t, expected.UpdatedAt.Unix(), actual.UpdatedAt, "updated timestamp")
	assert.Equal(t, len(expected.Tracks), len(actual.Tracks), "tracks slice len mismatch")

}
