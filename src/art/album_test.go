package art_test

import (
	"bytes"
	"context"
	"errors"
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

		t.Logf("MB handler: release: %s, artist: %s\n", release, artist)

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
				<release id="dd65beff-0bfb-4425-81af-ed4cb1945c7f" ns2:score="98">
					<title>Killers</title>
				</release>
				<release id="6518fd52-58bf-44a3-8150-00e7c3ffcae5" ns2:score="99">
					<title>Killers</title>
				</release>
			</release-list>
			</metadata>
		`)
	}
	mbrainz := httptest.NewServer(http.HandlerFunc(mbrainzHandler))
	defer mbrainz.Close()

	artCli := art.NewClient("euterpe/testing", 0, "")
	artCli.SetMusicBrainzAPIURL(mbrainz.URL)

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
	artCli.SetCAAClient(caaClient)

	// In in order to make sure no accidental requests to the Discogs API are made from
	// tests.
	artCli.SetDiscogsAPIURL(mbrainz.URL)

	ctx := context.Background()
	img, err := artCli.GetFrontImage(ctx, artistName, releaseName)

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

// TestClientGetFrontImageErrors checks various types of errors which may be
// returned by the art Client for albums images.
func TestClientGetFrontImageErrors(t *testing.T) {
	const (
		releaseName = "Killers"
		artistName  = "Iron Maiden"

		noImgsRelease = "Senjutsu"
		caaErrRelase  = "The Book of Souls"
	)

	mbrainzHandler := func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/ws/2/release/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		artist, release := parseMBQuery(req.URL.Query().Get("query"))
		if release == "" || artist == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		t.Logf("MB handler: release: %s, artist: %s\n", release, artist)

		if artist == artistName && release == noImgsRelease {
			fmt.Fprintf(w, `
				<metadata created="2021-09-18T11:04:00.452Z">
				<release-list count="2" offset="0">
					<release id="d341f368-c1f5-4fe8-b09d-8a7ce8294433" ns2:score="98">
						<title>Senjutsu</title>
					</release>
					<release id="42993e88-d656-40af-9f98-bfff3c4d09dd" ns2:score="99">
						<title>Senjutsu</title>
					</release>
				</release-list>
				</metadata>
			`)
			return
		}

		if artist == artistName && release == caaErrRelase {
			fmt.Fprintf(w, `
				<metadata created="2021-09-18T11:04:00.452Z">
				<release-list count="2" offset="0">
					<release id="1a62e97f-64b9-4c17-b892-f0536fe50e48" ns2:score="98">
						<title>The Book of Souls</title>
					</release>
				</release-list>
				</metadata>
			`)
			return
		}

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
				<release id="dd65beff-0bfb-4425-81af-ed4cb1945c7f" ns2:score="98">
					<title>Killers</title>
				</release>
				<release id="6518fd52-58bf-44a3-8150-00e7c3ffcae5" ns2:score="99">
					<title>Killers</title>
				</release>
			</release-list>
			</metadata>
		`)
	}
	mbrainz := httptest.NewServer(http.HandlerFunc(mbrainzHandler))
	defer mbrainz.Close()

	artCli := art.NewClient("euterpe/testing", 0, "")
	artCli.SetMusicBrainzAPIURL(mbrainz.URL)

	caaClient := &artfakes.FakeCAAClient{
		GetReleaseFrontStub: func(mbid uuid.UUID, size int) (caa.CoverArtImage, error) {
			withImageUUID := caa.StringToUUID("6518fd52-58bf-44a3-8150-00e7c3ffcae5")
			withErr := caa.StringToUUID("1a62e97f-64b9-4c17-b892-f0536fe50e48")

			if uuid.Equal(mbid, withErr) {
				return caa.CoverArtImage{}, caa.HTTPError{
					StatusCode: http.StatusInternalServerError,
					URL:        &url.URL{},
				}
			}

			if !uuid.Equal(mbid, withImageUUID) {
				return caa.CoverArtImage{}, caa.HTTPError{
					StatusCode: http.StatusNotFound,
					URL:        &url.URL{},
				}
			}

			return caa.CoverArtImage{
				Data:     []byte("some image"),
				Mimetype: "text/plain",
			}, nil
		},
	}
	artCli.SetCAAClient(caaClient)

	// In in order to make sure no accidental requests to the Discogs API are made from
	// tests.
	artCli.SetDiscogsAPIURL(mbrainz.URL)

	ctx := context.Background()

	// Check when there are no releases with at least min score.
	originalMinScore := artCli.MinScore
	artCli.MinScore = 100 // 100 ensures that no release will match.
	_, err := artCli.GetFrontImage(ctx, artistName, releaseName)
	if !errors.Is(err, art.ErrImageNotFound) {
		t.Errorf("min score: expected error 'not found' but got `%s`", err)
	}
	artCli.MinScore = originalMinScore // reset the min score

	// Check the error type for when no releases have been found in music brainz
	// whatsoever.
	_, err = artCli.GetFrontImage(ctx, "not found", "not found")
	if !errors.Is(err, art.ErrImageNotFound) {
		t.Errorf("not found: expected error 'not found' but got `%s`", err)
	}

	// There are matching releases but they don't have any images in the
	// cover art archive.
	_, err = artCli.GetFrontImage(ctx, artistName, noImgsRelease)
	if !errors.Is(err, art.ErrImageNotFound) {
		t.Errorf("no images: expected error 'not found' but got `%s`", err)
	}

	// Check that the original CAA Client error is returned when one happens.
	_, err = artCli.GetFrontImage(ctx, artistName, caaErrRelase)
	var caaErr caa.HTTPError
	if !errors.As(err, &caaErr) {
		t.Errorf("expected error of type caa.HTTPError but got %T\n", err)
	} else if caaErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected caa.HTTPError to be 500 but got %d", caaErr.StatusCode)
	}

	// Checks that making a bad request is explained in the error.
	_, err = artCli.GetFrontImage(ctx, "", "")
	if err == nil {
		t.Errorf("bad request: expected an error but got none")
	} else if !strings.Contains(err.Error(), "HTTP 400") {
		t.Errorf("bad request: expected HTTP 400 error but got: %s", err)
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
