package library

import (
	"context"
	"io"
)

// ArtworkManager is an interface for all the methods needed for managing album artwork
// in the local library.
type ArtworkManager interface {

	// FindAndSaveAlbumArtwork returns the artwork for a particular album by its ID.
	FindAndSaveAlbumArtwork(ctx context.Context, albumID int64) (io.ReadCloser, error)

	// SaveAlbumArtwork stores the artwork for particular album for later use.
	SaveAlbumArtwork(ctx context.Context, albumID int64, r io.Reader) error

	// RemoveAlbumArtwork removes the stored artwork for particular album.
	RemoveAlbumArtwork(ctx context.Context, albumID int64) error
}

// ArtistImageManager is an interface for all methods for managing artist imagery.
type ArtistImageManager interface {
	// FindAndSaveArtistImage returns the image for a particular artist by its ID.
	FindAndSaveArtistImage(ctx context.Context, artistID int64) (io.ReadCloser, error)

	// SaveArtistImage stores the image for particular artist for later use.
	SaveArtistImage(ctx context.Context, artistID int64, r io.Reader) error

	// RemoveArtistImage removes the stored image for particular artist.
	RemoveArtistImage(ctx context.Context, artistID int64) error
}
