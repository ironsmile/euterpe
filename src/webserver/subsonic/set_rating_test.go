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
	"github.com/ironsmile/euterpe/src/library/libraryfakes"
	"github.com/ironsmile/euterpe/src/playlists/playlistsfakes"
	"github.com/ironsmile/euterpe/src/radio/radiofakes"
	"github.com/ironsmile/euterpe/src/webserver/subsonic"
	"github.com/ironsmile/euterpe/src/webserver/subsonic/subsonicfakes"
)

// TestSetRating checks that the /setRating handler is really setting the appropriate
// ratings.
func TestSetRating(t *testing.T) {
	tests := []struct {
		desc     string
		url      string
		checkLib func(t *testing.T, lib *libraryfakes.FakeLibrary)
	}{
		{
			desc: "setting track rating",
			url:  fmt.Sprintf("/rest/setRating?f=json&id=%d&rating=3", int64(2e9+10)),
			checkLib: func(t *testing.T, lib *libraryfakes.FakeLibrary) {
				assert.Equal(t, 1, lib.SetTrackRatingCallCount(), "track rating calls")
				assert.Equal(t, 0, lib.SetAlbumRatingCallCount(), "album rating calls")
				assert.Equal(t, 0, lib.SetArtistRatingCallCount(), "artist rating calls")
				_, argID, argRating := lib.SetTrackRatingArgsForCall(0)
				assert.Equal(t, 10, argID, "track ID")
				assert.Equal(t, 3, argRating, "rating")
			},
		},
		{
			desc: "setting album rating",
			url:  fmt.Sprintf("/rest/setRating?f=json&id=%d&rating=4", int64(12)),
			checkLib: func(t *testing.T, lib *libraryfakes.FakeLibrary) {
				assert.Equal(t, 0, lib.SetTrackRatingCallCount(), "track rating calls")
				assert.Equal(t, 1, lib.SetAlbumRatingCallCount(), "album rating calls")
				assert.Equal(t, 0, lib.SetArtistRatingCallCount(), "artist rating calls")
				_, argID, argRating := lib.SetAlbumRatingArgsForCall(0)
				assert.Equal(t, 12, argID, "album ID")
				assert.Equal(t, 4, argRating, "rating")
			},
		},
		{
			desc: "setting artist rating",
			url:  fmt.Sprintf("/rest/setRating?f=json&id=%d&rating=5", int64(1e9+999)),
			checkLib: func(t *testing.T, lib *libraryfakes.FakeLibrary) {
				assert.Equal(t, 0, lib.SetTrackRatingCallCount(), "track rating calls")
				assert.Equal(t, 0, lib.SetAlbumRatingCallCount(), "album rating calls")
				assert.Equal(t, 1, lib.SetArtistRatingCallCount(), "artist rating calls")
				_, argID, argRating := lib.SetArtistRatingArgsForCall(0)
				assert.Equal(t, 999, argID, "artist ID")
				assert.Equal(t, 5, argRating, "rating")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			lib := &libraryfakes.FakeLibrary{}
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

			var jsonResp ratingRespJSON
			dec := json.NewDecoder(rec.Result().Body)
			assert.NilErr(t, dec.Decode(&jsonResp), "error decoding response")
			assert.Equal(t, "ok", jsonResp.Subsonic.Status)

			test.checkLib(t, lib)
		})
	}
}

// TestSetRatingErrors checks that the /setRating handler returns the appropriate errors.
func TestSetRatingErrors(t *testing.T) {
	tests := []struct {
		desc string
		url  string
		lib  *libraryfakes.FakeLibrary

		checkLib func(t *testing.T, lib *libraryfakes.FakeLibrary)
		errCode  int
	}{
		{
			desc: "lib error while setting track rating",
			url:  fmt.Sprintf("/rest/setRating?f=json&id=%d&rating=3", int64(2e9+10)),
			lib: &libraryfakes.FakeLibrary{
				SetTrackRatingStub: func(_ context.Context, _ int64, _ uint8) error {
					return fmt.Errorf("some error")
				},
			},
			checkLib: func(t *testing.T, lib *libraryfakes.FakeLibrary) {
				assert.Equal(t, 1, lib.SetTrackRatingCallCount(), "track rating calls")
				assert.Equal(t, 0, lib.SetAlbumRatingCallCount(), "album rating calls")
				assert.Equal(t, 0, lib.SetArtistRatingCallCount(), "artist rating calls")
			},
			errCode: 0,
		},
		{
			desc: "lib error while setting album rating",
			url:  fmt.Sprintf("/rest/setRating?f=json&id=%d&rating=4", int64(12)),
			lib: &libraryfakes.FakeLibrary{
				SetAlbumRatingStub: func(ctx context.Context, i int64, u uint8) error {
					return fmt.Errorf("some error")
				},
			},
			checkLib: func(t *testing.T, lib *libraryfakes.FakeLibrary) {
				assert.Equal(t, 0, lib.SetTrackRatingCallCount(), "track rating calls")
				assert.Equal(t, 1, lib.SetAlbumRatingCallCount(), "album rating calls")
				assert.Equal(t, 0, lib.SetArtistRatingCallCount(), "artist rating calls")
			},
			errCode: 0,
		},
		{
			desc: "lib error while setting artist rating",
			url:  fmt.Sprintf("/rest/setRating?f=json&id=%d&rating=5", int64(1e9+999)),
			lib: &libraryfakes.FakeLibrary{
				SetArtistRatingStub: func(ctx context.Context, i int64, u uint8) error {
					return fmt.Errorf("some error")
				},
			},
			checkLib: func(t *testing.T, lib *libraryfakes.FakeLibrary) {
				assert.Equal(t, 0, lib.SetTrackRatingCallCount(), "track rating calls")
				assert.Equal(t, 0, lib.SetAlbumRatingCallCount(), "album rating calls")
				assert.Equal(t, 1, lib.SetArtistRatingCallCount(), "artist rating calls")
			},
			errCode: 0,
		},
		{
			desc:    "no ID parameter",
			url:     "/rest/setRating?f=json&rating=5",
			errCode: 10,
		},
		{
			desc:    "wrong ID parameter type",
			url:     "/rest/setRating?f=json&rating=5&id=baba",
			errCode: 10,
		},
		{
			desc:    "no rating",
			url:     "/rest/setRating?f=json&id=10",
			errCode: 10,
		},
		{
			desc:    "rating wrong type",
			url:     "/rest/setRating?f=json&id=10&rating=baba",
			errCode: 10,
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

			var jsonResp ratingRespJSON
			dec := json.NewDecoder(rec.Result().Body)
			assert.NilErr(t, dec.Decode(&jsonResp), "error decoding response")
			assert.Equal(t, "failed", jsonResp.Subsonic.Status)
			assert.Equal(t, test.errCode, jsonResp.Subsonic.Error.Code, "wrong err code")

			if test.checkLib != nil {
				test.checkLib(t, lib)
			}
		})
	}
}

type ratingRespJSON struct {
	Subsonic struct {
		Status string `json:"status"`
		Error  struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"subsonic-response"`
}
