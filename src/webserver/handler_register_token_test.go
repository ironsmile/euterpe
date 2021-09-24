package webserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/ironsmile/euterpe/src/webserver"
)

// TestRegisterTokenHandler makes sure that the handler returns well formatted JSON
// and responds with HTTP 204.
func TestRegisterTokenHandler(t *testing.T) {
	h := routeRegisterTokenHandler(webserver.NewRigisterTokenHandler())

	req := httptest.NewRequest(http.MethodPost, "/v1/register/token/", nil)
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	responseCode := resp.Result().StatusCode
	if responseCode != http.StatusNoContent {
		t.Errorf(
			"expected HTTP response code `%d` but got `%d`",
			http.StatusNoContent,
			responseCode,
		)
	}
}

// routeRegisterTokenHandler wraps a handler the same way the web server will do when
// constructing the main application router. This is needed for tests so that the
// Gorilla mux variables will be parsed.
func routeRegisterTokenHandler(h http.Handler) http.Handler {
	router := mux.NewRouter()
	router.StrictSlash(true)
	router.UseEncodedPath()
	router.Handle(webserver.APIv1EndpointRegisterToken, h).Methods(
		webserver.APIv1Methods[webserver.APIv1EndpointRegisterToken]...,
	)

	return router
}
