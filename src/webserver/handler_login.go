package webserver

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gbrlsnchs/jwt/v3"

	"github.com/ironsmile/euterpe/src/config"
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
	returnTo := r.URL.Query().Get(returnToQueryParam)
	if !strings.HasPrefix(returnTo, "/") {
		returnTo = "/"
	}

	user := r.PostFormValue("username")
	pass := r.PostFormValue("password")

	if !checkLoginCreds(user, pass, h.auth) {
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
	now := time.Now()

	if sessionCookie {
		expiresAt = now.Add(sessionTokenDuration)
	} else {
		expiresAt = now.Add(rememberMeDuration)
	}

	pl := jwt.Payload{
		IssuedAt:       jwt.NumericDate(now),
		ExpirationTime: jwt.NumericDate(expiresAt),
	}

	if len(h.auth.Secret) == 0 {
		errMessage := "Error generating JWT: secret is empty"
		http.Error(w, errMessage, http.StatusInternalServerError)
		return
	}

	token, err := jwt.Sign(pl, jwt.NewHS256([]byte(h.auth.Secret)))
	if err != nil {
		errMessage := fmt.Sprintf("Error generating JWT: %s.", err)
		http.Error(w, errMessage, http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    string(token),
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
