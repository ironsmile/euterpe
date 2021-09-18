package art_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ironsmile/euterpe/src/art"
	"github.com/ironsmile/euterpe/src/art/artfakes"
	"github.com/pborman/uuid"
	caa "gopkg.in/mineo/gocaa.v1"
)

// TestClientGetFrontImage checks the golden path for getting a cover image for an
// album.
func TestClientGetFrontImage(t *testing.T) {
	const (
		releaseName = "Killers"
		artistName  = "Iron Maiden"
	)

	var (
		releaseImage = []byte("image contents")
		serverErrors []string
	)

	mbrainzHandler := func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/ws/2/release/" {
			serverErrors = append(
				serverErrors,
				fmt.Sprintf("unknown path requested: %s", req.URL.Path),
			)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		artist, release := parseMBQuery(req.URL.Query().Get("query"))
		if release == "" || artist == "" {
			serverErrors = append(
				serverErrors,
				"no release or artist found in the query string",
			)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		fmt.Printf("release: %s, artist: %s\n", release, artist)

		if release != releaseName || artist != artistName {
			fmt.Fprintf(w, `
				<metadata created="2021-09-18T11:04:00.452Z">
				<release-list count="0" offset="0">
				</release-list>
				</metadata>
			`)
			return
		}

		fmt.Fprintf(w, `
			<metadata created="2021-09-18T11:04:00.452Z">
			<release-list count="2" offset="0">
				<release id="dd65beff-0bfb-4425-81af-ed4cb1945c7f" ns2:score="99">
					<title>Killers</title>
				</release>
				<release id="6518fd52-58bf-44a3-8150-00e7c3ffcae5" ns2:score="100">
					<title>Killers</title>
				</release>
			</release-list>
			</metadata>
		`)
	}
	mbrainz := httptest.NewServer(http.HandlerFunc(mbrainzHandler))
	defer mbrainz.Close()

	c := art.NewClient("euterpe/testing", 0, "")
	c.SetMusicBrainzAPIURL(mbrainz.URL)

	caaClient := &artfakes.FakeCAAClient{
		GetReleaseFrontStub: func(mbid uuid.UUID, size int) (caa.CoverArtImage, error) {
			withImageUUID := caa.StringToUUID("6518fd52-58bf-44a3-8150-00e7c3ffcae5")

			if !uuid.Equal(mbid, withImageUUID) {
				return caa.CoverArtImage{}, caa.HTTPError{
					StatusCode: http.StatusNotFound,
					URL:        &url.URL{},
				}
			}

			imgCopy := make([]byte, len(releaseImage))
			copy(imgCopy, releaseImage)

			return caa.CoverArtImage{
				Data:     imgCopy,
				Mimetype: "text/plain",
			}, nil
		},
	}
	c.SetCAAClient(caaClient)

	// In in order to make sure no accidental requests to the Discogs API are made from
	// tests.
	c.SetDiscogsAPIURL(mbrainz.URL)

	ctx := context.Background()
	img, err := c.GetFrontImage(ctx, artistName, releaseName)

	for _, se := range serverErrors {
		t.Error(se)
	}

	if err != nil {
		t.Fatalf("expected no error but got `%s`", err)
	}

	if !bytes.Equal(releaseImage, img) {
		t.Errorf(
			"release image was not the same, expected `%s` but got `%s`",
			releaseImage,
			img,
		)
	}

	if caaClient.GetReleaseFrontCallCount() != 2 {
		t.Errorf(
			"expected 2 calls to the CoverArt image server but got %d",
			caaClient.GetReleaseFrontCallCount(),
		)
	}
}

func parseMBQuery(s string) (string, string) {
	parts := strings.Split(s, "AND")

	var (
		artist  string
		release string
	)

	for _, part := range parts {
		queryPair := strings.Split(strings.TrimSpace(part), ":")
		if len(queryPair) != 2 {
			continue
		}
		switch queryPair[0] {
		case "artist":
			artist = queryPair[1]
		case "release":
			release = queryPair[1]
		}
	}

	return artist, release
}
