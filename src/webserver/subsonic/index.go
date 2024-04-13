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

	albumArtHandler  CoverArtHandler
	artistArtHandler CoverArtHandler

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
	albumArt CoverArtHandler,
	artistArt CoverArtHandler,
) http.Handler {
	handler := &subsonic{
		prefix:           prefix,
		lib:              lib,
		libBrowser:       libBrowser,
		needsAuth:        cfg.Auth,
		auth:             cfg.Authenticate,
		albumArtHandler:  albumArt,
		artistArtHandler: artistArt,
		lastModified:     time.Now(),
	}

	handler.initRouter()

	return handler
}

func (s *subsonic) initRouter() {
	router := mux.NewRouter()
	router.StrictSlash(true)
	router.UseEncodedPath()

	router.Handle(
		Prefix+"/ping",
		http.HandlerFunc(s.apiPing),
	).Methods("GET")

	router.Handle(
		Prefix+"/ping.view",
		http.HandlerFunc(s.apiPing),
	).Methods("GET")

	router.Handle(
		Prefix+"/getLicense",
		http.HandlerFunc(s.getLicense),
	).Methods("GET")

	router.Handle(
		Prefix+"/getMusicFolders",
		http.HandlerFunc(s.getMusicFolders),
	).Methods("GET")
	router.Handle(
		Prefix+"/getMusicFolders.view",
		http.HandlerFunc(s.getMusicFolders),
	).Methods("GET")

	router.Handle(
		Prefix+"/getIndexes",
		http.HandlerFunc(s.getIndexes),
	).Methods("GET")
	router.Handle(
		Prefix+"/getIndexes.view",
		http.HandlerFunc(s.getIndexes),
	).Methods("GET")

	router.Handle(
		Prefix+"/getMusicDirectory",
		http.HandlerFunc(s.getMusicDirectory),
	).Methods("GET")
	router.Handle(
		Prefix+"/getMusicDirectory.view",
		http.HandlerFunc(s.getMusicDirectory),
	).Methods("GET")

	router.Handle(
		Prefix+"/getArtists",
		http.HandlerFunc(s.getArtists),
	).Methods("GET")
	router.Handle(
		Prefix+"/getArtists.view",
		http.HandlerFunc(s.getArtists),
	).Methods("GET")

	router.Handle(
		Prefix+"/getAlbum",
		http.HandlerFunc(s.getAlbum),
	).Methods("GET")
	router.Handle(
		Prefix+"/getAlbum.view",
		http.HandlerFunc(s.getAlbum),
	).Methods("GET")

	router.Handle(
		Prefix+"/getAlbumList2",
		http.HandlerFunc(s.getAlbumList2),
	).Methods("GET")
	router.Handle(
		Prefix+"/getAlbumList2.view",
		http.HandlerFunc(s.getAlbumList2),
	).Methods("GET")

	router.Handle(
		Prefix+"/getArtist",
		http.HandlerFunc(s.getArtist),
	).Methods("GET")
	router.Handle(
		Prefix+"/getArtist.view",
		http.HandlerFunc(s.getArtist),
	).Methods("GET")

	router.Handle(
		Prefix+"/getArtistInfo2",
		http.HandlerFunc(s.getArtistInfo2),
	).Methods("GET")
	router.Handle(
		Prefix+"/getArtistInfo2.view",
		http.HandlerFunc(s.getArtistInfo2),
	).Methods("GET")

	router.Handle(
		Prefix+"/getCoverArt",
		http.HandlerFunc(s.getCoverArt),
	).Methods("GET")
	router.Handle(
		Prefix+"/getCoverArt.view",
		http.HandlerFunc(s.getCoverArt),
	).Methods("GET")

	router.Handle(
		Prefix+"/stream",
		http.HandlerFunc(s.stream),
	).Methods("GET")
	router.Handle(
		Prefix+"/stream.view",
		http.HandlerFunc(s.stream),
	).Methods("GET")

	s.mux = s.authHandler(router)
}

func (s *subsonic) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.mux.ServeHTTP(w, req)
}

func (s *subsonic) getLastModified() time.Time {
	return s.lastModified
}
