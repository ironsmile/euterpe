package webserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/ironsmile/euterpe/src/assert"
	"github.com/ironsmile/euterpe/src/playlists"
	"github.com/ironsmile/euterpe/src/playlists/playlistsfakes"
	"github.com/ironsmile/euterpe/src/webserver"
)

// TestPlaylistsCreation makes an HTTP request for creating a playlist and
// checks that the appropriate calls are made to the playlists manager. Basically,
// it checks for correct HTTP request parsing.
func TestPlaylistsCreation(t *testing.T) {
	const (
		expectedID   = 10
		expectedName = "new playlist"
		expectedDesc = "some description"
	)
	var expectedTracks = []int64{4, 8}

	fakeplay := &playlistsfakes.FakePlaylister{}
	fakeplay.CreateReturns(expectedID, nil)

	handler := routePlaylistsHandler(
		webserver.NewPlaylistsHandler(fakeplay),
	)

	body := bytes.NewBufferString(`{
		"name": "new playlist",
		"description": "some description",
		"add_tracks_by_id": [4, 8]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/playlists", body)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	result := resp.Result()
	assert.Equal(t, http.StatusOK, result.StatusCode, "unexpected HTTP response")
	assertJSONContentType(t, result)

	assert.Equal(t, 1, fakeplay.CreateCallCount(), "unexpected number of Create calls")
	_, actualArgs := fakeplay.CreateArgsForCall(0)
	assert.Equal(t, expectedName, actualArgs.Name, "wrong name during creation")
	assert.Equal(t, expectedDesc, actualArgs.Description, "wrong description")
	assert.Equal(t, len(expectedTracks), len(actualArgs.Tracks), "wrong number of tracks")
	for ind, trackID := range expectedTracks {
		assert.Equal(t, trackID, actualArgs.Tracks[ind], "track at index %s mismatch", ind)
	}

	respJSON := struct {
		ID int64 `json:"created_playlsit_id"`
	}{}

	dec := json.NewDecoder(result.Body)
	if err := dec.Decode(&respJSON); err != nil {
		t.Logf("HTTP response body:\n---\n%s\n---\n", resp.Body)
		t.Fatalf("cannot parse response JSON: %s", err)
	}
	assert.Equal(t, expectedID, respJSON.ID, "wrong ID returned by the HTTP response")
}

// TestPlaylistsHandlerErrors checks how the playlists handler reacts to errors
// returned by the playlist manager.
func TestPlaylistsHandlerErrors(t *testing.T) {
	tests := []struct {
		desc         string
		method       string
		url          string
		body         string
		playlistsErr error
		countErr     error

		expectedCode int
	}{
		{
			desc:         "wrong http status code",
			method:       http.MethodPatch,
			url:          "/v1/playlists",
			expectedCode: http.StatusMethodNotAllowed,
		},
		{
			desc:         "wrong status code - delete",
			method:       http.MethodDelete,
			url:          "/v1/playlists",
			expectedCode: http.StatusMethodNotAllowed,
		},
		{
			desc:         "create playlist general error",
			method:       http.MethodPost,
			url:          "/v1/playlists",
			body:         `{"name": "does not matter"}`,
			playlistsErr: fmt.Errorf("some error"),
			expectedCode: http.StatusInternalServerError,
		},
		{
			desc:         "create wrong playlist JSON",
			method:       http.MethodPost,
			url:          "/v1/playlists",
			body:         `definitely not a JSON`,
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "list wrong page argument",
			method:       http.MethodGet,
			url:          "/v1/playlists?page=baba",
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "list wrong per-page argument",
			method:       http.MethodGet,
			url:          "/v1/playlists?per-page=baba",
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "negative per page",
			method:       http.MethodGet,
			url:          "/v1/playlists?per-page=-10",
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "zero per page",
			method:       http.MethodGet,
			url:          "/v1/playlists?per-page=0",
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "negative page",
			method:       http.MethodGet,
			url:          "/v1/playlists?page=-15",
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "on count error",
			method:       http.MethodGet,
			url:          "/v1/playlists",
			countErr:     fmt.Errorf("some error"),
			expectedCode: http.StatusInternalServerError,
		},
		{
			desc:         "on list error",
			method:       http.MethodGet,
			url:          "/v1/playlists",
			playlistsErr: fmt.Errorf("some error"),
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			fakeplay := &playlistsfakes.FakePlaylister{}
			fakeplay.ListReturns(nil, test.playlistsErr)
			fakeplay.CreateReturns(0, test.playlistsErr)

			if test.countErr != nil {
				fakeplay.CountReturns(0, test.countErr)
			} else {
				fakeplay.CountReturns(2, nil)
			}

			handler := routePlaylistsHandler(
				webserver.NewPlaylistsHandler(fakeplay),
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

// TestPlaylistsListing checks that listing playlists works as expected.
func TestPlaylistsListing(t *testing.T) {
	tests := []struct {
		desc             string
		url              string
		expectedListArgs playlists.ListArgs
		totalInDB        int64
		returnedFromDb   []playlists.Playlist
		expectedResponse apiPlaylistsResponse
	}{
		{
			desc: "middle page",
			url:  "/v1/playlists?page=2&per-page=3",
			expectedListArgs: playlists.ListArgs{
				Offset: 3,
				Count:  3,
			},
			totalInDB: 10,
			returnedFromDb: []playlists.Playlist{
				{
					ID:          5,
					Name:        "some name",
					Desc:        "super description",
					TracksCount: 8,
				},
				{
					ID:          92,
					Name:        "second playlist",
					TracksCount: 4,
				},
			},
			expectedResponse: apiPlaylistsResponse{
				Next:       "/v1/playlists?page=3&per-page=3",
				Previous:   "/v1/playlists?page=1&per-page=3",
				PagesCount: 4,
			},
		},
		{
			desc: "first page",
			url:  "/v1/playlists?per-page=2",
			expectedListArgs: playlists.ListArgs{
				Offset: 0,
				Count:  2,
			},
			totalInDB: 6,
			returnedFromDb: []playlists.Playlist{
				{
					ID:          5,
					Name:        "some name",
					Desc:        "super description",
					TracksCount: 8,
					Duration:    10 * time.Minute,
				},
				{
					ID:          92,
					Name:        "second playlist",
					TracksCount: 4,
				},
			},
			expectedResponse: apiPlaylistsResponse{
				Next:       "/v1/playlists?page=2&per-page=2",
				Previous:   "",
				PagesCount: 3,
			},
		},
		{
			desc: "last page",
			url:  "/v1/playlists?per-page=2&page=3",
			expectedListArgs: playlists.ListArgs{
				Offset: 4,
				Count:  2,
			},
			totalInDB: 6,
			returnedFromDb: []playlists.Playlist{
				{
					ID:          5,
					Name:        "some name",
					Desc:        "super description",
					TracksCount: 8,
					Duration:    10 * time.Minute,
					Public:      false,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				},
				{
					ID:          92,
					Name:        "second playlist",
					TracksCount: 4,
				},
			},
			expectedResponse: apiPlaylistsResponse{
				Next:       "",
				Previous:   "/v1/playlists?page=2&per-page=2",
				PagesCount: 3,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			returned := test.returnedFromDb

			fakeplay := &playlistsfakes.FakePlaylister{}
			fakeplay.ListReturns(returned, nil)
			fakeplay.CountReturns(test.totalInDB, nil)

			handler := routePlaylistsHandler(
				webserver.NewPlaylistsHandler(fakeplay),
			)

			req := httptest.NewRequest(http.MethodGet, test.url, nil)
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			result := resp.Result()
			assert.Equal(t, http.StatusOK, result.StatusCode, "unexpected HTTP response")
			assertJSONContentType(t, result)

			assert.Equal(t, 1, fakeplay.ListCallCount(), "wrong number of list calls")
			assert.Equal(t, 1, fakeplay.CountCallCount(), "wrong number of count calls")

			_, listArgs := fakeplay.ListArgsForCall(0)
			assert.Equal(t, test.expectedListArgs, listArgs, "wrong list arguments")

			expectedResponse := test.expectedResponse

			actualResponse := apiPlaylistsResponse{}
			dec := json.NewDecoder(result.Body)
			if err := dec.Decode(&actualResponse); err != nil {
				t.Logf("HTTP response body:\n---\n%s\n---\n", resp.Body)
				t.Fatalf("cannot parse response JSON: %s", err)
			}

			assert.Equal(t, expectedResponse.Next, actualResponse.Next, "next URI")
			assert.Equal(t, expectedResponse.Previous, actualResponse.Previous, "prev URI")
			assert.Equal(t, expectedResponse.PagesCount, actualResponse.PagesCount, "pages count")
			assert.Equal(t, len(returned), len(actualResponse.Playlists),
				"playlists count mismatch")

			for ind, expected := range returned {
				assertPlaylist(t, expected, actualResponse.Playlists[ind])
				assert.Equal(t, 0, len(actualResponse.Playlists[ind].Tracks),
					"playlist lists should not include the tracks as well",
				)
			}
		})
	}
}

// routePlaylistsHandler wraps a handler the same way the web server will do when
// constructing the main application router. This is needed for tests so that the
// Gorilla mux variables will be parsed.
func routePlaylistsHandler(h http.Handler) http.Handler {
	router := mux.NewRouter()
	router.StrictSlash(true)
	router.UseEncodedPath()
	router.Handle(webserver.APIv1EndpointPlaylists, h).Methods(
		webserver.APIv1Methods[webserver.APIv1EndpointPlaylists]...,
	)

	return router
}

type apiPlaylistsResponse struct {
	Playlists  []apiPlaylist `json:"playlists"`
	Next       string        `json:"next"`
	Previous   string        `json:"previous"`
	PagesCount int           `json:"pages_count"`
}
