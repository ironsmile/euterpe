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

	setUpGetHandler := func(path string, handler http.HandlerFunc) {
		router.Handle(
			Prefix+path,
			http.HandlerFunc(handler),
		).Methods("GET")

		router.Handle(
			Prefix+path+".view",
			http.HandlerFunc(handler),
		).Methods("GET")
	}

	setUpGetHandler("/ping", s.apiPing)
	setUpGetHandler("/getLicense", s.getLicense)
	setUpGetHandler("/getMusicFolders", s.getMusicFolders)
	setUpGetHandler("/getIndexes", s.getIndexes)
	setUpGetHandler("/getMusicDirectory", s.getMusicDirectory)
	setUpGetHandler("/getArtists", s.getArtists)
	setUpGetHandler("/getAlbum", s.getAlbum)
	setUpGetHandler("/getAlbumList2", s.getAlbumList2)
	setUpGetHandler("/getArtist", s.getArtist)
	setUpGetHandler("/getArtistInfo2", s.getArtistInfo2)
	setUpGetHandler("/getCoverArt", s.getCoverArt)
	setUpGetHandler("/stream", s.stream)
	setUpGetHandler("/download", s.stream)
	setUpGetHandler("/getSong", s.getSong)

	s.mux = s.authHandler(router)
}

func (s *subsonic) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.mux.ServeHTTP(w, req)
}

func (s *subsonic) getLastModified() time.Time {
	return s.lastModified
}
