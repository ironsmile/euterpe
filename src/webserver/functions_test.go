package webserver_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ironsmile/euterpe/src/webserver"
)

// TestWithInternalError makes sure that Internal Server Error status code is
// set when the underlying handler returns an error and hasn't written anything
// at the output.
func TestWithInternalError(t *testing.T) {
	var someError = fmt.Errorf("test-error")

	h := webserver.WithInternalError(func(_ http.ResponseWriter, _ *http.Request) error {
		return someError
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	statusCode := resp.Result().StatusCode
	if statusCode != http.StatusInternalServerError {
		t.Errorf("expected status code %d but got %d",
			http.StatusInternalServerError, statusCode)
	}

	if !strings.Contains(resp.Body.String(), someError.Error()) {
		// Make sure the error string is part of the response body. This makes
		// debugging any problems much easier.
		t.Errorf("response body did not include the error string")
	}
}
