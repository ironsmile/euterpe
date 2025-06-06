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
		expectedSongsArgs  *library.BrowseArgs
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
				Offset:  5,
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
				Offset:  5,
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
				Offset:  0,
				OrderBy: library.OrderByName,
				Order:   library.OrderAsc,
			},
		},
		{
			desc:         "random browsing for artist",
			url:          "/v1/browse?by=artist&order-by=random",
			expectedCode: http.StatusOK,
			expectedArtistArgs: &library.BrowseArgs{
				PerPage: 10,
				Offset:  0,
				OrderBy: library.OrderByRandom,
				Order:   library.OrderAsc,
			},
		},
		{
			desc:         "random browsing for artist",
			url:          "/v1/browse?by=album&order-by=random",
			expectedCode: http.StatusOK,
			expectedAlbumArgs: &library.BrowseArgs{
				PerPage: 10,
				Offset:  0,
				OrderBy: library.OrderByRandom,
				Order:   library.OrderAsc,
			},
		},
		{
			desc:         "default args for tracks",
			url:          "/v1/browse?by=song",
			expectedCode: http.StatusOK,
			expectedSongsArgs: &library.BrowseArgs{
				PerPage: 10,
				Offset:  0,
				OrderBy: library.OrderByID,
				Order:   library.OrderAsc,
			},
		},
		{
			desc:         "all the arguments for tracks",
			url:          "/v1/browse?by=song&per-page=15&page=2&order-by=name&order=desc",
			expectedCode: http.StatusOK,
			expectedSongsArgs: &library.BrowseArgs{
				PerPage: 15,
				Offset:  15,
				OrderBy: library.OrderByName,
				Order:   library.OrderDesc,
			},
		},
		{
			desc:         "tracks - play count",
			url:          "/v1/browse?by=song&per-page=30&order-by=frequency&order=desc",
			expectedCode: http.StatusOK,
			expectedSongsArgs: &library.BrowseArgs{
				PerPage: 30,
				Offset:  0,
				OrderBy: library.OrderByFrequentlyPlayed,
				Order:   library.OrderDesc,
			},
		},
		{
			desc:         "tracks - last played",
			url:          "/v1/browse?by=song&per-page=30&order-by=recency&order=desc",
			expectedCode: http.StatusOK,
			expectedSongsArgs: &library.BrowseArgs{
				PerPage: 30,
				Offset:  0,
				OrderBy: library.OrderByRecentlyPlayed,
				Order:   library.OrderDesc,
			},
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
				t.Logf("HTTP response: %s", resp.Body.String())
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

			if test.expectedSongsArgs != nil {
				if fakeBrowser.BrowseTracksCallCount() != 1 {
					t.Fatalf(
						"browse tracks called %d times instead of once",
						fakeBrowser.BrowseTracksCallCount(),
					)
				}

				expected := *test.expectedSongsArgs
				foundArgs := fakeBrowser.BrowseTracksArgsForCall(0)
				if foundArgs != expected {
					t.Errorf("expected track args %+v but got %+v", expected, foundArgs)
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
	songsResponse := []library.TrackInfo{
		{
			ID:          5,
			ArtistID:    10,
			Artist:      "Iron Maiden",
			AlbumID:     10,
			Album:       "Senjutsu",
			Title:       "The Writing On The Wall",
			TrackNumber: 3,
			Format:      "flac",
			Duration:    (5*60 + 58) * 1000,
			Plays:       846726,
			Favourite:   1234231234,
			LastPlayed:  1234231234,
			Rating:      5,
			Year:        2021,
			Bitrate:     337023,
			Size:        5231235,
		},
		{
			ID:          6,
			ArtistID:    10,
			Artist:      "Iron Maiden",
			AlbumID:     10,
			Album:       "Senjutsu",
			Title:       "Lost In A Lost World",
			TrackNumber: 4,
			Format:      "flac",
			Duration:    (5*60 + 58) * 1000,
		},
	}

	fakeBrowser := libraryfakes.FakeBrowser{
		BrowseAlbumsStub: func(
			args library.BrowseArgs,
		) ([]library.Album, int) {
			return []library.Album{
				{
					ID:         10,
					Name:       "Senjutsu",
					Artist:     "Iron Maiden",
					SongCount:  11,
					Duration:   5000,
					Plays:      999231,
					Favourite:  1715625142,
					LastPlayed: 1715625142,
					Rating:     5,
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
					ID:         101,
					Name:       "Iron Maiden",
					AlbumCount: 5,
				},
				{
					ID:   102,
					Name: "David Bowie",
				},
			}, 4
		},

		BrowseTracksStub: func(
			args library.BrowseArgs,
		) ([]library.TrackInfo, int) {
			return songsResponse, 4
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
			ID:         10,
			Name:       "Senjutsu",
			Artist:     "Iron Maiden",
			SongCount:  11,
			Duration:   5000,
			Plays:      999231,
			Favourite:  1715625142,
			LastPlayed: 1715625142,
			Rating:     5,
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
			ID:         101,
			Name:       "Iron Maiden",
			AlbumCount: 5,
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

	// Try songs response.
	req = httptest.NewRequest(
		http.MethodGet,
		"/v1/browse?by=song&page=2&per-page=2&order=asc&order-by=id",
		nil,
	)
	resp = httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	assertContentTypeJSON(t, resp.Header().Get("Content-Type"))
	var decSongs struct {
		PageCount uint32              `json:"pages_count"`
		Next      string              `json:"next"`
		Previvous string              `json:"previous"`
		Songs     []library.TrackInfo `json:"data"`
	}
	dec = json.NewDecoder(resp.Body)
	if err := dec.Decode(&decSongs); err != nil {
		t.Fatalf("decoding artist JSON response: %s", err)
	}

	if decSongs.PageCount != 2 {
		t.Errorf("expected artist page_count to be 2 but it was %d", decSongs.PageCount)
	}

	if decSongs.Next != "" {
		t.Errorf("expected next to be empty but it was `%s`", decSongs.Next)
	}

	const prevSongsPage = "/v1/browse?by=song&page=1&per-page=2&order=asc&order-by=id"
	if decSongs.Previvous != prevSongsPage {
		t.Errorf(
			"expected prev to be `%s` but it was `%s`",
			prevSongsPage,
			decSongs.Previvous,
		)
	}

	expectedSongs := songsResponse

	if len(expectedSongs) != len(decSongs.Songs) {
		t.Fatalf(
			"songs response expected `%+v` but got `%+v`",
			expectedSongs,
			decSongs.Songs,
		)
	}

	for i, song := range expectedSongs {
		respSong := decSongs.Songs[i]
		if respSong != song {
			t.Errorf(
				"expected song %d to be `%+v` but it was `%+v`",
				i, song, respSong,
			)
		}
	}
}

type responseArtistEntry struct {
	ID         int64  `json:"artist_id"`
	Name       string `json:"artist"`
	AlbumCount int64  `json:"album_count"`
}

type responseAlbumEntry struct {
	ID         int64  `json:"album_id"`
	Name       string `json:"album"`
	Artist     string `json:"artist"`
	SongCount  int64  `json:"track_count"`
	Duration   int64  `json:"duration"`
	Plays      int64  `json:"plays"`
	Favourite  int64  `json:"favourite"`
	LastPlayed int64  `json:"last_played"`
	Rating     uint8  `json:"rating"`
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
