package subsonic_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ironsmile/euterpe/src/assert"
	"github.com/ironsmile/euterpe/src/config"
	"github.com/ironsmile/euterpe/src/library"
	"github.com/ironsmile/euterpe/src/library/libraryfakes"
	"github.com/ironsmile/euterpe/src/playlists/playlistsfakes"
	"github.com/ironsmile/euterpe/src/radio/radiofakes"
	"github.com/ironsmile/euterpe/src/webserver/subsonic"
	"github.com/ironsmile/euterpe/src/webserver/subsonic/subsonicfakes"
)

// TestGetArtistInfo2 checks that the get artist info returns expected responses.
func TestGetArtistInfo2(t *testing.T) {
	var (
		artistID   int64 = 42
		artistName       = "Some Name"
	)

	ssHandler := subsonic.NewHandler(
		subsonic.Prefix,
		&libraryfakes.FakeLibrary{
			GetArtistStub: func(_ context.Context, id int64) (library.Artist, error) {
				if id != artistID {
					return library.Artist{}, library.ErrArtistNotFound
				}

				return library.Artist{
					ID:   artistID,
					Name: artistName,
				}, nil
			},
		},
		&libraryfakes.FakeBrowser{},
		&radiofakes.FakeStations{},
		&playlistsfakes.FakePlaylister{},
		config.Config{
			Authenticate: config.Auth{
				User: "test-user",
			},
		},
		&subsonicfakes.FakeCoverArtHandler{},
		&subsonicfakes.FakeCoverArtHandler{},
	)

	tests := []struct {
		desc    string
		headers map[string]string
		query   map[string]string

		expectedSmall  string
		expectedMedium string
		expectedLarge  string
	}{
		{
			desc:           "no host header or forwarded",
			expectedSmall:  "http://example.com/rest/getCoverArt?id=ar-42&size=150",
			expectedMedium: "http://example.com/rest/getCoverArt?id=ar-42&size=300",
			expectedLarge:  "http://example.com/rest/getCoverArt?id=ar-42&size=600",
		},
		{
			desc: "forwarded query u",
			query: map[string]string{
				"u": "user",
			},
			expectedSmall:  "http://example.com/rest/getCoverArt?id=ar-42&size=150&u=user",
			expectedMedium: "http://example.com/rest/getCoverArt?id=ar-42&size=300&u=user",
			expectedLarge:  "http://example.com/rest/getCoverArt?id=ar-42&size=600&u=user",
		},
		{
			desc: "many query params",
			query: map[string]string{
				"c": "app",
				"s": "salt",
				"t": "token",
				"p": "pass",
				"v": "1.13.0",
				"u": "user",
			},
			expectedSmall:  "http://example.com/rest/getCoverArt?id=ar-42&size=150&c=app&s=salt&t=token&p=pass&v=1.13.0&u=user",
			expectedMedium: "http://example.com/rest/getCoverArt?id=ar-42&size=300&c=app&s=salt&t=token&p=pass&v=1.13.0&u=user",
			expectedLarge:  "http://example.com/rest/getCoverArt?id=ar-42&size=600&c=app&s=salt&t=token&p=pass&v=1.13.0&u=user",
		},
		{
			desc: "HTTPS with X-Forwarded-Proto",
			headers: map[string]string{
				"X-Forwarded-Proto": "https",
			},

			expectedSmall:  "https://example.com/rest/getCoverArt?id=ar-42&size=150",
			expectedMedium: "https://example.com/rest/getCoverArt?id=ar-42&size=300",
			expectedLarge:  "https://example.com/rest/getCoverArt?id=ar-42&size=600",
		},
		{
			desc: "HTTPS with Forwarded",
			headers: map[string]string{
				"Forwarded": "proto=https",
			},

			expectedSmall:  "https://example.com/rest/getCoverArt?id=ar-42&size=150",
			expectedMedium: "https://example.com/rest/getCoverArt?id=ar-42&size=300",
			expectedLarge:  "https://example.com/rest/getCoverArt?id=ar-42&size=600",
		},
		{
			desc: "different hostname with X-Forwarded-Host",
			headers: map[string]string{
				"X-Forwarded-Host": "music.example.com",
			},

			expectedSmall:  "http://music.example.com/rest/getCoverArt?id=ar-42&size=150",
			expectedMedium: "http://music.example.com/rest/getCoverArt?id=ar-42&size=300",
			expectedLarge:  "http://music.example.com/rest/getCoverArt?id=ar-42&size=600",
		},
		{
			desc: "different hostname with Forwarded",
			headers: map[string]string{
				"Forwarded": `host="music.example.com"`,
			},

			expectedSmall:  "http://music.example.com/rest/getCoverArt?id=ar-42&size=150",
			expectedMedium: "http://music.example.com/rest/getCoverArt?id=ar-42&size=300",
			expectedLarge:  "http://music.example.com/rest/getCoverArt?id=ar-42&size=600",
		},
		{
			desc: "proto and host in Forwarded",
			headers: map[string]string{
				"Forwarded": `host=mu.example.com;proto="https"`,
			},

			expectedSmall:  "https://mu.example.com/rest/getCoverArt?id=ar-42&size=150",
			expectedMedium: "https://mu.example.com/rest/getCoverArt?id=ar-42&size=300",
			expectedLarge:  "https://mu.example.com/rest/getCoverArt?id=ar-42&size=600",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				fmt.Sprintf("/rest/getArtistInfo2?id=%d", int64(1e9+artistID)),
				nil,
			)
			for key, val := range test.headers {
				req.Header.Set(key, val)
			}
			q := req.URL.Query()
			q.Set("f", "json")
			for key, val := range test.query {
				q.Set(key, val)
			}
			req.URL.RawQuery = q.Encode()

			rec := httptest.NewRecorder()

			ssHandler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Result().StatusCode, "HTTP status code")

			t.Logf("getArtistInfo2 response:\n----\n%s\n----\n", rec.Body.String())

			var jsonResp artistInfo2Resp

			dec := json.NewDecoder(rec.Result().Body)
			assert.NilErr(t, dec.Decode(&jsonResp), "error decoding response")
			assert.Equal(t, "ok", jsonResp.Subsonic.Status, "response status")
			assertEqualURLs(t, test.expectedSmall, jsonResp.Subsonic.ArtistInfo2.SmallImageURL,
				"small image URL is different than expected",
			)
			assertEqualURLs(t, test.expectedMedium, jsonResp.Subsonic.ArtistInfo2.MediaumImageURL,
				"small image URL is different than expected",
			)
			assertEqualURLs(t, test.expectedLarge, jsonResp.Subsonic.ArtistInfo2.LargeImageURL,
				"small image URL is different than expected",
			)
		})
	}
}

func assertEqualURLs(t *testing.T, expected, actual string, message string) {
	t.Helper()

	expectedParsed, err := url.Parse(expected)
	assert.NilErr(t, err, "%s: cannot parse expected URL", message)

	actualParsed, err := url.Parse(actual)
	assert.NilErr(t, err, "%s: cannot parse actual URL", message)

	assert.Equal(t, expectedParsed.Scheme, actualParsed.Scheme, "%s: scheme", message)
	assert.Equal(t, expectedParsed.Host, actualParsed.Host, "%s: host", message)
	assert.Equal(t, expectedParsed.Path, actualParsed.Path, "%s: path", message)
	assert.Equal(t, len(expectedParsed.Query()), len(actualParsed.Query()),
		"%s: query len", message,
	)

	for key, expectedParam := range expectedParsed.Query() {
		acutalParam := actualParsed.Query().Get(key)
		assert.Equal(t, expectedParam[0], acutalParam,
			"%s: query param `%s`", message, key,
		)
	}
}

// TestBothGetArtistInfosErrors checks different types of errors returned by the
// the getArtistInfo and getArtistInfo2 handlers.
func TestBothGetArtistInfosErrors(t *testing.T) {
	tests := []struct {
		desc         string
		lib          *libraryfakes.FakeLibrary
		url          string
		expectedCode int
	}{
		{
			desc:         "getArtistInfo: no ID in request",
			url:          "/rest/getArtistInfo?f=json",
			expectedCode: 70,
		},
		{
			desc:         "getArtistInfo2: no ID in request",
			url:          "/rest/getArtistInfo2?f=json",
			expectedCode: 70,
		},
		{
			desc:         "getArtistInfo: malformed ID which cannot be parsed",
			url:          "/rest/getArtistInfo?f=json&id=baba",
			expectedCode: 70,
		},
		{
			desc:         "getArtistInfo2: malformed ID which cannot be parsed",
			url:          "/rest/getArtistInfo2?f=json&id=baba",
			expectedCode: 70,
		},
		{
			desc:         "getArtistInfo: malformed ID not for artist",
			url:          "/rest/getArtistInfo?f=json&id=2",
			expectedCode: 70,
		},
		{
			desc:         "getArtistInfo2: malformed ID not for artist",
			url:          "/rest/getArtistInfo2?f=json&id=2",
			expectedCode: 70,
		},
		{
			desc: "getArtistInfo: artist not found",
			url:  "/rest/getArtistInfo?f=json&id=1000000002",
			lib: &libraryfakes.FakeLibrary{
				GetArtistStub: func(_ context.Context, _ int64) (library.Artist, error) {
					return library.Artist{}, library.ErrArtistNotFound
				},
			},
			expectedCode: 70,
		},
		{
			desc: "getArtistInfo2: artist not found",
			url:  "/rest/getArtistInfo2?f=json&id=1000000002",
			lib: &libraryfakes.FakeLibrary{
				GetArtistStub: func(_ context.Context, _ int64) (library.Artist, error) {
					return library.Artist{}, library.ErrArtistNotFound
				},
			},
			expectedCode: 70,
		},
		{
			desc: "getArtistInfo: library error",
			url:  "/rest/getArtistInfo?f=json&id=1000000002",
			lib: &libraryfakes.FakeLibrary{
				GetArtistStub: func(_ context.Context, _ int64) (library.Artist, error) {
					return library.Artist{}, fmt.Errorf("some error")
				},
			},
			expectedCode: 0,
		},
		{
			desc: "getArtistInfo2: library error",
			url:  "/rest/getArtistInfo2?f=json&id=1000000002",
			lib: &libraryfakes.FakeLibrary{
				GetArtistStub: func(_ context.Context, _ int64) (library.Artist, error) {
					return library.Artist{}, fmt.Errorf("some error")
				},
			},
			expectedCode: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			lib := &libraryfakes.FakeLibrary{}
			if test.lib != nil {
				lib = test.lib
			}

			ssHandler := subsonic.NewHandler(
				subsonic.Prefix,
				lib,
				&libraryfakes.FakeBrowser{},
				&radiofakes.FakeStations{},
				&playlistsfakes.FakePlaylister{},
				config.Config{
					Authenticate: config.Auth{
						User: "test-user",
					},
				},
				&subsonicfakes.FakeCoverArtHandler{},
				&subsonicfakes.FakeCoverArtHandler{},
			)

			req := httptest.NewRequest(http.MethodGet, test.url, nil)
			rec := httptest.NewRecorder()

			ssHandler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Result().StatusCode, "HTTP status code")

			t.Logf("handler response:\n----\n%s\n----\n", rec.Body.String())

			var jsonResp respErrorJSON

			dec := json.NewDecoder(rec.Result().Body)
			assert.NilErr(t, dec.Decode(&jsonResp), "error decoding response")
			assert.Equal(t, "failed", jsonResp.Subsonic.Status, "response status")
			assert.Equal(t, test.expectedCode, jsonResp.Subsonic.Error.Code, "err code")
		})
	}
}

type artistInfo2Resp struct {
	Subsonic struct {
		Status      string `json:"status"`
		ArtistInfo2 struct {
			SmallImageURL   string `json:"smallImageUrl"`
			MediaumImageURL string `json:"mediumImageUrl"`
			LargeImageURL   string `json:"largeImageUrl"`
		} `json:"ArtistInfo2"`
	} `json:"subsonic-response"`
}

type respErrorJSON struct {
	Subsonic struct {
		Status string `json:"status"`
		Error  struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"subsonic-response"`
}
