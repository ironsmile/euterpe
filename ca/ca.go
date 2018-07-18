package ca

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"time"

	cca "gopkg.in/mineo/gocaa.v1"
)

type CoverArtImage = cca.CoverArtImage

const (
	musicBrainzReleaseIDEndpint = "https://musicbrainz.org/ws/2/release/"
	musicBrainzQueryValue       = "release:%s AND artist:%s"
	userAgent                   = "HTTP Media Server/1.1 (github.com/ironsmile/httpms)"
)

// ErrImageNotFound is returned by the Get* functions when no suitable cover image
// was found anywhere.
var ErrImageNotFound = errors.New("image not found")

// GetFrontImage returns the front image for particular `album` from `artist`.
func GetFrontImage(artist, album string) (CoverArtImage, error) {
	mbIDs, err := getMusicBrainzReleaseID(artist, album)
	if err != nil {
		return CoverArtImage{}, err
	}

	var shouldSleep bool

	for _, mbidStr := range mbIDs {
		if shouldSleep {
			// The kind people at MusicBrainz provide their API at no cost for everyone
			// to use. For that reason they have kindly asked for all applications to
			// throttle their usage as much as possible and do not exceed one request
			// per second. So we are good citizen and throttle ourselves.
			// More info: https://musicbrainz.org/doc/XML_Web_Service/Rate_Limiting
			time.Sleep(1 * time.Second)
		}
		shouldSleep = true

		mbid := cca.StringToUUID(mbidStr)
		ccaClient := cca.NewCAAClient(userAgent)

		img, err := ccaClient.GetReleaseFront(mbid, cca.ImageSizeOriginal)
		if err == nil {
			return img, nil
		}

		httpErr, ok := err.(cca.HTTPError)
		if ok && httpErr.StatusCode == http.StatusNotFound {
			continue
		}

		return img, err
	}

	return CoverArtImage{}, ErrImageNotFound
}
func getMusicBrainzReleaseID(artist, album string) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, musicBrainzReleaseIDEndpint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating music brainz XML API req: %s", err)
	}

	query := req.URL.Query()
	query.Add("query", fmt.Sprintf(musicBrainzQueryValue, album, artist))
	req.URL.RawQuery = query.Encode()
	req.Header.Set("User-Agent", userAgent)

	deadline := time.Now().Add(20 * time.Second)
	ctx, cancelContext := context.WithDeadline(context.Background(), deadline)
	defer cancelContext()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("music brainz XML API returned HTTP %d", resp.StatusCode)
	}

	root := mbMetadata{}
	dec := xml.NewDecoder(resp.Body)

	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("decoding music brainz XML API response: %s", err)
	}

	if len(root.RelaseList.Relases) < 1 {
		return nil, ErrImageNotFound
	}

	var releaseIDs []string
	for _, release := range root.RelaseList.Relases {
		if release.Score > 95 {
			releaseIDs = append(releaseIDs, release.ID)
		}
	}

	if len(releaseIDs) < 1 {
		return nil, ErrImageNotFound
	}

	return releaseIDs, nil
}

type mbMetadata struct {
	RelaseList mbReleaseList `xml:"release-list"`
}

type mbReleaseList struct {
	Relases []mbRelease `xml:"release"`
}

type mbRelease struct {
	ID    string `xml:"id,attr"`
	Score int    `xml:"score,attr"`
	Title string `xml:"title"`
}
