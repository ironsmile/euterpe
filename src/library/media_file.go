package library

import "time"

// MediaFile is an interface which a media object should satisfy in order to be inserted
// in the library database.
type MediaFile interface {

	// Artist returns a string which represents the artist responsible for this media file
	Artist() string

	// Album returns a string for the name of the album this media file is part of
	Album() string

	// Title returns the name of this piece of media
	Title() string

	// Track returns the media track number in its album
	Track() int

	// Length returns the duration of this piece of media
	Length() time.Duration
}
