package library

import "io"

// ArtworkFinder is an interface for all the methods needed for finding an artwork
// in the loca library.
type ArtworkFinder interface {

	// GetAlbumArtwork returns the artwork for a particular album by its ID.
	GetAlbumArtwork(albumID int64) (io.ReadCloser, error)
}
