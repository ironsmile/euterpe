package caa

import (
	"fmt"
	"net/url"
)

// HTTPError is an error that occured while accessing URL.
// Instead of a status code of 200, StatusCode was returned by the server.
type HTTPError struct {
	StatusCode int
	URL        *url.URL
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("%d on %s", e.StatusCode, e.URL.String())
}

// InvalidImageSizeError indicates that Size is not valid for EntityType
type InvalidImageSizeError struct {
	EntityType string
	Size       int
}

func (e InvalidImageSizeError) Error() string {
	return fmt.Sprintf("%s doesn't support image size %d", e.EntityType, e.Size)
}
