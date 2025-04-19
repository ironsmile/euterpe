package webutils_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ironsmile/euterpe/src/webserver/webutils"
)

// TestJSONError makes sure that the JSONError function really encodes the response
// as a valid JSON.
func TestJSONError(t *testing.T) {
	rec := httptest.NewRecorder()
	errMsg := "some error message for testing"

	webutils.JSONError(rec, errMsg, http.StatusBadGateway)

	res := rec.Result()
	defer func() {
		res.Body.Close()
	}()

	if res.StatusCode != http.StatusBadGateway {
		t.Errorf("Expected Bad Gateway status but got %d", res.StatusCode)
	}

	var respJSON json.RawMessage
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&respJSON); err != nil {
		t.Errorf("Failed decoding the JSON response: %s", err)
	}
}
