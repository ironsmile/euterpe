// Package library deals with the actual media library. It is creates the Library type.
//
// Every media receives an ID in the library. The main thing a search result returns
// is the tracks' IDs. They are used to get the media, again using the Library. That
// way the real location of the file is never revealed to the interface.
package library

import (
	"context"
	"time"
)

// SearchResult contains a result for a search term. Contains all the necessary
// information to uniquely identify a media in the library.
type SearchResult struct {

	// ID in the library for a media file
	ID int64 `json:"id"`

	// Meta info: Artist ID
	ArtistID int64 `json:"artist_id"`

	// Meta info: Artist
	Artist string `json:"artist"`

	// Meta info: Album ID
	AlbumID int64 `json:"album_id"`

	// Meta info: Album for music
	Album string `json:"album"`

	// Meta info: the title of this media file
	Title string `json:"title"`

	// Meta info: track number for music
	TrackNumber int64 `json:"track"`

	// File format of the underlying data file. Examples: "mp3", "flac", "ogg" etc.
	Format string `json:"format"`

	// Duration is the track length in milliseconds.
	Duration int64 `json:"duration"`

	// Plays is the number of times this media file has been played.
	Plays int64 `json:"plays,omitempty"`

	// Favourite is non-zero when the media file has been added to the list
	// of favourites. Its value when non-zero is the Unix timestamp at which
	// the track has been added to the favourites.
	Favourite int64 `json:"favourite,omitempty"`

	// LastPlayed is the Unix timestamp (in seconds) at which this media file
	// was last played.
	LastPlayed int64 `json:"last_played,omitempty"`

	// Rating is the user rating given to this media file. It will be a number
	// in the [1-5] range or 0 if no rating was given.
	Rating uint8 `json:"rating,omitempty"`

	// Year is the four digit year at which this track was recorded.
	Year int32 `json:"year,omitempty"`

	// Bitrate is measured in bits per second.
	Bitrate uint64 `json:"bitrate,omitempty"`

	// Size is the size of the media file in bytes.
	Size int64 `json:"size,omitempty"`

	// CreatedAt is a unix timestamp of the time this track was added to the
	// library.
	//
	// Not encoded in the JSON response the API for the moment.
	CreatedAt int64 `json:"-"`
}

// SearchArgs is the input parameters for searching in the library.
type SearchArgs struct {
	// Query is the search query string. Search results will match this query.
	// An empty query will return all elements.
	Query string

	// Offset is an offset in the returned search results. Clients can use it
	// to skip results they already know about.
	Offset uint32

	// Count limits the number of items returned by a search. A Count of zero
	// means "no limit".
	Count uint32
}

// TrackInfo contains information for a single media file.
type TrackInfo = SearchResult

// Artist represents an artist from the database
type Artist struct {
	ID         int64  `json:"artist_id"`
	Name       string `json:"artist"`
	AlbumCount int64  `json:"album_count"`

	// Favourite is non-zero when the artist has been added to the list
	// of favourites. When non-zero its value is the Unix timestamp at
	// witch the artist was added to the list of favourites.
	Favourite int64 `json:"favourite,omitempty"`

	// Rating is the user rating given to this artist. It will be a number
	// in the [1-5] range or 0 if no rating was given.
	Rating uint8 `json:"rating,omitempty"`
}

// Album represents an album from the database
type Album struct {
	ID        int64  `json:"album_id"`
	Name      string `json:"album"`
	Artist    string `json:"artist"`
	SongCount int64  `json:"track_count"`
	Duration  int64  `json:"duration"` // in milliseconds

	// Plays is the number of times tracks in this album has been played.
	Plays int64 `json:"plays,omitempty"`

	// Favourite is non-zero when the album has been added to the list
	// of favourites. When non-zero its value is the Unix timestamp at
	// witch the album was added to the list of favourites.
	Favourite int64 `json:"favourite,omitempty"`

	// LastPlayed is the Unix timestamp (in seconds) at which a track from tims
	// album has been played.
	LastPlayed int64 `json:"last_played,omitempty"`

	// Rating is the user rating given to this album. It will be a number
	// in the [1-5] range or 0 if no rating was given.
	Rating uint8 `json:"rating,omitempty"`

	// Year is a four digit number for the year in which the album has been released.
	Year int32 `json:"year,omitempty"`
}

// Favourites describes a set of favourite tracks, artists and albums.
type Favourites struct {
	ArtistIDs []int64
	AlbumIDs  []int64
	TrackIDs  []int64
}

//counterfeiter:generate . Library

// Library represents the media library which is played using the HTTPMS.
// It is responsible for scanning the library directories, watching for new files,
// actually searching for a media by a search term and finding the exact file path
// in the file system for a media.
type Library interface {

	// Adds a new path to the library paths. If it hasn't been scanned yet a new scan
	// will be started.
	AddLibraryPath(directory string)

	// Search the library using a search string. It will match against Artist, Album
	// and Title. Will OR the results. So it is "return anything which Artist matches or
	// Album matches or Title matches".
	Search(ctx context.Context, args SearchArgs) []SearchResult

	// SearchAlbums searches the library for the given terms and returns matching
	// albums. It may look into artist names, song names and actual album names.
	SearchAlbums(ctx context.Context, args SearchArgs) []Album

	// SearchArtists searches the library for the given terms and returns matching
	// artists. It looks into the artist name only.
	SearchArtists(ctx context.Context, args SearchArgs) []Artist

	// Returns the real filesystem path. Requires the media ID.
	GetFilePath(ctx context.Context, mediaID int64) string

	// Returns search result will all the files of this album.
	GetAlbumFiles(ctx context.Context, albumID int64) []TrackInfo

	// GetArtistAlbums returns all the albums which this artist has an at least
	// on track in.
	GetArtistAlbums(ctx context.Context, artistID int64) []Album

	// GetTrack returns information for particular track identified by its
	// media ID.
	GetTrack(ctx context.Context, mediaID int64) (TrackInfo, error)

	// GetArtist returns information for particular artist in the database.
	GetArtist(ctx context.Context, artistID int64) (Artist, error)

	// GetAlbum returns information for particular album in the database.
	GetAlbum(ctx context.Context, albumID int64) (Album, error)

	// RecordTrackPlay stores the fact that this track has been played
	// at this particular time. This means updating its "last played" property
	// and increasing its play count in the stats database.
	RecordTrackPlay(ctx context.Context, mediaID int64, atTime time.Time) error

	// SetTrackRating sets the rating for particular track. Only values in the
	// [0-5] range are accepted. 0 unsets the rating.
	SetTrackRating(ctx context.Context, mediaID int64, rating uint8) error

	// SetAlbumRating sets the rating for particular album. Only values in the
	// [0-5] range are accepted. 0 unsets the rating.
	SetAlbumRating(ctx context.Context, albumID int64, rating uint8) error

	// SetArtistRating sets the rating for particular artist. Only values in the
	// [0-5] range are accepted. 0 unsets the rating.
	SetArtistRating(ctx context.Context, artistID int64, rating uint8) error

	// RecordFavourite marks as "favourite" a variable number of tracks, albums
	// or artists. If an item is already in the favourites the operation is a
	// no-op and it stays there as before.
	RecordFavourite(ctx context.Context, favs Favourites) error

	// RemoveFavourite unmarks as "favourite" a variable number of tracks, albums
	// or artists. For items which are not among the favourite nothing is done.
	RemoveFavourite(ctx context.Context, favs Favourites) error

	// Starts a full library scan. Will scan all paths if
	// they are not scanned already.
	Scan()

	// Adds this media (file) to the library.
	AddMedia(fileName string) error

	// Makes sure the library is initialized. This method will be called once on
	// every start of Euterpe.
	Initialize() error

	// Makes the library forget everything. Also Closes the library.
	Truncate() error

	// Frees all resources this library object is using.
	// Any operations (except Truncate) on closed library will result in panic.
	Close()
}
