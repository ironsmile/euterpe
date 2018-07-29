package webserver

import "net/http"

// NewLogoutHandler returns a handler which will logout the user form his HTTP
// session by unsetting his session cookie.
func NewLogoutHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie := &http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
		}
		http.SetCookie(w, cookie)

		w.Header().Set("Location", "/login/")
		w.WriteHeader(http.StatusFound)
	})
}
