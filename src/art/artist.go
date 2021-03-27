package art

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	musicBrainzArtistSearchEndpint = "%s/ws/2/artist/"
	musicBrainzArtistRelsEndpint   = "%s/ws/2/artist/%s?inc=url-rels"
	musicBrainzArtistQueryValue    = "artist:%s"

	discogsArtistEndpoint = "%s/artists/%s"
)

// ErrNoDiscogsAuth signals that there is no configured Discogs token in the
// configuration. This directly means that trying to get an artist image is
// doomed from the get-go.
var ErrNoDiscogsAuth = fmt.Errorf("authentication with Discogs is not configured")

// GetArtistImage finds and returns an image of particular artist. If none is found
// it returns ErrImageNotFound.
func (c *Client) GetArtistImage(
	ctx context.Context,
	artist string,
) ([]byte, error) {
	if c.discogsAuthToken == "" {
		return nil, ErrNoDiscogsAuth
	}

	mbIDs, err := c.getMusicBrainzArtistID(ctx, artist)
	if err != nil {
		return nil, err
	}

	const maxTries = 2
	var (
		discogID string
		tries    int
	)

	for _, mbID := range mbIDs {
		if tries >= maxTries {
			return nil, ErrImageNotFound
		}

		dID, err := c.getDiscogsArtistID(ctx, mbID)
		if err == nil {
			discogID = dID
			break
		}
		tries++

		if errors.Is(err, ErrImageNotFound) {
			continue
		}

		return nil, err
	}

	if discogID == "" {
		return nil, ErrImageNotFound
	}

	return c.getDiscogsArtistImage(ctx, discogID)
}

// getMusicBrainzArtistID uses the MusicBrainz API to retrieve a list of matching
// MusicBrainzIDs (or mbid) for particular "artist".
func (c *Client) getMusicBrainzArtistID(
	ctx context.Context,
	artist string,
) ([]string, error) {
	c.Lock()
	defer c.Unlock()

	<-c.delayer.C
	defer c.delayer.Reset(c.delay)

	mbURL := fmt.Sprintf(musicBrainzArtistSearchEndpint, c.musicBrainzAPIHost)
	req, err := http.NewRequest(http.MethodGet, mbURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating MusicBrainz XML API req: %w", err)
	}

	query := req.URL.Query()
	query.Add("query", fmt.Sprintf(musicBrainzArtistQueryValue, artist))
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
		return nil, fmt.Errorf(
			"artist search XML API (MusicBrainz) returned HTTP %d",
			resp.StatusCode,
		)
	}

	root := mbArtistSearchData{}
	dec := xml.NewDecoder(resp.Body)

	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf(
			"decoding MusicBrainz artist search XML API response: %w",
			err,
		)
	}

	if len(root.ArtistList.Artists) < 1 {
		return nil, ErrImageNotFound
	}

	var artistIDs []string
	for _, artist := range root.ArtistList.Artists {
		if artist.Score >= c.MinScore {
			artistIDs = append(artistIDs, artist.ID)
		}
	}

	if len(artistIDs) < 1 {
		return nil, ErrImageNotFound
	}

	return artistIDs, nil
}

// getDiscogsArtistID parses the URL relations for particular MusicBrainz ID and searches
// for the Discogs ID among them. Then returns it if found.
func (c *Client) getDiscogsArtistID(
	ctx context.Context,
	artistMBid string,
) (string, error) {
	c.Lock()
	defer c.Unlock()

	<-c.delayer.C
	defer c.delayer.Reset(c.delay)

	endpointURL := fmt.Sprintf(
		musicBrainzArtistRelsEndpint,
		c.musicBrainzAPIHost,
		url.PathEscape(artistMBid),
	)
	req, err := http.NewRequest(http.MethodGet, endpointURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creating MusicBrainz XML API req: %w", err)
	}
	req.Header.Set("User-Agent", c.useragent)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"artist XML API (MusicBrainz) returned HTTP %d",
			resp.StatusCode,
		)
	}

	root := mbArtistData{}
	dec := xml.NewDecoder(resp.Body)

	if err := dec.Decode(&root); err != nil {
		return "", fmt.Errorf("decoding MusicBrainz artist XML API response: %w", err)
	}

	for _, artistXML := range root.Artist.RelationsList.Relations {
		if artistXML.Type != "discogs" {
			continue
		}

		discogsURL, err := url.Parse(artistXML.Target)
		if err != nil {
			return "", fmt.Errorf("error parsing Discogs artist URL: %w", err)
		}

		discogsID := strings.TrimPrefix(discogsURL.Path, "/artist/")
		discogsID = strings.TrimSuffix(discogsID, "/")

		if discogsID == "" {
			return "", fmt.Errorf("unrecognised Discogs artist URL format: %s",
				artistXML.Target)
		}

		return discogsID, nil
	}

	return "", ErrImageNotFound
}

func (c *Client) getDiscogsArtistImage(
	ctx context.Context,
	discogID string,
) ([]byte, error) {
	// The Discogs API requests are not guarded behind the Client delayer since all
	// of them are naturally throttled by the MusicBrainz API delays. Why? Because
	// the Discogs API calls can only happen as a result from a MusicBrainz API call
	// proceeding it.

	endpointURL := fmt.Sprintf(
		discogsArtistEndpoint,
		c.discogsAPIHost,
		url.PathEscape(discogID),
	)
	req, err := http.NewRequest(http.MethodGet, endpointURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating Discogs API req: %w", err)
	}
	req.Header.Set("User-Agent", c.useragent)
	req.Header.Set("Authorization", fmt.Sprintf("Discogs token=%s", c.discogsAuthToken))

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to Discogs API failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"artist XML API (Discogs) returned HTTP %d",
			resp.StatusCode,
		)
	}

	var dca dcArtist
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&dca); err != nil {
		return nil, fmt.Errorf("unrecognised JSON returned by Discogs: %w", err)
	}

	// First search for the primary image and use it if found.
	for _, image := range dca.Images {
		if image.URI == "" {
			continue
		}

		imgBytes, err := c.downloadDiscogsImage(ctx, image.URI)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		} else if err != nil {
			log.Printf("error downloading Discogs image: %s\n", err)
			continue
		}

		return imgBytes, nil
	}

	return nil, ErrImageNotFound
}

func (c *Client) downloadDiscogsImage(
	ctx context.Context,
	URL string,
) ([]byte, error) {
	imgReq, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, fmt.Errorf(
			"malformed URL returned by the Discogs API (%s): %w",
			URL,
			err,
		)
	}
	imgReq.Header.Set("User-Agent", c.useragent)

	imgResp, err := http.DefaultClient.Do(imgReq)
	if err != nil {
		return nil, fmt.Errorf("request for Discogs image failed: %w", err)
	}
	defer imgResp.Body.Close()

	if imgResp.StatusCode != http.StatusOK {
		return nil, ErrImageNotFound
	}

	const imageLimitSize = 1024 * 1024 * 2
	imgBytes, err := io.ReadAll(io.LimitReader(imgResp.Body, imageLimitSize))
	if (err == nil || errors.Is(err, io.EOF)) && len(imgBytes) == imageLimitSize {
		return nil, ErrImageTooBig
	}
	if err != nil {
		return nil, fmt.Errorf("getting Discogs image failed: %w", err)
	}

	return imgBytes, nil
}

// The following are structures only used to decode the XML response from MusicBrainz
// API. And only the stuff we are interested and nothing more.
type mbArtistSearchData struct {
	ArtistList mbArtistList `xml:"artist-list"`
}

type mbArtistList struct {
	Artists []mbArtist `xml:"artist"`
}

type mbArtist struct {
	ID            string                `xml:"id,attr"`
	Score         int                   `xml:"score,attr"`
	Name          string                `xml:"name"`
	RelationsList mbArtistRelationsList `xml:"relation-list"`
}

/*
mbArtistData represents the response from the MusicBrainz artist XML. Truncated
example:

<metadata>
    <artist id="id" type="Group" type-id="typeid">
        <name>Iron Maiden</name>
        <relation-list target-type="url">
            <relation type="discogs" type-id="04a5b104-a4c2-4bac-99a1-7b837c37d9e4">
                <target id="target-id">https://www.discogs.com/artist/251595</target>
                <direction>forward</direction>
            </relation>
        </relation-list>
    </artist>
</metadata>
*/
type mbArtistData struct {
	Artist mbArtist `xml:"artist"`
}

type mbArtistRelationsList struct {
	Relations []mbArtistRelation `xml:"relation"`
}

type mbArtistRelation struct {
	Type   string `xml:"type,attr"`
	Target string `xml:"target"`
}

// dcArtist is a type which matches the Discogs JSON representation of an
// artist. It defines only the strictly required fields by the art Finder.
type dcArtist struct {
	ID     int64          `json:"id"`
	Name   string         `json:"name"`
	Images []dcArtstImage `json:"images"`
}

type dcArtstImage struct {
	Type   string `json:"type"`
	URI    string `json:"uri"`
	URI150 string `json:"uri150"`
}
