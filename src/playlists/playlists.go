package playlists

import (
	"context"
	"errors"
	"time"

	"github.com/ironsmile/euterpe/src/library"
)

//counterfeiter:generate . Playlister

// Playlister is the interface for handling playlists in Euterpe.
type Playlister interface {
	// Get returns a single playlist by its ID.
	Get(ctx context.Context, id int64) (Playlist, error)

	// List returns a list playlists. Does not return the tracks associated with each
	// playlist. Set both [args.Count] and [args.Offset] to zero in order to
	// list all playlists at once.
	List(ctx context.Context, args ListArgs) ([]Playlist, error)

	// Count returns the count of all playlists available.
	Count(ctx context.Context) (int64, error)

	// Create creates a new playlist with the given name. `songs` is an list
	// of track IDs to be added in the playlist.
	//
	// Returns the unique ID of the newly created playlist.
	Create(ctx context.Context, name string, tracks []int64) (int64, error)

	// Update updates the playlist with ID `id` with the values
	// given in `args`. Note that everything in args is optional
	// and will not change the playlist if the zero value of the
	// property is left.
	Update(ctx context.Context, id int64, args UpdateArgs) error

	// Delete removes a playlist by its `id`.
	Delete(ctx context.Context, id int64) error
}

// Playlist represents a single playlist.
type Playlist struct {
	ID     int64  // ID is the unique number which identifies this playlist.
	Name   string // Name is the user-facing name of the playlist.
	Desc   string // Desc is a text which describes the playlist.
	Public bool   // Public is true if the playlist will be visible for all users.

	Duration  time.Duration // Duration is the overall duration of the playlist.
	CreatedAt time.Time     // CreatedAt is the time when this playlist was created.
	UpdatedAt time.Time     // UpdatedAt is the time of the last update of the playlist.

	// TracksCount is the number of tracks in this playlist. Relevant for when
	// the playlist is returned without populated `Tracks`.
	TracksCount int64

	// Tracks is the which are added to this playlist. The slice is ordered by
	// the tracks' explicit order in the playlist.
	Tracks []library.TrackInfo
}

// UpdateArgs is all the possible arguments which could be updated
// for a given playlist.
type UpdateArgs struct {
	Name   string // Name is the new name of the playlist.
	Desc   string // Desc sets the playlist description.
	Public *bool  // Public sets the public field of the playlist.

	// AddTracks is a list of track IDs which will be added to the
	// playlist. Tracks are added _after_ removing is done.
	AddTracks []int64

	// RemoveTracks is a list of positions in the playlist for tracks
	// to be removed from it.
	RemoveTracks []int64

	// MoveTracks tracks is a list move move operations for tracks by their
	// indexes. Moving is evaluated _after_ adding is done.
	MoveTracks []MoveArgs

	// RemoveAllTracks causes all tracks of the playlist to be removed.
	RemoveAllTracks bool
}

// MoveArgs defines a single move of a track from one position in the playlist
// to another. Positions are encoded as indexes in the playlist.
type MoveArgs struct {
	FromIndex uint32
	ToIndex   uint32
}

// ListArgs defines what portion of the playlists list will be returned.
type ListArgs struct {
	// Offset is an index in the list of playlist from which to start the list.
	Offset int64

	// Count defines how many playlists to include in the response. The special value
	// of zero means "all playlists".
	Count int64
}

// ErrNotFound is returned when a playlist was not found for a given operation.
var ErrNotFound = errors.New("playlist not found")
