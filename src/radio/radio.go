package radio

import (
	"context"
	"errors"
	"net/url"
)

//counterfeiter:generate . Stations

// Stations is the interface which is used for handling internet radio stations.
type Stations interface {
	// GetAll returns all radio stations stored in the server.
	GetAll(ctx context.Context) ([]Station, error)

	// Create creates a new radio station with the information from `new`. The ID field
	// is ignored.
	//
	// Returns the ID of the newly created station when error is nil.
	Create(ctx context.Context, new Station) (int64, error)

	// Replace changes the data for the radio station with ID `updated.ID`. It uses all
	// the properties of `updated` for updating. If `updated.HomePage` is nil then the
	// home page of the station will be reset even if it previously had one.
	Replace(ctx context.Context, updated Station) error

	// Delete removes a radio station with id `stationID`.
	Delete(ctx context.Context, stationID int64) error
}

// Station represents a single radio station.
type Station struct {
	// ID is a unique identifier for the radio station.
	ID int64

	// Name is a human readable name of the radio station. Used for presentation.
	Name string

	// StreamURL is the address at which the radio station stream is being broadcasted.
	StreamURL url.URL

	// HomePage is the web page of the radio station if it has one. May be nil when
	// the radio station does not have a web page.
	HomePage *url.URL
}

// ErrNotFound is returned when a radio station was not found for a given operation.
var ErrNotFound = errors.New("station not found")
