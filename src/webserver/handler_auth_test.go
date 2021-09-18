package webserver_test

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gbrlsnchs/jwt"
	"github.com/ironsmile/euterpe/src/webserver"
)

// TestAuthHandlerDifferentAuthMethods makes sure that the auth handler still supports
// all of its authentication methods.
func TestAuthHandlerDifferentAuthMethods(t *testing.T) {
	const (
		username = "auth_user"
		password = "auth_pass"
		secret   = "auth_secret_which_is_completely_unknown_to_anyone_promise"
	)

	getToken := func() string {
		tokenOpts := &jwt.Options{
			Timestamp:      true,
			ExpirationTime: time.Now().Add(10 * time.Minute),
		}
		token, err := jwt.Sign(jwt.HS256(secret), tokenOpts)
		if err != nil {
			panic(err)
		}
		return token
	}

	tests := []struct {
		desc         string
		newRequest   func() *http.Request
		expectedCode int
		exceptions   []string
	}{
		{
			desc: "bearer JWT token",
			newRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", getToken()))
				return req
			},
			expectedCode: http.StatusOK,
		},
		{
			desc: "query token",
			newRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				query := req.URL.Query()
				query.Add("token", getToken())
				req.URL.RawQuery = query.Encode()
				return req
			},
			expectedCode: http.StatusOK,
		},
		{
			desc: "session cookie",
			newRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.AddCookie(&http.Cookie{
					Name:  "session",
					Value: getToken(),
				})
				return req
			},
			expectedCode: http.StatusOK,
		},
		{
			desc: "basic authenticate",
			newRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.SetBasicAuth(username, password)
				return req
			},
			expectedCode: http.StatusOK,
		},
		{
			desc: "path with exception",
			newRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/noauth", nil)
			},
			exceptions:   []string{"/noauth"},
			expectedCode: http.StatusOK,
		},
		{
			desc: "unauthenticated JSON client",
			newRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Accept", "application/json")
				return req
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			desc: "unauthenticated HTML client (browser)",
			newRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Accept", "text/html")
				return req
			},
			expectedCode: http.StatusFound,
		},
		{
			desc: "no authentication whatsoever",
			newRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Accept", "application/json")
				return req
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			desc: "malformed token",
			newRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Accept", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", "baba"))
				return req
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			desc: "token created with different secret",
			newRequest: func() *http.Request {
				tokenOpts := &jwt.Options{
					Timestamp:      true,
					ExpirationTime: time.Now().Add(10 * time.Minute),
				}
				token, err := jwt.Sign(jwt.HS256("not the correct secret"), tokenOpts)
				if err != nil {
					panic(err)
				}

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Accept", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
				return req
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			desc: "expired token",
			newRequest: func() *http.Request {
				tokenOpts := &jwt.Options{
					Timestamp:      true,
					ExpirationTime: time.Now().Add(-10 * time.Minute),
				}
				token, err := jwt.Sign(jwt.HS256(secret), tokenOpts)
				if err != nil {
					panic(err)
				}

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Accept", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
				return req
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			desc: "different encoding algo used",
			newRequest: func() *http.Request {
				tokenOpts := &jwt.Options{
					Timestamp:      true,
					ExpirationTime: time.Now().Add(-10 * time.Minute),
				}
				token, err := jwt.Sign(jwt.HS384(secret), tokenOpts)
				if err != nil {
					panic(err)
				}

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Accept", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
				return req
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			desc: "badly encoded base64 basic auth",
			newRequest: func() *http.Request {
				authString := "baba004456--"

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Accept", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Basic %s", authString))
				return req
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			desc: "no password basic auth",
			newRequest: func() *http.Request {
				authString := base64.StdEncoding.EncodeToString([]byte("justonevalue"))

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("Accept", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("Basic %s", authString))
				return req
			},
			expectedCode: http.StatusUnauthorized,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			wrapped := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprintf(w, "OK")
			})

			auh := webserver.NewAuthHandler(
				wrapped,
				username,
				password,
				nil,
				secret,
				test.exceptions,
			)

			req := test.newRequest()
			resp := httptest.NewRecorder()

			auh.ServeHTTP(resp, req)

			if resp.Code != test.expectedCode {
				t.Errorf("authentication expected HTTP %d but got %d",
					test.expectedCode,
					resp.Code,
				)
			}
		})
	}
}
