package subsonic_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// TestGetArtistInfo checks that the get artist info returns expected responses.
func TestGetArtistInfo(t *testing.T) {
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

		expectedSmall  string
		expectedMedium string
		expectedLarge  string
		expectedLastFM string
	}{
		{
			desc:           "no host header or forwarded",
			expectedSmall:  "http://example.com/rest/getCoverArt?id=ar-42&size=150",
			expectedMedium: "http://example.com/rest/getCoverArt?id=ar-42&size=300",
			expectedLarge:  "http://example.com/rest/getCoverArt?id=ar-42&size=600",
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
				fmt.Sprintf("/rest/getArtistInfo?id=%d&f=json", int64(1e9+artistID)),
				nil,
			)
			for key, val := range test.headers {
				req.Header.Set(key, val)
			}

			rec := httptest.NewRecorder()

			ssHandler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Result().StatusCode, "HTTP status code")

			t.Logf("getArtistInfo response:\n----\n%s\n----\n", rec.Body.String())

			var jsonResp artistInfoResp

			dec := json.NewDecoder(rec.Result().Body)
			assert.NilErr(t, dec.Decode(&jsonResp), "error decoding response")
			assert.Equal(t, "ok", jsonResp.Subsonic.Status, "response status")
			assert.Equal(t, test.expectedSmall, jsonResp.Subsonic.ArtistInfo.SmallImageURL,
				"small image URL is different than expected",
			)
			assert.Equal(t, test.expectedMedium, jsonResp.Subsonic.ArtistInfo.MediaumImageURL,
				"small image URL is different than expected",
			)
			assert.Equal(t, test.expectedLarge, jsonResp.Subsonic.ArtistInfo.LargeImageURL,
				"small image URL is different than expected",
			)
		})
	}
}

type artistInfoResp struct {
	Subsonic struct {
		Status     string `json:"status"`
		ArtistInfo struct {
			SmallImageURL   string `json:"smallImageUrl"`
			MediaumImageURL string `json:"mediumImageUrl"`
			LargeImageURL   string `json:"largeImageUrl"`
		} `json:"ArtistInfo"`
	} `json:"subsonic-response"`
}
