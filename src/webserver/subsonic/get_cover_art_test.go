package subsonic_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ironsmile/euterpe/src/assert"
	"github.com/ironsmile/euterpe/src/config"
	"github.com/ironsmile/euterpe/src/library/libraryfakes"
	"github.com/ironsmile/euterpe/src/playlists/playlistsfakes"
	"github.com/ironsmile/euterpe/src/radio/radiofakes"
	"github.com/ironsmile/euterpe/src/webserver/subsonic"
	"github.com/ironsmile/euterpe/src/webserver/subsonic/subsonicfakes"
)

// TestGetCoverArt checks that getting cover art for artists, albums and
// tracks works and the correct artwork is recognized by the id string
// format.
func TestGetCoverArt(t *testing.T) {
	const (
		albumArtwork  = `album artwork body`
		artistArtwork = `artist artwork body`
	)

	albumArtFinder := &subsonicfakes.FakeCoverArtHandler{
		FindStub: func(w http.ResponseWriter, _ *http.Request, id int64) error {
			if id != 42 {
				w.WriteHeader(http.StatusNotFound)
				return nil
			}

			fmt.Fprint(w, albumArtwork)
			return nil
		},
	}
	artistArtFinder := &subsonicfakes.FakeCoverArtHandler{
		FindStub: func(w http.ResponseWriter, _ *http.Request, id int64) error {
			if id != 42 {
				w.WriteHeader(http.StatusNotFound)
				return nil
			}

			fmt.Fprint(w, artistArtwork)
			return nil
		},
	}

	ssHandler := subsonic.NewHandler(
		subsonic.Prefix,
		&libraryfakes.FakeLibrary{},
		&libraryfakes.FakeBrowser{},
		&radiofakes.FakeStations{},
		&playlistsfakes.FakePlaylister{},
		config.Config{
			Authenticate: config.Auth{
				User: "test-user",
			},
		},
		albumArtFinder,
		artistArtFinder,
	)

	req := httptest.NewRequest(http.MethodGet, "/rest/getCoverArt?id=al-42", nil)
	rec := httptest.NewRecorder()
	ssHandler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Result().StatusCode, "HTTP status code")
	assert.Equal(t, albumArtwork, rec.Body.String(), "album response body")
	assert.Equal(t, 1, albumArtFinder.FindCallCount(), "wrong number of Find calls")
	_, findReq, findID := albumArtFinder.FindArgsForCall(albumArtFinder.FindCallCount() - 1)
	assert.Equal(t, 42, findID, "wrong album ID send to the art finder")
	assert.Equal(t, "", findReq.URL.Query().Get("size"),
		"size shouldn't have been set for art finder request",
	)

	// Check for handling of clients which wrongly just send the album ID
	// without any prefix.
	req = httptest.NewRequest(http.MethodGet, "/rest/getCoverArt?id=42", nil)
	rec = httptest.NewRecorder()
	ssHandler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Result().StatusCode, "HTTP status code")
	assert.Equal(t, albumArtwork, rec.Body.String(), "album response body")
	assert.Equal(t, 2, albumArtFinder.FindCallCount(), "wrong number of Find calls")

	req = httptest.NewRequest(http.MethodGet, "/rest/getCoverArt?id=ar-42", nil)
	rec = httptest.NewRecorder()
	ssHandler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Result().StatusCode, "HTTP status code")
	assert.Equal(t, artistArtwork, rec.Body.String(), "artist response body")
	assert.Equal(t, 1, artistArtFinder.FindCallCount(), "wrong number of Find calls")

	// Check for handling of clients which wrongly just send the artist ID without any
	// prefix.
	artistArtURL := fmt.Sprintf("/rest/getCoverArt?id=%d", int64(1e9+42))
	req = httptest.NewRequest(http.MethodGet, artistArtURL, nil)
	rec = httptest.NewRecorder()
	ssHandler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Result().StatusCode, "HTTP status code")
	assert.Equal(t, artistArtwork, rec.Body.String(), "artist response body")
	assert.Equal(t, 2, artistArtFinder.FindCallCount(), "wrong number of Find calls")

	// Check the size argument handling.
	req = httptest.NewRequest(http.MethodGet, "/rest/getCoverArt?id=al-42&size=120", nil)
	rec = httptest.NewRecorder()
	ssHandler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Result().StatusCode, "HTTP status code")
	assert.Equal(t, albumArtwork, rec.Body.String(), "album response body")
	assert.Equal(t, 3, albumArtFinder.FindCallCount(), "wrong number of Find calls")
	_, findReq, findID = albumArtFinder.FindArgsForCall(albumArtFinder.FindCallCount() - 1)
	assert.Equal(t, 42, findID, "wrong album ID send to the art finder")
	assert.Equal(t, "small", findReq.URL.Query().Get("size"),
		"wrong size requested from the art finder",
	)

	notFoundTests := []struct {
		desc string
		url  string
	}{
		{
			desc: "playlists have no artwork",
			url:  "/rest/getCoverArt?id=pl-42",
		},
		{
			desc: "tracks have no artwork",
			url:  fmt.Sprintf("/rest/getCoverArt?id=%d", int64(2e9+42)),
		},
		{
			desc: "malformed URLs have no artwork",
			url:  "/rest/getCoverArt?id=baba",
		},
		{
			desc: "album with no artwork in the DB",
			url:  "/rest/getCoverArt?id=al-12",
		},
		{
			desc: "artist with no artwork in the DB",
			url:  "/rest/getCoverArt?id=ar-12",
		},
		{
			desc: "malformed artist URL",
			url:  "/rest/getCoverArt?id=ar-baba",
		},
		{
			desc: "malformed album URL",
			url:  "/rest/getCoverArt?id=al-baba",
		},
	}

	for _, test := range notFoundTests {
		t.Run(test.desc, func(t *testing.T) {
			req = httptest.NewRequest(http.MethodGet, test.url, nil)
			rec = httptest.NewRecorder()
			ssHandler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusNotFound, rec.Result().StatusCode, "HTTP status code")
		})
	}
}
