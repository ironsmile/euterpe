package webserver

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/ironsmile/httpms/src/library"
)

// SearchHandler is a http.Handler responsible for search requests. It will use
// the Library to return a list of matched files to the interface.
type SearchHandler struct {
	library library.Library
}

// ServeHTTP is required by the http.Handler's interface
func (sh SearchHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	InternalErrorOnErrorHandler(writer, req, sh.search)
}

func (sh SearchHandler) search(writer http.ResponseWriter, req *http.Request) error {

	writer.Header().Add("Content-Type", "application/json; charset=utf-8")

	if err := req.ParseForm(); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		if _, err := writer.Write([]byte(err.Error())); err != nil {
			log.Printf("error writing body in search handler: %s", err)
		}
		return nil
	}

	query := req.Form.Get("q")

	if query == "" {
		var err error

		vars := mux.Vars(req)
		query, err = url.QueryUnescape(vars["searchQuery"])

		if err != nil {
			return err
		}
	}

	results := sh.library.Search(query)

	if len(results) == 0 {
		_, err := writer.Write([]byte("[]"))
		return err
	}

	enc := json.NewEncoder(writer)
	return enc.Encode(results)
}

// NewSearchHandler returns a new SearchHandler for processing search queries. They
// will be run against the supplied library
func NewSearchHandler(lib library.Library) *SearchHandler {
	sh := new(SearchHandler)
	sh.library = lib
	return sh
}
