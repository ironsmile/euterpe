package subsonic

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ironsmile/euterpe/src/config"
	"github.com/ironsmile/euterpe/src/library"
)

type subsonic struct {
	prefix     string
	libBrowser library.Browser
	lib        library.Library
	needsAuth  bool
	auth       config.Auth

	//!TODO: track real lastModified centrally. On every insert or
	// delete in the database.
	lastModified time.Time

	mux http.Handler
}

// Prefix is the URL path prefix for all subsonic API endpoints.
const Prefix = "/rest"

// NewHandler returns a HTTP handler which would serve the subsonic API
// (https://www.subsonic.org/pages/api.jsp). Endpoints are served after
// the `prefix` URL path.
func NewHandler(
	prefix string,
	lib library.Library,
	libBrowser library.Browser,
	cfg config.Config,
) http.Handler {
	handler := &subsonic{
		prefix:       prefix,
		lib:          lib,
		libBrowser:   libBrowser,
		needsAuth:    cfg.Auth,
		auth:         cfg.Authenticate,
		lastModified: time.Now(),
	}

	handler.initRouter()

	return handler
}

func (s *subsonic) initRouter() {
	router := mux.NewRouter()
	router.StrictSlash(true)
	router.UseEncodedPath()

	router.Handle(
		Prefix+"/ping/",
		http.HandlerFunc(s.apiPing),
	).Methods("GET")

	router.Handle(
		Prefix+"/getLicense/",
		http.HandlerFunc(s.getLicense),
	).Methods("GET")

	router.Handle(
		Prefix+"/getMusicFolders/",
		http.HandlerFunc(s.getMusicFolders),
	).Methods("GET")

	router.Handle(
		Prefix+"/getIndexes/",
		http.HandlerFunc(s.getIndexes),
	).Methods("GET")

	router.Handle(
		Prefix+"/getMusicDirectory/",
		http.HandlerFunc(s.getMusicDirectory),
	).Methods("GET")

	router.Handle(
		Prefix+"/getArtists/",
		http.HandlerFunc(s.getArtists),
	).Methods("GET")

	s.mux = s.authHandler(router)
}

func (s *subsonic) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.mux.ServeHTTP(w, req)
}

func (s *subsonic) getLastModified() time.Time {
	return s.lastModified
}
