package webserver_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ironsmile/euterpe/src/assert"
	"github.com/ironsmile/euterpe/src/version"
	"github.com/ironsmile/euterpe/src/webserver"
)

// TestAboutHandler makes sure that the /v1/about HTTP handler returns the information
// promised in the API contract.
func TestAboutHandler(t *testing.T) {
	aboutHandler := webserver.NewAboutHandler()

	req := httptest.NewRequest(http.MethodGet, "/v1/about", nil)
	resp := httptest.NewRecorder()

	aboutHandler.ServeHTTP(resp, req)
	defer resp.Result().Body.Close()

	assert.Equal(t, http.StatusOK, resp.Result().StatusCode)
	if !strings.Contains(resp.Result().Header.Get("Content-Type"), "application/json") {
		t.Errorf("expected application/json content type")
	}

	kvStore := map[string]any{}
	dec := json.NewDecoder(resp.Result().Body)
	err := dec.Decode(&kvStore)
	assert.NilErr(t, err, "decoding JSON response failed")

	val, found := kvStore["server_version"]
	if !found {
		t.Fatalf("did not find `server_version` in JSON object")
	}

	valStr, ok := val.(string)
	if !ok {
		t.Fatalf("`server_version` was not a string")
	}

	assert.Equal(t, version.Version, valStr, "version string mismatch")
}
