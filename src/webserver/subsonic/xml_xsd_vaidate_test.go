package subsonic_test

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ironsmile/euterpe/src/config"
	"github.com/ironsmile/euterpe/src/library"
	"github.com/ironsmile/euterpe/src/library/libraryfakes"
	"github.com/ironsmile/euterpe/src/webserver/subsonic"
	xsdvalidate "github.com/terminalstatic/go-xsd-validate"
)

// TestSubsonicXMLResponses checks that responses from the Subsonic
// API methods return the XML defined in the Subsonic API XSD.
func TestSubsonicXMLResponses(t *testing.T) {
	libSongs := []library.SearchResult{
		{
			ID:          11,
			ArtistID:    10,
			Artist:      "First Artist",
			AlbumID:     10,
			Album:       "First Album",
			Title:       "First Song",
			TrackNumber: 1,
			Format:      "mp3",
			Duration:    162000,
			Plays:       12,
			LastPlayed:  1714856348,
			Favourite:   1714856348,
			Rating:      3,
		},
		{
			ID:          12,
			ArtistID:    11,
			Artist:      "Second Artist",
			AlbumID:     13,
			Album:       "Second Album",
			Title:       "Third Song",
			TrackNumber: 5,
			Format:      "mp3",
			Duration:    195000,
		},
		{
			ID:          13,
			ArtistID:    11,
			Artist:      "Second Artist",
			AlbumID:     13,
			Album:       "Second Album",
			Title:       "Fourth Song",
			TrackNumber: 6,
			Format:      "mp3",
			Duration:    95000,
		},
	}

	lib := &libraryfakes.FakeLibrary{
		GetArtistAlbumsStub: func(ctx context.Context, i int64) []library.Album {
			return []library.Album{
				{
					ID:         22,
					Name:       "First Album",
					Artist:     "First Artist",
					SongCount:  8,
					Plays:      222,
					LastPlayed: 1714856348,
					Favourite:  1614856248,
					Rating:     3,
				},
				{
					ID:        33,
					Name:      "Second Album",
					Artist:    "First Artist",
					SongCount: 12,
				},
				{
					ID:        44,
					Name:      "Third Album",
					Artist:    "First Artist",
					SongCount: 9,
				},
			}
		},
		GetAlbumFilesStub: func(ctx context.Context, i int64) []library.SearchResult {
			return []library.SearchResult{
				{
					ID:          11,
					ArtistID:    10,
					Artist:      "First Artist",
					AlbumID:     10,
					Album:       "First Album",
					Title:       "First Song",
					TrackNumber: 1,
					Format:      "mp3",
					Duration:    162000,
					Plays:       345,
					LastPlayed:  1714856348,
				},
				{
					ID:          12,
					ArtistID:    10,
					Artist:      "First Artist",
					AlbumID:     10,
					Album:       "First Album",
					Title:       "Second Song",
					TrackNumber: 2,
					Format:      "mp3",
					Duration:    195000,
					Favourite:   1714856348,
					Rating:      3,
				},
			}
		},
		GetTrackStub: func(ctx context.Context, i int64) (library.SearchResult, error) {
			return library.TrackInfo{
				ID:          i,
				ArtistID:    22,
				Artist:      "First Artist",
				AlbumID:     11,
				Album:       "First Album",
				Title:       "Track Title",
				TrackNumber: 5,
				Format:      "mp3",
				Duration:    123000,
				Plays:       333,
				LastPlayed:  1714856348,
				Rating:      3,
				Favourite:   1714856348,
			}, nil
		},
		SearchStub: func(
			ctx context.Context,
			sa library.SearchArgs,
		) []library.SearchResult {
			return libSongs
		},
		SearchAlbumsStub: func(
			ctx context.Context, sa library.SearchArgs,
		) []library.Album {
			return []library.Album{
				{
					ID:         10,
					Name:       "First Album",
					Artist:     "Various Artists",
					SongCount:  5,
					Duration:   42318473,
					Plays:      932,
					LastPlayed: 1714856348,
					Rating:     3,
					Favourite:  1714856348,
				},
			}
		},
		SearchArtistsStub: func(
			ctx context.Context,
			sa library.SearchArgs,
		) []library.Artist {
			return []library.Artist{
				{
					ID:         11,
					Name:       "First Artist",
					AlbumCount: 3,
					Favourite:  1714856348,
					Rating:     4,
				},
			}
		},
	}
	browser := &libraryfakes.FakeBrowser{
		BrowseArtistsStub: func(ba library.BrowseArgs) ([]library.Artist, int) {
			resp := []library.Artist{
				{
					ID:         1,
					Name:       "First Artist",
					AlbumCount: 2,
					Favourite:  1714856348,
					Rating:     2,
				},
				{
					ID:         2,
					Name:       "Second Artist",
					AlbumCount: 1,
				},
				{
					ID:         5223,
					Name:       "Third Artist",
					AlbumCount: 23,
				},
			}

			if ba.Page > 0 || ba.Offset >= uint64(len(resp)) { //nolint: staticcheck
				return nil, 3
			}

			return resp, len(resp)
		},

		BrowseAlbumsStub: func(ba library.BrowseArgs) ([]library.Album, int) {
			resp := []library.Album{
				{
					ID:         1,
					Name:       "First Album",
					Artist:     "First Artist",
					SongCount:  5,
					Plays:      333,
					LastPlayed: 1714856348,
					Favourite:  1714856348,
					Rating:     5,
				},
				{
					ID:        2,
					Name:      "Second Album",
					Artist:    "Second Artist",
					SongCount: 9,
					Favourite: 1614856348,
				},
				{
					ID:        5223,
					Name:      "Third Album",
					Artist:    "First Artist",
					SongCount: 22,
				},
			}

			if ba.Page > 0 || ba.Offset > uint64(len(resp)) { //nolint: staticcheck
				return nil, 3
			}

			return resp, len(resp)
		},

		BrowseTracksStub: func(ba library.BrowseArgs) ([]library.SearchResult, int) {
			if ba.Page > 0 || ba.Offset >= uint64(len(libSongs)) { //nolint: staticcheck
				return nil, len(libSongs)
			}

			return libSongs, len(libSongs)
		},
	}

	err := xsdvalidate.Init()
	if err != nil {
		t.Fatalf("failed to initialize xsdvalidate: %s", err)
	}
	defer xsdvalidate.Cleanup()

	xsdhandler, err := xsdvalidate.NewXsdHandlerUrl(
		xsdFileName,
		xsdvalidate.ParsErrDefault,
	)
	if err != nil {
		t.Fatalf("failed to create XSD handler: %s", err)
	}
	defer xsdhandler.Free()

	ssHandler := subsonic.NewHandler(
		subsonic.Prefix,
		lib,
		browser,
		config.Config{},
		nil, nil,
	)

	testURL := func(format string, args ...any) string {
		return subsonic.Prefix + fmt.Sprintf(format, args...)
	}

	tests := []struct {
		desc string
		url  string
	}{
		{
			desc: "ping",
			url:  testURL("/ping"),
		},
		{
			desc: "getLicense",
			url:  testURL("/getLicense"),
		},
		{
			desc: "getMusicFolders",
			url:  testURL("/getMusicFolders"),
		},
		{
			desc: "getIndexes",
			url:  testURL("/getIndexes"),
		},
		{
			desc: "getMusicDirectory artist",
			url:  testURL("/getMusicDirectory?id=%d", int64(1e9+10)),
		},
		{
			desc: "getMusicDirectory album",
			url:  testURL("/getMusicDirectory?id=%d", 10),
		},
		{
			desc: "getArtist",
			url:  testURL("/getArtist?id=%d", int64(1e9+10)),
		},
		{
			desc: "getAlbum",
			url:  testURL("/getAlbum?id=%d", 10),
		},
		{
			desc: "getAlbumList2",
			url:  testURL("/getAlbumList2?type=random&id=%d", 10),
		},
		{
			desc: "getAlbumList",
			url:  testURL("/getAlbumList?type=random&id=%d", 10),
		},
		{
			desc: "getArtistInfo2",
			url:  testURL("/getArtistInfo2?id=%d", int64(1e9+10)),
		},
		{
			desc: "getArtistInfo",
			url:  testURL("/getArtistInfo?id=%d", int64(1e9+10)),
		},
		{
			desc: "getArtists",
			url:  testURL("/getArtists"),
		},
		{
			desc: "getSong",
			url:  testURL("/getSong?id=%d", int64(2e9+66)),
		},
		{
			desc: "getGenres",
			url:  testURL("/getGenres"),
		},
		{
			desc: "getVideos",
			url:  testURL("/getVideos"),
		},
		{
			desc: "search3",
			url:  testURL("/search3?query=baba"),
		},
		{
			desc: "search2",
			url:  testURL("/search2?query=baba"),
		},
		{
			desc: "search any",
			url:  testURL("/search2?any=baba"),
		},
		{
			desc: "search artits",
			url:  testURL("/search?arist=baba"),
		},
		{
			desc: "search albums",
			url:  testURL("/search?album=baba"),
		},
		{
			desc: "search tracks",
			url:  testURL("/search?title=baba"),
		},
		{
			desc: "scrobble",
			url:  testURL("/scrobble?id=%d&time=1714834066", int64(2e9+33)),
		},
		{
			desc: "star track",
			url:  testURL("/star?id=%d", int64(2e9+33)),
		},
		{
			desc: "star artist",
			url:  testURL("/star?id=%d", int64(1e9+10)),
		},
		{
			desc: "star artist ID3",
			url:  testURL("/star?artistId=%d", int64(1e9+10)),
		},
		{
			desc: "star album",
			url:  testURL("/star?id=%d", int64(10)),
		},
		{
			desc: "star album ID3",
			url:  testURL("/star?albumId=%d", int64(10)),
		},
		{
			desc: "star multiple values",
			url: testURL(
				"/star?id=%d&id=%d&id=%d&artistId=%d&albumId=%d",
				int64(10), int64(1e9+10), int64(2e9+10), int64(1e9+11), int64(10),
			),
		},
		{
			desc: "unstar track",
			url:  testURL("/unstar?id=%d", int64(2e9+33)),
		},
		{
			desc: "unstar artist",
			url:  testURL("/unstar?id=%d", int64(1e9+10)),
		},
		{
			desc: "unstar artist ID3",
			url:  testURL("/unstar?artistId=%d", int64(1e9+10)),
		},
		{
			desc: "unstar album",
			url:  testURL("/unstar?id=%d", int64(10)),
		},
		{
			desc: "unstar album ID3",
			url:  testURL("/unstar?albumId=%d", int64(10)),
		},
		{
			desc: "unstar multiple values",
			url: testURL(
				"/unstar?id=%d&id=%d&id=%d&artistId=%d&albumId=%d",
				int64(10), int64(1e9+10), int64(2e9+10), int64(1e9+11), int64(10),
			),
		},
		{
			desc: "getStarred",
			url:  testURL("/getStarred"),
		},
		{
			desc: "getStarred2",
			url:  testURL("/getStarred2"),
		},
		{
			desc: "getTopSongs",
			url:  testURL("/getTopSongs?artist=First+Artist&count=3"),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				test.url,
				nil,
			)

			rec := httptest.NewRecorder()

			ssHandler.ServeHTTP(rec, req)

			resp := rec.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("HTTP response had code %d", resp.StatusCode)
			}

			respBody := rec.Body.String()

			xmlResp := baseResponse{}
			dec := xml.NewDecoder(bytes.NewBufferString(respBody))
			if err := dec.Decode(&xmlResp); err != nil {
				t.Fatalf("cannot decode XML: %s", err)
			}

			if xmlResp.Status != "ok" {
				t.Logf("XML response: %s\n", respBody)
				t.Fatalf("expected OK response but got `%s`", xmlResp.Status)
			}

			err := xsdhandler.ValidateMem(
				[]byte(respBody),
				xsdvalidate.ValidErrDefault,
			)
			if err != nil {
				switch verr := err.(type) {
				case xsdvalidate.ValidationError:
					var errors int
					for _, xmlErr := range verr.Errors {
						if xmlErr.NodeName == "subsonic-response" &&
							xmlErr.Code == 1866 {
							// Few attributes are added for more
							// information even though they are
							// not part of the specification.
							continue
						}

						t.Logf("Error in line: %d\n", xmlErr.Line)
						t.Log(xmlErr.Message)

						t.Logf("error: %#v\n", xmlErr)

						errors++
					}

					if errors > 0 {
						t.Errorf("XSD validation failed with %d errors", errors)
					}
				default:
					t.Errorf("general XSD validation error: %s", err)
				}
			}

		})
	}
}

// TestSubsonicXMLErrors checks that errors returned from the Subsonic API have the
// correct error code and also have a valid XML.
func TestSubsonicXMLErrors(t *testing.T) {
	lib := &libraryfakes.FakeLibrary{}
	browser := &libraryfakes.FakeBrowser{}

	err := xsdvalidate.Init()
	if err != nil {
		t.Fatalf("failed to initialize xsdvalidate: %s", err)
	}
	defer xsdvalidate.Cleanup()

	xsdhandler, err := xsdvalidate.NewXsdHandlerUrl(
		xsdFileName,
		xsdvalidate.ParsErrDefault,
	)
	if err != nil {
		t.Fatalf("failed to create XSD handler: %s", err)
	}
	defer xsdhandler.Free()

	ssHandler := subsonic.NewHandler(
		subsonic.Prefix,
		lib,
		browser,
		config.Config{},
		nil, nil,
	)

	testURL := func(format string, args ...any) string {
		return subsonic.Prefix + fmt.Sprintf(format, args...)
	}

	tests := []struct {
		desc      string
		url       string
		errorCode int
	}{
		{
			desc:      "getVideoInfo",
			url:       testURL("/getVideoInfo?id=20"),
			errorCode: 70,
		},
		{
			desc:      "no scrobble ID",
			url:       testURL("/scrobble"),
			errorCode: 10,
		},
		{
			desc:      "invalid scrobble ID",
			url:       testURL("/scrobble?id=baba"),
			errorCode: 70,
		},
		{
			desc:      "invalid scrobble time",
			url:       testURL("/scrobble?id=%d&time=baba", int64(2e9+555)),
			errorCode: 0,
		},
		{
			desc:      "getArtistInfo ID for something which is not artist",
			url:       testURL("/getArtistInfo?id=2"),
			errorCode: 70,
		},
		{
			desc:      "invalid getArtistInfo ID",
			url:       testURL("/getArtistInfo?id=baba"),
			errorCode: 70,
		},
		{
			desc:      "no getArtistInfo ID",
			url:       testURL("/getArtistInfo"),
			errorCode: 70,
		},
		{
			desc:      "getArtistInfo2 ID for something which is not artist",
			url:       testURL("/getArtistInfo2?id=2"),
			errorCode: 70,
		},
		{
			desc:      "invalid getArtistInfo2 ID",
			url:       testURL("/getArtistInfo2?id=baba"),
			errorCode: 70,
		},
		{
			desc:      "no getArtistInfo2 ID",
			url:       testURL("/getArtistInfo2"),
			errorCode: 70,
		},
		{
			desc:      "scrobble with no track",
			url:       testURL("/scrobble"),
			errorCode: 10,
		},
		{
			desc:      "scrobble wrong track",
			url:       testURL("/scrobble?id=baba"),
			errorCode: 70,
		},
		{
			desc:      "scrobble track ID which is not for a track",
			url:       testURL("/scrobble?id=10"),
			errorCode: 70,
		},
		{
			desc:      "scrobble wrong Unix time",
			url:       testURL("/scrobble?id=%d&time=baba", int64(2e9+55)),
			errorCode: 0,
		},
		{
			desc:      "scrobble negative Unix time",
			url:       testURL("/scrobble?id=%d&time=-50", int64(2e9+55)),
			errorCode: 0,
		},
		{
			desc:      "star wrong track ID",
			url:       testURL("/star?id=baba"),
			errorCode: 0,
		},
		{
			desc:      "star wrong album ID",
			url:       testURL("/star?albumId=baba"),
			errorCode: 0,
		},
		{
			desc:      "star wrong artist ID",
			url:       testURL("/star?artistId=baba"),
			errorCode: 0,
		},
		{
			desc:      "star wrong ID type - reserved ID",
			url:       testURL("/star?id=%d", int64(1e9)),
			errorCode: 0,
		},
		{
			desc:      "star wrong album ID type - reserved ID",
			url:       testURL("/star?albumId=%d", int64(1e9)),
			errorCode: 0,
		},
		{
			desc:      "star wrong artist ID type - reserved ID",
			url:       testURL("/star?artistId=%d", int64(1e9)),
			errorCode: 0,
		},
		{
			desc:      "star no arguments",
			url:       testURL("/star"),
			errorCode: 10,
		},
		{
			desc:      "unstar wrong track ID",
			url:       testURL("/unstar?id=baba"),
			errorCode: 0,
		},
		{
			desc:      "unstar wrong album ID",
			url:       testURL("/unstar?albumId=baba"),
			errorCode: 0,
		},
		{
			desc:      "unstar wrong artist ID",
			url:       testURL("/unstar?artistId=baba"),
			errorCode: 0,
		},
		{
			desc:      "unstar wrong ID type - reserved ID",
			url:       testURL("/unstar?id=%d", int64(1e9)),
			errorCode: 0,
		},
		{
			desc:      "unstar wrong album ID type - reserved ID",
			url:       testURL("/unstar?albumId=%d", int64(1e9)),
			errorCode: 0,
		},
		{
			desc:      "unstar wrong artist ID type - reserved ID",
			url:       testURL("/unstar?artistId=%d", int64(1e9)),
			errorCode: 0,
		},
		{
			desc:      "unstar no arguments",
			url:       testURL("/unstar"),
			errorCode: 10,
		},
		{
			desc:      "getTopSongs no arguments",
			url:       testURL("/getTopSongs"),
			errorCode: 10,
		},
		{
			desc:      "getTopSongs artist not found",
			url:       testURL("/getTopSongs?artist=Not+Found"),
			errorCode: 70,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				test.url,
				nil,
			)

			rec := httptest.NewRecorder()

			ssHandler.ServeHTTP(rec, req)

			resp := rec.Result()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("HTTP response had code %d", resp.StatusCode)
			}

			respBody := rec.Body.String()

			xmlResp := errorResponse{}
			dec := xml.NewDecoder(bytes.NewBufferString(respBody))
			if err := dec.Decode(&xmlResp); err != nil {
				t.Fatalf("cannot decode XML: %s", err)
			}

			if xmlResp.Status != "failed" {
				t.Logf("XML response: %s\n", respBody)
				t.Fatalf("expected FAILED response but got `%s`", xmlResp.Status)
			}

			if test.errorCode != xmlResp.Error.Code {
				t.Errorf("expected error code %d but got %d",
					test.errorCode,
					xmlResp.Error.Code,
				)
			}

			if xmlResp.Error.Message == "" {
				t.Errorf("error message was empty. It should always have text")
			}

			err := xsdhandler.ValidateMem(
				[]byte(respBody),
				xsdvalidate.ValidErrDefault,
			)
			if err != nil {
				switch verr := err.(type) {
				case xsdvalidate.ValidationError:
					var errors int
					for _, xmlErr := range verr.Errors {
						if xmlErr.NodeName == "subsonic-response" &&
							xmlErr.Code == 1866 {
							// Few attributes are added for more
							// information even though they are
							// not part of the specification.
							continue
						}

						t.Logf("Error in line: %d\n", xmlErr.Line)
						t.Log(xmlErr.Message)

						t.Logf("error: %#v\n", xmlErr)

						errors++
					}

					if errors > 0 {
						t.Errorf("XSD validation failed with %d errors", errors)
					}
				default:
					t.Errorf("general XSD validation error: %s", err)
				}
			}

		})
	}
}

const xsdFileName = "subsonic-rest-api-1.16.1.xsd"
