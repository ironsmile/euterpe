package webserver

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/ironsmile/httpms/src/library"
)

// Handler responsible for search requests. It will use the Library to
// return a list of matched files to the interface.
type SearchHandler struct {
	library library.Library
}

// This method is required by the http.Handler's interface
func (sh SearchHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	InternalErrorOnErrorHandler(writer, req, sh.search)
}

func (sh SearchHandler) search(writer http.ResponseWriter, req *http.Request) error {

	writer.Header().Add("Content-Type", "application/json; charset=utf-8")

	query, err := url.QueryUnescape(req.URL.Path)

	if err != nil {
		return err
	}

	results := sh.library.Search(query)

	if len(results) == 0 {
		writer.Write([]byte("[]"))
		return nil
	}

	marshalled, err := json.Marshal(results)

	if err != nil {
		return err
	}

	writer.Write(marshalled)

	return nil
}

// Returns a new SearchHandler for processing search queries. They will be run
// against the supplied library
func NewSearchHandler(lib library.Library) *SearchHandler {
	sh := new(SearchHandler)
	sh.library = lib
	return sh
}
