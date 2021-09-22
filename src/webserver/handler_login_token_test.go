package webserver_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/ironsmile/euterpe/src/config"
	"github.com/ironsmile/euterpe/src/webserver"
)

// TestLoginTokenHandler uses the login-with-token HTTP handler and makes sure the
// generated token is correct.
func TestLoginTokenHandler(t *testing.T) {
	cfg := config.Auth{
		User:     "test-user",
		Password: "test-pass",
		Secret:   "test-secret",
	}

	tests := []struct {
		desc               string
		reqBody            func(t *testing.T) io.Reader
		expectedStatusCode int
	}{
		{
			desc: "successful login",
			reqBody: func(t *testing.T) io.Reader {
				reqBodyJSON := struct {
					User string `json:"username"`
					Pass string `json:"password"`
				}{
					User: cfg.User,
					Pass: cfg.Password,
				}
				var reqBody bytes.Buffer
				enc := json.NewEncoder(&reqBody)
				if err := enc.Encode(reqBodyJSON); err != nil {
					t.Fatalf("encoding request JSON failed: %s", err)
				}

				return &reqBody
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			desc: "wrong credentials for login",
			reqBody: func(t *testing.T) io.Reader {
				reqBodyJSON := struct {
					User string `json:"username"`
					Pass string `json:"password"`
				}{
					User: "wrong",
					Pass: "also wrong possibly",
				}
				var reqBody bytes.Buffer
				enc := json.NewEncoder(&reqBody)
				if err := enc.Encode(reqBodyJSON); err != nil {
					t.Fatalf("encoding request JSON failed: %s", err)
				}

				return &reqBody
			},
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			desc: "malformed JSON",
			reqBody: func(t *testing.T) io.Reader {
				return bytes.NewBufferString("totally not a JSON")
			},
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			h := routeLoginTokenHandler(webserver.NewLoginTokenHandler(cfg))
			req := httptest.NewRequest(
				http.MethodPost,
				"/v1/login/token/",
				test.reqBody(t),
			)
			resp := httptest.NewRecorder()

			h.ServeHTTP(resp, req)

			responseCode := resp.Result().StatusCode
			if responseCode != test.expectedStatusCode {
				t.Fatalf(
					"expected HTTP status code `%d` but got `%d`",
					test.expectedStatusCode,
					responseCode,
				)
			}
			assertContentTypeJSON(t, resp.Result().Header.Get("Content-Type"))

			if test.expectedStatusCode != http.StatusOK {
				return
			}

			tokenResponse := struct {
				Token string `json:"token"`
			}{}

			dec := json.NewDecoder(resp.Result().Body)
			if err := dec.Decode(&tokenResponse); err != nil {
				t.Fatalf("failed to JSON decode token response: %s", err)
			}

			assertToken(t, tokenResponse.Token, cfg.Secret)
		})
	}
}

// routeLoginTokenHandler wraps a handler the same way the web server will do when
// constructing the main application router. This is needed for tests so that the
// Gorilla mux variables will be parsed.
func routeLoginTokenHandler(h http.Handler) http.Handler {
	router := mux.NewRouter()
	router.StrictSlash(true)
	router.UseEncodedPath()
	router.Handle(webserver.APIv1EndpointLoginToken, h).Methods(
		webserver.APIv1Methods[webserver.APIv1EndpointLoginToken]...,
	)

	return router
}
