package subsonic_test

import (
	"bytes"
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
	lib := &libraryfakes.FakeLibrary{
		GetArtistAlbumsStub: func(i int64) []library.Album {
			return []library.Album{
				{
					ID:        22,
					Name:      "First Album",
					Artist:    "First Artist",
					SongCount: 8,
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
		GetAlbumFilesStub: func(i int64) []library.SearchResult {
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
				},
			}
		},
	}
	browser := &libraryfakes.FakeBrowser{
		BrowseArtistsStub: func(ba library.BrowseArgs) ([]library.Artist, int) {
			if ba.Page > 1 {
				return nil, 3
			}

			resp := []library.Artist{
				{
					ID:         1,
					Name:       "First Artist",
					AlbumCount: 2,
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

			return resp, len(resp)
		},

		BrowseAlbumsStub: func(ba library.BrowseArgs) ([]library.Album, int) {
			if ba.Page > 1 {
				return nil, 3
			}

			resp := []library.Album{
				{
					ID:        1,
					Name:      "First Album",
					Artist:    "First Artist",
					SongCount: 5,
				},
				{
					ID:        2,
					Name:      "Second Album",
					Artist:    "Second Artist",
					SongCount: 9,
				},
				{
					ID:        5223,
					Name:      "Third Album",
					Artist:    "First Artist",
					SongCount: 22,
				},
			}

			return resp, len(resp)
		},
	}

	xsdvalidate.Init()
	defer xsdvalidate.Cleanup()

	xsdhandler, err := xsdvalidate.NewXsdHandlerUrl(
		"subsonic-rest-api-1.16.1.xsd",
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
			url:  testURL("/getMusicDirectory?id=%d", int64(2e9+10)),
		},
		{
			desc: "getArtist",
			url:  testURL("/getArtist?id=%d", int64(1e9+10)),
		},
		{
			desc: "getAlbum",
			url:  testURL("/getAlbum?id=%d", int64(2e9+10)),
		},
		{
			desc: "getAlbumList2",
			url:  testURL("/getAlbumList2?type=random&id=%d", int64(2e9+10)),
		},
		{
			desc: "getArtistInfo2",
			url:  testURL("/getArtistInfo2?id=%d", int64(1e9+10)),
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
