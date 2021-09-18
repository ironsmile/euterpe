package art_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ironsmile/euterpe/src/art"
)

// TestClientGetArtistImage makes sure that the art.Client is making the appropriate
// requests to the Music Brainz and Discogs APIs. And that then it returns the
// expected image. This is the "happy" path of the process where it manages to find
// an image without much problems.
func TestClientGetArtistImage(t *testing.T) {
	const (
		discogsToken = "discogsToken"
		userAgent    = "euterpe/testing"
		artistName   = "Iron Maiden"
	)

	var (
		imgHandlerCalled bool
		mbzHandlerCalled bool
		dscHandlerCalled bool
	)

	imageBytes := []byte("some image")
	serverErrors := []string{}

	imgHandler := func(w http.ResponseWriter, req *http.Request) {
		imgHandlerCalled = true
		_, _ = w.Write(imageBytes)
	}
	imgServer := httptest.NewServer(http.HandlerFunc(imgHandler))
	defer imgServer.Close()

	mbrainzHandler := func(w http.ResponseWriter, req *http.Request) {
		mbzHandlerCalled = true

		if req.UserAgent() != userAgent {
			serverErrors = append(
				serverErrors,
				fmt.Sprintf("expected user agent '%s' but got '%s'",
					userAgent,
					req.UserAgent(),
				),
			)
		}

		if req.Method != http.MethodGet {
			serverErrors = append(
				serverErrors,
				fmt.Sprintf("mbhandler: HTTP method %s used instead of get", req.Method),
			)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if req.URL.Path == "/ws/2/artist/" {
			query := req.URL.Query().Get("query")
			if query == "" {
				serverErrors = append(
					serverErrors,
					"mbhandler: empty query for artist",
				)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			parts := strings.Split(query, ":")
			if len(parts) != 2 || parts[0] != "artist" || len(parts[1]) < 1 {
				serverErrors = append(
					serverErrors,
					fmt.Sprintf("mbhandler: malformed query: `%s`", query),
				)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			queryArtist := parts[1]
			if queryArtist != artistName {
				serverErrors = append(
					serverErrors,
					fmt.Sprintf("mbhandler: unknown artist in query: `%s`", queryArtist),
				)

				fmt.Fprint(w, `<metadata created="2021-09-17T19:16:24.295Z">
					<artist-list count="0" offset="0"/>
					</metadata>`)
				return
			}

			fmt.Fprintf(w, `
				<metadata created="2021-09-17T19:15:05.632Z">
				<artist-list count="3" offset="0">
					<artist id="not-the-good-maiden" type="Group" type-id="e431f5f6-b5d2-343d-8b36-72607fffb74b" ns2:score="99">
						<name>Iron Maiden</name>
					</artist>
					<artist id="ca891d65-d9b0-4258-89f7-e6ba29d83767" type="Group" type-id="e431f5f6-b5d2-343d-8b36-72607fffb74b" ns2:score="98">
						<name>Iron Maiden</name>
					</artist>
					<artist id="7c3762a3-51f8-4cf3-8565-1ee26a90efe2" type="Group" type-id="e431f5f6-b5d2-343d-8b36-72607fffb74b" ns2:score="85">
						<name>Iron Maiden</name>
					</artist>
				</artist-list>
				</metadata>
			`)
		} else if req.URL.Path == "/ws/2/artist/not-the-good-maiden" {
			// This response intentionally does not have relation for Discogs in order
			// to test that the Client will continue with the next Music Brainz ID.

			fmt.Fprintf(w, `
				<metadata>
					<artist id="ca891d65-d9b0-4258-89f7-e6ba29d83767" type="Group" type-id="e431f5f6-b5d2-343d-8b36-72607fffb74b">
						<name>Iron Maiden</name>
					</artist>
				</metadata>
			`)

		} else if req.URL.Path == "/ws/2/artist/ca891d65-d9b0-4258-89f7-e6ba29d83767" {
			if req.URL.Query().Get("inc") != "url-rels" {
				serverErrors = append(
					serverErrors,
					"mbhandler: request for artist information did not have ?inc=url-rels",
				)
			}

			fmt.Fprintf(w, `
				<metadata>
					<artist id="ca891d65-d9b0-4258-89f7-e6ba29d83767" type="Group" type-id="e431f5f6-b5d2-343d-8b36-72607fffb74b">
						<name>Iron Maiden</name>
						<relation-list target-type="url">
							<relation type="fanpage" type-id="f484f897-81cc-406e-96f9-cd799a04ee24">
								<target id="b94957e9-0424-4a52-aa9d-c1a795d583e6">http://maidenfans.com/</target>
								<direction>forward</direction>
							</relation>
							<relation type="discogs" type-id="04a5b104-a4c2-4bac-99a1-7b837c37d9e4">
								<target id="85ed2140-457c-4a3d-8660-a870ab4e6432">https://www.discogs.com/artist/251595</target>
								<direction>forward</direction>
							</relation>
						</relation-list>
					</artist>
				</metadata>
			`)
		} else {
			serverErrors = append(
				serverErrors,
				fmt.Sprintf("mbhandler: unknown URI: `%s`", req.URL.Path),
			)

			w.WriteHeader(http.StatusNotFound)
		}
	}
	mbrainz := httptest.NewServer(http.HandlerFunc(mbrainzHandler))
	defer mbrainz.Close()

	discogsHandler := func(w http.ResponseWriter, req *http.Request) {
		dscHandlerCalled = true

		if req.UserAgent() != userAgent {
			serverErrors = append(
				serverErrors,
				fmt.Sprintf("expected user agent '%s' but got '%s'",
					userAgent,
					req.UserAgent(),
				),
			)
		}

		if req.Method != http.MethodGet {
			serverErrors = append(
				serverErrors,
				fmt.Sprintf("dshandler: HTTP method not allowed: `%s`", req.Method),
			)

			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if req.URL.Path != "/artists/251595" {
			serverErrors = append(
				serverErrors,
				fmt.Sprintf("dshandler: unknown path requested: `%s`", req.URL.Path),
			)

			w.WriteHeader(http.StatusNotFound)
			return
		}

		fmt.Fprintf(w, `{
			"id": 251595,
			"name": "Iron Maiden",
			"images": [
				{
					"type": "primary",
					"uri": ""
				},
				{
					"type": "primary",
					"uri": "%s"
				}
			]
		}`, imgServer.URL)
	}
	discogs := httptest.NewServer(http.HandlerFunc(discogsHandler))
	defer discogs.Close()

	c := art.NewClient(userAgent, 0, discogsToken)
	c.SetMusicBrainzAPIURL(mbrainz.URL)
	c.SetDiscogsAPIURL(discogs.URL)

	foundImage, err := c.GetArtistImage(context.Background(), artistName)

	for _, serverError := range serverErrors {
		t.Errorf("test server error: %s", serverError)
	}

	if !mbzHandlerCalled {
		t.Error("the mbrainz test server was never called")
	}

	if !dscHandlerCalled {
		t.Error("the discogs test server was never called")
	}

	if !imgHandlerCalled {
		t.Error("the image test server was never called")
	}

	if err != nil {
		t.Fatalf("Getting image error: %s\n", err)
	}
	if !bytes.Equal(imageBytes, foundImage) {
		t.Errorf("expected image response to be `%s` but got `%s`",
			imageBytes, foundImage)
	}
}

// TestClientNoDiscogsAuth makes sure the appropriate error is returned when
// the Discogs client hasn't been configured.
func TestClientNoDiscogsAuth(t *testing.T) {
	c := art.NewClient("euterpe/testing", 0, "")
	buff, err := c.GetArtistImage(context.Background(), "Iron Maiden")

	if !errors.Is(err, art.ErrNoDiscogsAuth) {
		t.Errorf("Wrong error returned. Expected ErrNoDiscogsAuth, got %v", err)
	}

	if buff != nil {
		t.Errorf("Expected image buffer to be empty when Discogs auth errors are returned")
	}
}
