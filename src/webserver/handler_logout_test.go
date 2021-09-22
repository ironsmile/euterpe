package webserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ironsmile/euterpe/src/webserver"
)

// TestLogoutHandler make sure that the logout handler clears the session cookie
// and redirects back to another rpage.
func TestLogoutHandler(t *testing.T) {
	h := webserver.NewLogoutHandler()

	req := httptest.NewRequest(http.MethodGet, "/logout/", nil)
	req.AddCookie(&http.Cookie{
		Name:     "session",
		Value:    "some-token",
		Path:     "/",
		HttpOnly: true,
	})
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	respCode := resp.Result().StatusCode
	if respCode != http.StatusFound {
		t.Fatalf("expected redirect bug got HTTP status %d", respCode)
	}

	locationHeader := resp.Result().Header.Get("Location")
	if locationHeader != "/login/" {
		t.Errorf("expected location header `/login/` but got `%s`", locationHeader)
	}

	var sessionCookie *http.Cookie
	for _, cookie := range resp.Result().Cookies() {
		if cookie.Name == "session" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("no session cookie was unset by the logout handler")
	}

	if sessionCookie.Value != "" {
		t.Error("session cookie value was not set to empty string")
	}

	if sessionCookie.MaxAge >= 0 {
		t.Errorf(
			"session cookie is not expired, its max-mage is %d",
			sessionCookie.MaxAge,
		)
	}

	if sessionCookie.Path != "/" {
		t.Errorf(
			"expected session cookie path to be `/` but it was `%s`",
			sessionCookie.Path,
		)
	}

	if !sessionCookie.HttpOnly {
		t.Error("session cookie was not http-only")
	}
}
