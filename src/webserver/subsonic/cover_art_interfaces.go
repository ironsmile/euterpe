package subsonic

import "net/http"

//counterfeiter:generate . CoverArtHandler

// CoverArtHandler is an interface which exposes a http.Handler like function for
// serving art images. It uses the database IDs for the albums and artists.
type CoverArtHandler interface {
	Find(w http.ResponseWriter, req *http.Request, id int64) error
}
