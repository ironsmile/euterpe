package webserver

import (
	"net/http"

	"github.com/ironsmile/httpms/src/library"
)

// BrowseHandler is a http.Handler which will allow you to browse through artists or albums
// with the help of pagination.
type BrowseHandler struct {
	library library.Library
}

// ServeHTTP is required by the http.Handler's interface
func (fh BrowseHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	InternalErrorOnErrorHandler(writer, req, fh.browse)
}

// Actually generates a browse results using the library
func (fh BrowseHandler) browse(writer http.ResponseWriter, req *http.Request) error {
	writer.Header().Add("Content-Type", "application/json; charset=utf-8")

	return nil
}

// NewBrowseHandler returns a new Browse handler. It needs a library to browse through.
func NewBrowseHandler(lib library.Library) *BrowseHandler {
	fh := new(BrowseHandler)
	fh.library = lib
	return fh
}
