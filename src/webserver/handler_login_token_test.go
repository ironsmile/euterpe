package webserver_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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
			h := webserver.NewLoginTokenHandler(cfg)
			req := httptest.NewRequest(
				http.MethodPost,
				"/v1/login/token",
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
