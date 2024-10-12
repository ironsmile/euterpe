package subsonic_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ironsmile/euterpe/src/config"
	"github.com/ironsmile/euterpe/src/library/libraryfakes"
	"github.com/ironsmile/euterpe/src/playlists/playlistsfakes"
	"github.com/ironsmile/euterpe/src/radio/radiofakes"
	"github.com/ironsmile/euterpe/src/webserver/subsonic"
)

// TestOpenSubsonicFromPostExtension makes sure that the "fromPost" Open Subsonic
// extension works as expected by sending a POST `/query` request with data encoded in
// the body.
func TestOpenSubsonicFromPostExtension(t *testing.T) {
	const (
		authUser     = "test-user"
		authPassword = "test-password"
	)

	lib := &libraryfakes.FakeLibrary{}
	browser := &libraryfakes.FakeBrowser{}
	stations := &radiofakes.FakeStations{}
	playlister := &playlistsfakes.FakePlaylister{}

	ssHandler := subsonic.NewHandler(
		subsonic.Prefix,
		lib,
		browser,
		stations,
		playlister,
		config.Config{
			Auth: true,
			Authenticate: config.Auth{
				User:     authUser,
				Password: authPassword,
			},
		},
		nil, nil,
	)

	body := url.Values{}
	body.Set("u", authUser)
	body.Set("p", authPassword)
	body.Set("c", "osExtenionsTest")
	body.Set("v", "1.16")

	req := httptest.NewRequest(
		http.MethodPost,
		subsonic.Prefix+"/ping",
		strings.NewReader(body.Encode()),
	)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	ssHandler.ServeHTTP(rec, req)

	if status := rec.Result().StatusCode; status != http.StatusOK {
		response, err := io.ReadAll(rec.Result().Body)
		if err != nil {
			t.Logf("Cannot read response body: %s", err)
		} else {
			t.Logf("HTTP response:\n\n%s", response)
		}
		t.Errorf("Expected status %d but got %d", http.StatusOK, status)
	}
}
