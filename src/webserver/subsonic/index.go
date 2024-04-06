package subsonic

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ironsmile/euterpe/src/config"
	"github.com/ironsmile/euterpe/src/library"
)

type subsonic struct {
	prefix    string
	lib       library.Browser
	needsAuth bool
	auth      config.Auth
	libraries []string

	mux http.Handler
}

// Prefix is the URL path prefix for all subsonic API endpoints.
const Prefix = "/subsonic"

// NewHandler returns a HTTP handler which would serve the subsonic API
// (https://www.subsonic.org/pages/api.jsp). Endpoints are served after
// the `prefix` URL path.
func NewHandler(
	prefix string,
	lib library.Browser,
	cfg config.Config,
) http.Handler {
	handler := &subsonic{
		prefix:    prefix,
		lib:       lib,
		needsAuth: cfg.Auth,
		auth:      cfg.Authenticate,
		libraries: cfg.Libraries,
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

	s.mux = s.authHandler(router)
}

func (s *subsonic) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.mux.ServeHTTP(w, req)
}
