package webserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/ironsmile/euterpe/src/webserver"
)

// TestFileHandlerWithNoLibrary makes sure that the handler works even without a
// library and that it returns "internal server error" in this case.
func TestFileHandlerWithNoLibrary(t *testing.T) {
	h := routeFileHandler(webserver.NewFileHandler(nil))

	req := httptest.NewRequest(http.MethodGet, "/v1/file/23", nil)
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	statusCode := resp.Result().StatusCode
	if statusCode != http.StatusInternalServerError {
		t.Errorf(
			"expected HTTP status code %d but got %d",
			http.StatusInternalServerError,
			statusCode,
		)
	}
}

// TestFileHandlerWithWrongPathVars makes sure that the handler returns "not found"
// when there is no ID in its gorilla mux.
func TestFileHandlerWithWrongPathVars(t *testing.T) {
	// Simulate no gorilla mux by not having one! :D
	h := webserver.NewFileHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	statusCode := resp.Result().StatusCode
	if statusCode != http.StatusNotFound {
		t.Errorf(
			"expected HTTP status code %d but got %d",
			http.StatusNotFound,
			statusCode,
		)
	}
}

// routeFileHandler wraps a handler the same way the web server will do when
// constructing the main application router. This is needed for tests so that the
// Gorilla mux variables will be parsed.
func routeFileHandler(h http.Handler) http.Handler {
	router := mux.NewRouter()
	router.StrictSlash(true)
	router.UseEncodedPath()
	router.Handle(webserver.APIv1EndpointFile, h).Methods(
		webserver.APIv1Methods[webserver.APIv1EndpointFile]...,
	)

	return router
}
