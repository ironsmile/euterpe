package webserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gbrlsnchs/jwt/v3"

	"github.com/ironsmile/euterpe/src/config"
)

const (
	wrongLoginText = "wrong username or password"
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	reqBody := struct {
		User string `json:"username"`
		Pass string `json:"password"`
	}{}

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&reqBody); err != nil {
		respondWithJSONError(
			w,
			http.StatusBadRequest,
			"Error parsing JSON request: %s.",
			err,
		)
		return
	}

	if !checkLoginCreds(reqBody.User, reqBody.Pass, h.auth) {
		respondWithJSONError(w, http.StatusUnauthorized, wrongLoginText)
		return
	}

	now := time.Now()
	pl := jwt.Payload{
		IssuedAt:       jwt.NumericDate(now),
		ExpirationTime: jwt.NumericDate(time.Now().Add(rememberMeDuration)),
	}

	if len(h.auth.Secret) == 0 {
		respondWithJSONError(
			w,
			http.StatusInternalServerError,
			"Error generating JWT: secret is empty.",
		)
		return
	}

	token, err := jwt.Sign(pl, jwt.NewHS256([]byte(h.auth.Secret)))
	if err != nil {
		respondWithJSONError(
			w,
			http.StatusInternalServerError,
			"Error generating JWT: %s.",
			err,
		)
		return
	}

	enc := json.NewEncoder(w)
	err = enc.Encode(&struct {
		Token string `json:"token"`
	}{
		Token: string(token),
	})

	if err != nil {
		respondWithJSONError(
			w,
			http.StatusInternalServerError,
			"Error writing token response: %s.",
			err,
		)
		return
	}
}

func respondWithJSONError(
	w http.ResponseWriter,
	code int,
	msgf string,
	args ...interface{},
) {
	resp := struct {
		Error string `json:"error"`
	}{
		Error: fmt.Sprintf(msgf, args...),
	}

	enc := json.NewEncoder(w)

	w.WriteHeader(code)
	_ = enc.Encode(resp)
}
