package webserver

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gbrlsnchs/jwt"

	"github.com/ironsmile/httpms/src/config"
)

var (
	// Session cookie is when the user has *not* clicked the "remember me" check box.
	// It will expire with the browser session. Remember me cookie (!sessionCookie)
	// on the other hand should live for far longer. We set the session token
	// optimistically to expire in seven days. Hopefully the session will not be that
	// long. The remember me cookie on the other hand gets to live for longer.
	sessionTokenDuration = 7 * 24 * time.Hour
	rememberMeDuration   = 62 * 24 * time.Hour
)

type loginHandler struct {
	auth config.Auth
}

// NewLoginHandler returns a new login handler which will use the information in
// auth for deciding when user has logged in correctly and also for generating
// tokens.
func NewLoginHandler(auth config.Auth) http.Handler {
	return &loginHandler{
		auth: auth,
	}
}

func (h *loginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := r.PostFormValue("username")
	pass := r.PostFormValue("password")
	returnTo := r.URL.Query().Get(returnToQueryParam)
	if !strings.HasPrefix(returnTo, "/") {
		returnTo = "/"
	}

	// The following check is carefully orchestrated so that it will take constant
	// time for wrong and correct pairs of username and password. This mitigates
	// simple timing attacks.
	userCheck := subtle.ConstantTimeCompare([]byte(user), []byte(h.auth.User))
	passCheck := subtle.ConstantTimeCompare([]byte(pass), []byte(h.auth.Password))

	if userCheck&passCheck != 1 {
		h.respondWrong(w, r, returnTo)
		return
	}

	h.respondCorrect(w, r, returnTo)
}

func (h *loginHandler) respondWrong(
	w http.ResponseWriter,
	r *http.Request,
	returnTo string,
) {
	query := url.Values{}
	query.Set(returnToQueryParam, returnTo)
	query.Set("wrongCreds", "1")

	w.Header().Set("Location", fmt.Sprintf("/login/?%s", query.Encode()))
	w.WriteHeader(http.StatusFound)
}

func (h *loginHandler) respondCorrect(
	w http.ResponseWriter,
	r *http.Request,
	returnTo string,
) {
	sessionCookie := true

	if r.PostFormValue("remember_me") == "on" {
		sessionCookie = false
	}

	var expiresAt time.Time

	if sessionCookie {
		expiresAt = time.Now().Add(sessionTokenDuration)
	} else {
		expiresAt = time.Now().Add(rememberMeDuration)
	}

	tokenOpts := &jwt.Options{
		Timestamp:      true,
		ExpirationTime: expiresAt,
	}
	token, err := jwt.Sign(jwt.HS256(h.auth.Secret), tokenOpts)
	if err != nil {
		errMessage := fmt.Sprintf("Error generating JWT: %s.", err)
		http.Error(w, errMessage, http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
	}
	if !sessionCookie {
		cookie.Expires = expiresAt
	}
	http.SetCookie(w, cookie)

	w.Header().Set("Location", returnTo)
	w.WriteHeader(http.StatusFound)
}
