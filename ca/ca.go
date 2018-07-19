package ca

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	cca "gopkg.in/mineo/gocaa.v1"
)

// CoverArtImage represents a cover image from the Cover Arts Archive.
type CoverArtImage = cca.CoverArtImage

const (
	musicBrainzReleaseIDEndpint = "https://musicbrainz.org/ws/2/release/"
	musicBrainzQueryValue       = "release:%s AND artist:%s"
)

// ErrImageNotFound is returned by the Get* functions when no suitable cover image
// was found anywhere.
var ErrImageNotFound = errors.New("image not found")

// CovertArtFinder defines a type which is capable of finding
type CovertArtFinder interface {
	GetFrontImage(ctx context.Context, artist, album string) (CoverArtImage, error)
}

// Client is a client for the Cover Arts Archive. It supports getting images from
// the Cover Arts Archive and automatically throttles itself so that it does not make too
// many requests at once. It is safe for concurrent use.
//
// It works in two steps:
//
// * Gets a list of mbids (aka release IDs) from the Music Brainz API which are above
// MinScore.
//
// * Uses the mbids for fetching a cover art from the Cover Art Archive. The first
// release ID which has a cover art wins.
//
// Why a list of mbids? Because a certain album may have many records in Music Brainz
// which correspond to different releases for this album. Perhaps for multiple years
// or countries. Generally all releases have the same cover art. So we accept any of
// them.
//
// It implements CovertArtFinder.
type Client struct {
	sync.Mutex

	// MinScore is the minimal accepted score above which a release is considered
	// a match for the search in the Music Brainz API. The API returns a list of
	// matches and every one of them comes with a "score" metric in 0-100 scale
	// which represents how good a match is this result for the query. 100 means
	// absolutely sure. By lowering this score you may receive more images but
	// some of them may be inaccurate.
	MinScore int

	delay     time.Duration
	delayer   *time.Timer
	useragent string
}

// NewClient returns fully configured Client.
//
// The kind people at MusicBrainz provide their API at no cost for everyone
// to use. For that reason they have kindly asked for all applications to
// throttle their usage as much as possible and do not exceed one request
// per second. So we are good citizen and throttle ourselves.
// More info: https://musicbrainz.org/doc/XML_Web_Service/Rate_Limiting
// For this reason the delayer and delay are defined here.
//
// Throttling is done with the help of the arguments `useragent` and a `delay`. The user
// agent is used for representing itself when contacting the Music Brainz API. It is
// required so that they can use it for throttling and filtering out bad applications.
// The delay is used to throttle requests to the API. No more than one request per
// `delay` will be made.
func NewClient(useragent string, delay time.Duration) *Client {
	return &Client{
		MinScore:  95,
		useragent: useragent,
		delay:     delay,
		delayer:   time.NewTimer(delay),
	}
}

// GetFrontImage returns the front image for particular `album` from `artist`.
func (c *Client) GetFrontImage(
	ctx context.Context,
	artist,
	album string,
) (CoverArtImage, error) {
	mbIDs, err := c.getMusicBrainzReleaseID(ctx, artist, album)
	if err != nil {
		return CoverArtImage{}, err
	}

	for _, mbidStr := range mbIDs {
		mbid := cca.StringToUUID(mbidStr)
		ccaClient := cca.NewCAAClient(c.useragent)

		img, err := ccaClient.GetReleaseFront(mbid, cca.ImageSize500)
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

// getMusicBrainzReleaseID uses the MusicBrainz API to retrieve a list of matching
// MusicBrainzIDs (or mbid) for particular "release". Or album in HTTPMS parlance.
func (c *Client) getMusicBrainzReleaseID(
	ctx context.Context,
	artist,
	album string,
) ([]string, error) {
	c.Lock()
	defer c.Unlock()

	<-c.delayer.C
	defer c.delayer.Reset(c.delay)

	req, err := http.NewRequest(http.MethodGet, musicBrainzReleaseIDEndpint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating music brainz XML API req: %s", err)
	}

	query := req.URL.Query()
	query.Add("query", fmt.Sprintf(musicBrainzQueryValue, album, artist))
	req.URL.RawQuery = query.Encode()
	req.Header.Set("User-Agent", c.useragent)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
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
		if release.Score >= c.MinScore {
			releaseIDs = append(releaseIDs, release.ID)
		}
	}

	if len(releaseIDs) < 1 {
		return nil, ErrImageNotFound
	}

	return releaseIDs, nil
}

// The following are structures only used to decode the XML response from MusicBrainz
// API. And only the stuff we are interested and nothing more.
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
