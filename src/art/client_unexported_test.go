package art

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestClientGettingMusicBrainzAristIDs checks the behaviour of the Client's method for
// getting a Music Brainz ID for artists in all kind of situations.
func TestClientGettingMusicBrainzAristIDs(t *testing.T) {
	tests := []struct {
		desc       string
		handler    http.HandlerFunc
		inspectErr func(*testing.T, error)
	}{
		{
			desc: "non 200 status code",
			handler: func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			inspectErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected an error")
				}
				if !strings.Contains(err.Error(), "returned HTTP 404") {
					t.Error("expected an error showing what the XML API returned")
				}
			},
		},
		{
			desc: "malformed XML",
			handler: func(w http.ResponseWriter, req *http.Request) {
				fmt.Fprint(w, `definitely not an XML response`)
			},
			inspectErr: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected an error")
				}
				if !strings.Contains(err.Error(), "decoding") ||
					!strings.Contains(err.Error(), "XML API response") {
					t.Error("expected XML parsing error")
				}
			},
		},
		{
			desc: "no artists in the returned list",
			handler: func(w http.ResponseWriter, req *http.Request) {
				fmt.Fprint(w, `
					<metadata created="2021-09-17T19:15:05.632Z">
					<artist-list count="0" offset="0">
					</artist-list>
					</metadata>
				`)
			},
			inspectErr: func(t *testing.T, err error) {
				if !errors.Is(err, ErrImageNotFound) {
					t.Errorf("expected %v but got %v", ErrImageNotFound, err)
				}
			},
		},
		{
			desc: "no artists passing the min score list",
			handler: func(w http.ResponseWriter, req *http.Request) {
				fmt.Fprint(w, `
					<metadata created="2021-09-17T19:15:05.632Z">
					<artist-list count="0" offset="0">
						<artist id="not-good" ns2:score="2" type="Group" type-id="some-id">
							<name>Iron Maiden</name>
						</artist>
					</artist-list>
					</metadata>
				`)
			},
			inspectErr: func(t *testing.T, err error) {
				if !errors.Is(err, ErrImageNotFound) {
					t.Errorf("expected %v but got %v", ErrImageNotFound, err)
				}
			},
		},
		{
			desc: "malformed HTTP response",
			handler: func(w http.ResponseWriter, req *http.Request) {
				w.Header().Add("content-length", "22")
				_, _ = w.Write([]byte("12"))
			},
			inspectErr: func(t *testing.T, err error) {
				if !errors.Is(err, io.ErrUnexpectedEOF) {
					t.Errorf("expected %v but got %v", io.ErrUnexpectedEOF, err)
				}
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			mbrainz := httptest.NewServer(test.handler)
			defer mbrainz.Close()

			c := NewClient("user-agent/testing", 0, "")
			c.musicBrainzAPIHost = mbrainz.URL

			_, err := c.getMusicBrainzArtistID(context.Background(), "does not matter")
			test.inspectErr(t, err)
		})
	}
}
