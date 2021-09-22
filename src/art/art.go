package art

import (
	"context"
	"errors"
	"sync"
	"time"

	cca "gopkg.in/mineo/gocaa.v1"
)

// ErrImageNotFound is returned by the Get* functions when no suitable cover image
// was found anywhere.
var ErrImageNotFound = errors.New("image not found")

// ErrImageTooBig is returned when some image has been find but it is deemed to big
// for the server to handle.
var ErrImageTooBig = errors.New("image is too big")

//counterfeiter:generate . Finder

// Finder defines a type which is capable of finding art for artists or albums.
type Finder interface {
	// GetFrontImage returns the front album artwork for particular album
	// by an artist.
	GetFrontImage(ctx context.Context, artist, album string) ([]byte, error)

	// GetArtistImage returns an image which represents a particular artist.
	// Hopefully a good one! ;D
	GetArtistImage(ctx context.Context, artist string) ([]byte, error)
}

// Client is a client for recovering artwork. It supports getting images from
// the Cover Arts Archive and automatically throttles itself so that it does not make too
// many requests at once. It also supports getting artist images from the Discogs
// database. It is safe for concurrent use.
//
// Getting images from Cover Arts Archive works in two steps:
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
// Getting images for artists from Discogs works in similar way. The only difference
// is that there is an additional request for getting the discogsID of an artist
// using the mbid.
//
// It implements Finder.
type Client struct {
	sync.Mutex

	// MinScore is the minimal accepted score above which a release is considered
	// a match for the search in the Music Brainz API. The API returns a list of
	// matches and every one of them comes with a "score" metric in 0-100 scale
	// which represents how good a match is this result for the query. 100 means
	// absolutely sure. By lowering this score you may receive more images but
	// some of them may be inaccurate.
	MinScore int

	delay            time.Duration
	delayer          *time.Timer
	useragent        string
	discogsAuthToken string
	caaClient        CAAClient

	musicBrainzAPIHost string
	discogsAPIHost     string
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
//
// The Discogs API has taken a different path to achieve the same. It requires
// you to make requests with a Discogs Token which you generate with your personal
// account.
func NewClient(useragent string, delay time.Duration, discogsToken string) *Client {
	return &Client{
		MinScore:           95,
		useragent:          useragent,
		delay:              delay,
		delayer:            time.NewTimer(delay),
		caaClient:          cca.NewCAAClient(useragent),
		musicBrainzAPIHost: "https://musicbrainz.org",
		discogsAPIHost:     "https://api.discogs.com",
		discogsAuthToken:   discogsToken,
	}
}
