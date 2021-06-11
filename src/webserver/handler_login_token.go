package webserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gbrlsnchs/jwt"

	"github.com/ironsmile/euterpe/src/config"
)

const (
	wrongLoginJSON = `{"error": "wrong username or password"}`
)

type loginTokenHandler struct {
	auth config.Auth
}

// NewLoginTokenHandler returns a new login handler which will use the information in
// auth for deciding when device or program was logged in correctly by entering
// username and password.
func NewLoginTokenHandler(auth config.Auth) http.Handler {
	return &loginTokenHandler{
		auth: auth,
	}
}

func (h *loginTokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf8")

	reqBody := struct {
		User string `json:"username"`
		Pass string `json:"password"`
	}{}

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&reqBody); err != nil {
		errMessage := fmt.Sprintf("Error parsing JSON request: %s.", err)
		http.Error(w, errMessage, http.StatusBadRequest)
		return
	}

	if !checkLoginCreds(reqBody.User, reqBody.Pass, h.auth) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(wrongLoginJSON))
		return
	}

	tokenOpts := &jwt.Options{
		Timestamp:      true,
		ExpirationTime: time.Now().Add(rememberMeDuration),
	}
	token, err := jwt.Sign(jwt.HS256(h.auth.Secret), tokenOpts)
	if err != nil {
		errMessage := fmt.Sprintf("Error generating JWT: %s.", err)
		http.Error(w, errMessage, http.StatusInternalServerError)
		return
	}

	enc := json.NewEncoder(w)
	err = enc.Encode(&struct {
		Token string `json:"token"`
	}{
		Token: token,
	})

	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf8")
		errMessage := fmt.Sprintf("Error writing token response: %s.", err)
		http.Error(w, errMessage, http.StatusInternalServerError)
		return
	}
}
