package webserver_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ironsmile/euterpe/src/config"
	"github.com/ironsmile/euterpe/src/webserver"
)

// TestLoginHandlerSuccessful checks the golden path for logging in with the
// correct user and password.
func TestLoginHandlerSuccessful(t *testing.T) {
	cfg := config.Auth{
		User:     "test-user",
		Password: "test-password",
		Secret:   "test-secret",
	}

	tests := []struct {
		desc       string
		url        string
		reqUser    string
		reqPass    string
		rememberMe bool

		expectedReturnTo string
	}{
		{
			desc:    "standard login",
			url:     "/",
			reqUser: cfg.User,
			reqPass: cfg.Password,

			expectedReturnTo: "/",
		},
		{
			desc:    "with some return_to",
			url:     "/?return_to=/a/test/place?with=query",
			reqUser: cfg.User,
			reqPass: cfg.Password,

			expectedReturnTo: "/a/test/place?with=query",
		},
		{
			desc:       "with remember_me",
			url:        "/?return_to=/a/return-to",
			reqUser:    cfg.User,
			reqPass:    cfg.Password,
			rememberMe: true,

			expectedReturnTo: "/a/return-to",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			h := webserver.NewLoginHandler(cfg)

			formSting := fmt.Sprintf(
				"username=%s&password=%s", cfg.User, cfg.Password,
			)
			if test.rememberMe {
				formSting += "&remember_me=on"
			}
			req := httptest.NewRequest(
				http.MethodPost,
				test.url,
				strings.NewReader(formSting),
			)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp := httptest.NewRecorder()

			h.ServeHTTP(resp, req)

			if resp.Result().StatusCode != http.StatusFound {
				t.Fatalf(
					"expected HTTP status %d but got %d",
					http.StatusFound,
					resp.Result().StatusCode,
				)
			}

			var sessionCookie *http.Cookie
			for _, cookie := range resp.Result().Cookies() {
				if cookie.Name != "session" {
					continue
				}
				sessionCookie = cookie
			}

			if sessionCookie == nil {
				t.Fatal("login handler did not return a session cookie")
			}

			locationHeader := resp.Result().Header.Get("Location")
			if locationHeader != test.expectedReturnTo {
				t.Fatalf(
					"expected Location: header to be `%s` but it was `%s`",
					test.expectedReturnTo,
					locationHeader,
				)
			}

			assertToken(t, sessionCookie.Value, cfg.Secret)

			if test.rememberMe {
				if !sessionCookie.Expires.After(time.Now().Add(744 * time.Hour)) {
					t.Errorf(
						"expected cookies to expire at least in a month but it does in %s",
						time.Until(sessionCookie.Expires),
					)
				}
			} else if !sessionCookie.Expires.IsZero() {
				t.Errorf("session cookies without remember me should not have expiration")
			}
		})
	}
}

// TestLoginHandlerWrong tests that wrong credentials result in denied access and
// an redirect to the proper page.
func TestLoginHandlerWrong(t *testing.T) {
	cfg := config.Auth{
		User:     "test-user",
		Password: "test-password",
		Secret:   "test-secret",
	}

	const returnTo = "/a/test/place?with=query"

	h := webserver.NewLoginHandler(cfg)
	req := httptest.NewRequest(
		http.MethodPost,
		"/?return_to="+returnTo,
		strings.NewReader("username=wrong&password=credetianls"),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	if resp.Result().StatusCode != http.StatusFound {
		t.Fatalf(
			"expected HTTP status %d but got %d",
			http.StatusFound,
			resp.Result().StatusCode,
		)
	}

	var sessionCookie *http.Cookie
	for _, cookie := range resp.Result().Cookies() {
		fmt.Printf("cookie: %+v\n", cookie)
		if cookie.Name != "session" {
			continue
		}
		sessionCookie = cookie
	}

	if sessionCookie != nil {
		t.Fatal("login handler was supposed to not set session cookies on error")
	}

	locationHeader := resp.Result().Header.Get("Location")
	if !strings.HasPrefix(locationHeader, "/login/") {
		t.Fatalf(
			"wrong creds were supposed to redirect back to login page but we got `%s`",
			locationHeader,
		)
	}
}
