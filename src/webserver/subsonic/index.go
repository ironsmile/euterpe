package subsonic

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ironsmile/euterpe/src/config"
	"github.com/ironsmile/euterpe/src/library"
	"github.com/ironsmile/euterpe/src/radio"
)

type subsonic struct {
	prefix     string
	libBrowser library.Browser
	lib        library.Library
	radio      radio.Stations
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
	stations radio.Stations,
	cfg config.Config,
	albumArt CoverArtHandler,
	artistArt CoverArtHandler,
) http.Handler {
	handler := &subsonic{
		prefix:           prefix,
		lib:              lib,
		libBrowser:       libBrowser,
		radio:            stations,
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

	setUpHandler := func(path string, handler http.HandlerFunc, methods ...string) {
		if len(methods) == 0 {
			methods = []string{http.MethodGet, http.MethodPost}
		}
		router.Handle(
			Prefix+path,
			http.HandlerFunc(handler),
		).Methods(methods...)

		router.Handle(
			Prefix+path+".view",
			http.HandlerFunc(handler),
		).Methods(methods...)
	}

	setUpHandler("/ping", s.apiPing)
	setUpHandler("/getLicense", s.getLicense)
	setUpHandler("/getOpenSubsonicExtensions", s.getOpenSubsonicExtensions)
	setUpHandler("/getMusicFolders", s.getMusicFolders)
	setUpHandler("/getIndexes", s.getIndexes)
	setUpHandler("/getMusicDirectory", s.getMusicDirectory)
	setUpHandler("/getArtists", s.getArtists)
	setUpHandler("/getAlbum", s.getAlbum)
	setUpHandler("/getAlbumList", s.getAlbumList)
	setUpHandler("/getAlbumList2", s.getAlbumList2)
	setUpHandler("/getArtist", s.getArtist)
	setUpHandler("/getArtistInfo", s.getArtistInfo)
	setUpHandler("/getArtistInfo2", s.getArtistInfo2)
	setUpHandler("/getCoverArt", s.getCoverArt, "GET", "HEAD")
	setUpHandler("/stream", s.stream, "GET", "HEAD")
	setUpHandler("/download", s.stream, "GET", "HEAD")
	setUpHandler("/getSong", s.getSong)
	setUpHandler("/getGenres", s.getGenres)
	setUpHandler("/getVideos", s.getVideos)
	setUpHandler("/getVideoInfo", s.getVideoInfo)
	setUpHandler("/search3", s.search3)
	setUpHandler("/search2", s.search2)
	setUpHandler("/search", s.search)
	setUpHandler("/scrobble", s.scrobble)
	setUpHandler("/setRating", s.setRating)
	setUpHandler("/star", s.star)
	setUpHandler("/unstar", s.unstar)
	setUpHandler("/getStarred", s.getStarred)
	setUpHandler("/getStarred2", s.getStarred2)
	setUpHandler("/getTopSongs", s.getTopSongs)
	setUpHandler("/getAlbumInfo", s.getAlbumInfo)
	setUpHandler("/getAlbumInfo2", s.getAlbumInfo2)
	setUpHandler("/getInternetRadioStations", s.getInternetRadionStations)
	setUpHandler("/createInternetRadioStation", s.createInternetRadioStation)
	setUpHandler("/updateInternetRadioStation", s.updateInternetRadioStation)
	setUpHandler("/deleteInternetRadioStation", s.deleteInternetRadioStation)
	setUpHandler("/getUser", s.getUser)
	setUpHandler("/getRandomSongs", s.getRandomSongs)
	setUpHandler("/createPlaylist", s.createPlaylist)

	s.mux = s.authHandler(router)
}

func (s *subsonic) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.mux.ServeHTTP(w, req)
}

func (s *subsonic) getLastModified() time.Time {
	return s.lastModified
}
