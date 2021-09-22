package webserver_test

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/ironsmile/euterpe/src/library"
	"github.com/ironsmile/euterpe/src/library/libraryfakes"
	"github.com/ironsmile/euterpe/src/webserver"
)

// TestBrowseHandlerArgumentParsing makes sure that the HTTP handler parses its arguments
// correctly and passes them to the underlying browser correctly.
func TestBrowseHandlerArgumentParsing(t *testing.T) {
	tests := []struct {
		desc               string
		url                string
		expectedCode       int
		expectedArtistArgs *library.BrowseArgs
		expectedAlbumArgs  *library.BrowseArgs
	}{
		{
			desc:         "default arguments",
			url:          "/v1/browse",
			expectedCode: http.StatusOK,
			expectedAlbumArgs: &library.BrowseArgs{
				PerPage: 10,
				Page:    0,
				OrderBy: library.OrderByName,
				Order:   library.OrderAsc,
			},
		},
		{
			desc:         "for particular album",
			url:          "/v1/browse?by=album&per-page=5&page=2&order-by=id&order=desc",
			expectedCode: http.StatusOK,
			expectedAlbumArgs: &library.BrowseArgs{
				PerPage: 5,
				Page:    1,
				OrderBy: library.OrderByID,
				Order:   library.OrderDesc,
			},
		},
		{
			desc:         "for particular artist",
			url:          "/v1/browse?by=artist&per-page=5&page=2&order-by=id&order=desc",
			expectedCode: http.StatusOK,
			expectedArtistArgs: &library.BrowseArgs{
				PerPage: 5,
				Page:    1,
				OrderBy: library.OrderByID,
				Order:   library.OrderDesc,
			},
		},
		{
			desc:         "default args for artist",
			url:          "/v1/browse?by=artist",
			expectedCode: http.StatusOK,
			expectedArtistArgs: &library.BrowseArgs{
				PerPage: 10,
				Page:    0,
				OrderBy: library.OrderByName,
				Order:   library.OrderAsc,
			},
		},
		{
			desc:         "unsupported by argument",
			url:          "/v1/browse?by=song",
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "negative per-page",
			url:          "/v1/browse?per-page=-2",
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "per-page which is not a number",
			url:          "/v1/browse?per-page=two",
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "zero per-page",
			url:          "/v1/browse?per-page=0",
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "negative page",
			url:          "/v1/browse?page=-2",
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "zero page",
			url:          "/v1/browse?page=0",
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "page which is not a number",
			url:          "/v1/browse?page=five",
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "unsupported order-by",
			url:          "/v1/browse?order-by=genre",
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "unsupported order",
			url:          "/v1/browse?order=best",
			expectedCode: http.StatusBadRequest,
		},
		{
			desc:         "malformed query",
			url:          "/v1/browse?k;ey=5&baba=2%",
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			fakeBrowser := libraryfakes.FakeBrowser{
				BrowseAlbumsStub: func(
					args library.BrowseArgs,
				) ([]library.Album, int) {
					return nil, 0
				},

				BrowseArtistsStub: func(
					args library.BrowseArgs,
				) ([]library.Artist, int) {
					return nil, 0
				},
			}

			handler := routeBrowseHandler(webserver.NewBrowseHandler(&fakeBrowser))
			req := httptest.NewRequest(http.MethodGet, test.url, nil)
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			if test.expectedCode != resp.Code {
				t.Errorf("expected HTTP code %d but got %d", test.expectedCode, resp.Code)
			}

			if test.expectedAlbumArgs != nil {
				if fakeBrowser.BrowseAlbumsCallCount() != 1 {
					t.Fatalf(
						"browse albums called %d times instead of once",
						fakeBrowser.BrowseAlbumsCallCount(),
					)
				}

				expected := *test.expectedAlbumArgs
				foundArgs := fakeBrowser.BrowseAlbumsArgsForCall(0)
				if foundArgs != expected {
					t.Errorf("expected album args %+v but got %+v", expected, foundArgs)
				}
			}

			if test.expectedArtistArgs != nil {
				if fakeBrowser.BrowseArtistsCallCount() != 1 {
					t.Fatalf(
						"browse artist called %d times instead of once",
						fakeBrowser.BrowseArtistsCallCount(),
					)
				}

				expected := *test.expectedArtistArgs
				foundArgs := fakeBrowser.BrowseArtistsArgsForCall(0)
				if foundArgs != expected {
					t.Errorf("expected artist args %+v but got %+v", expected, foundArgs)
				}
			}

			assertContentTypeJSON(t, resp.Header().Get("Content-Type"))
			assertResponseIsValidJSON(t, resp.Body)
		})
	}
}

// TestBrowseHandlerResponseEncoding checks the returned response from the handler. It
// makes sure the returned JSON is the same as the one advertised in the API docs.
func TestBrowseHandlerResponseEncoding(t *testing.T) {
	fakeBrowser := libraryfakes.FakeBrowser{
		BrowseAlbumsStub: func(
			args library.BrowseArgs,
		) ([]library.Album, int) {
			return []library.Album{
				{
					ID:     10,
					Name:   "Senjutsu",
					Artist: "Iron Maiden",
				},
				{
					ID:     11,
					Name:   "The Man Who Sold the World",
					Artist: "David Bowie",
				},
			}, 10
		},

		BrowseArtistsStub: func(
			args library.BrowseArgs,
		) ([]library.Artist, int) {
			return []library.Artist{
				{
					ID:   101,
					Name: "Iron Maiden",
				},
				{
					ID:   102,
					Name: "David Bowie",
				},
			}, 4
		},
	}

	handler := webserver.NewBrowseHandler(&fakeBrowser)

	// Try album response.
	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/browse?by=album&page=2&per-page=2",
		nil,
	)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	assertContentTypeJSON(t, resp.Header().Get("Content-Type"))
	var decAlbums struct {
		PageCount uint32               `json:"pages_count"`
		Next      string               `json:"next"`
		Previvous string               `json:"previous"`
		Albums    []responseAlbumEntry `json:"data"`
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&decAlbums); err != nil {
		t.Fatalf("decoding album JSON response: %s", err)
	}

	if decAlbums.PageCount != 5 {
		t.Errorf("expected page_count to be 5 but it was %d", decAlbums.PageCount)
	}

	const nextAlbumPage = "/v1/browse?by=album&page=3&per-page=2"
	if decAlbums.Next != nextAlbumPage {
		t.Errorf("expected next to be `%s` but it was `%s`", nextAlbumPage, decAlbums.Next)
	}

	const prevAlbumPage = "/v1/browse?by=album&page=1&per-page=2"
	if decAlbums.Previvous != prevAlbumPage {
		t.Errorf(
			"expected prev to be `%s` but it was `%s`",
			prevAlbumPage,
			decAlbums.Previvous,
		)
	}

	expectedAlbums := []responseAlbumEntry{
		{
			ID:     10,
			Name:   "Senjutsu",
			Artist: "Iron Maiden",
		},
		{
			ID:     11,
			Name:   "The Man Who Sold the World",
			Artist: "David Bowie",
		},
	}

	if len(expectedAlbums) != len(decAlbums.Albums) {
		t.Fatalf(
			"albums response expected `%+v` but got `%+v`",
			expectedAlbums,
			decAlbums.Albums,
		)
	}

	for i, album := range expectedAlbums {
		respAlbum := decAlbums.Albums[i]
		if respAlbum != album {
			t.Errorf(
				"expected album %d to be `%+v` but it was `%+v`",
				i, album, respAlbum,
			)
		}
	}

	// Try artists response.
	req = httptest.NewRequest(
		http.MethodGet,
		"/v1/browse?by=artist&page=2&per-page=2&order=asc&order-by=id",
		nil,
	)
	resp = httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	assertContentTypeJSON(t, resp.Header().Get("Content-Type"))
	var decArtists struct {
		PageCount uint32                `json:"pages_count"`
		Next      string                `json:"next"`
		Previvous string                `json:"previous"`
		Artists   []responseArtistEntry `json:"data"`
	}
	dec = json.NewDecoder(resp.Body)
	if err := dec.Decode(&decArtists); err != nil {
		t.Fatalf("decoding artist JSON response: %s", err)
	}

	if decArtists.PageCount != 2 {
		t.Errorf("expected artist page_count to be 2 but it was %d", decArtists.PageCount)
	}

	if decArtists.Next != "" {
		t.Errorf("expected next to be empty but it was `%s`", decArtists.Next)
	}

	const prevArtistPage = "/v1/browse?by=artist&page=1&per-page=2&order=asc&order-by=id"
	if decArtists.Previvous != prevArtistPage {
		t.Errorf(
			"expected prev to be `%s` but it was `%s`",
			prevArtistPage,
			decArtists.Previvous,
		)
	}

	expectedArtists := []responseArtistEntry{
		{
			ID:   101,
			Name: "Iron Maiden",
		},
		{
			ID:   102,
			Name: "David Bowie",
		},
	}

	if len(expectedArtists) != len(decArtists.Artists) {
		t.Fatalf(
			"artists response expected `%+v` but got `%+v`",
			expectedArtists,
			decArtists.Artists,
		)
	}

	for i, artist := range expectedArtists {
		respArtist := decArtists.Artists[i]
		if respArtist != artist {
			t.Errorf(
				"expected artist %d to be `%+v` but it was `%+v`",
				i, artist, respArtist,
			)
		}
	}
}

type responseArtistEntry struct {
	ID   int64  `json:"artist_id"`
	Name string `json:"artist"`
}

type responseAlbumEntry struct {
	ID     int64  `json:"album_id"`
	Name   string `json:"album"`
	Artist string `json:"artist"`
}

func assertContentTypeJSON(t *testing.T, contentType string) {
	mediatype, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatalf("wrong response content-type (%s), err: %s", contentType, err)
	}

	expectedMediaType := "application/json"
	if mediatype != expectedMediaType {
		t.Errorf(
			"expected `%s` content-type but got `%s`",
			expectedMediaType,
			mediatype,
		)
	}

	expectedCharst := "utf-8"
	foundCharset := ""
	for k, v := range params {
		if k == "charset" {
			foundCharset = v
		}
	}

	if expectedCharst != foundCharset {
		t.Errorf(
			"expected content-type charset `%s` but got `%s`",
			expectedCharst,
			foundCharset,
		)
	}
}

func assertResponseIsValidJSON(t *testing.T, r io.Reader) {
	var decoded map[string]interface{}
	dec := json.NewDecoder(r)
	if err := dec.Decode(&decoded); err != nil {
		t.Errorf("invalid JSON: %s", err)
	}
}

// routeBrowseHandler wraps a handler the same way the web server will do when
// constructing the main application router. This is needed for tests so that the
// Gorilla mux variables will be parsed.
func routeBrowseHandler(h http.Handler) http.Handler {
	router := mux.NewRouter()
	router.StrictSlash(true)
	router.UseEncodedPath()
	router.Handle(webserver.APIv1EndpointBrowse, h).Methods(
		webserver.APIv1Methods[webserver.APIv1EndpointBrowse]...,
	)

	return router
}
