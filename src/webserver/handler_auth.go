package webserver

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gbrlsnchs/jwt/v3"
	"github.com/ironsmile/euterpe/src/config"
)

const (
	authRequiredJSON = `{"error": "authentication required"}`
)

// AuthHandler is a handler wrapper used for authentication. Its only job is
// to do the authentication and then pass the work to the Handler it wraps around.
// Possible methods for authentication:
//
//  * Basic Auth with the username and password
//  * Authorization Bearer JWT token
//  * JWT token in a session cookie
//  * JWT token as a query string
//
// Basic auth is preserved for backward compatibility. Needless to say, it so not
// a preferred method for authentication.
type AuthHandler struct {
	wrapped    http.Handler // The actual handler that does the APP Logic job
	username   string       // Username to be used for basic authenticate
	password   string       // Password to be used for basic authenticate
	templates  Templates    // Template finder
	secret     string       // Secret used to craft and decode tokens
	exceptions []string     // Paths which will be exempt from authentication
}

// NewAuthHandler returns a new AuthHandler.
func NewAuthHandler(
	wrapped http.Handler,
	username string,
	password string,
	templatesResolver Templates,
	secret string,
	exceptions []string,
) *AuthHandler {
	return &AuthHandler{
		wrapped:    wrapped,
		username:   username,
		password:   password,
		templates:  templatesResolver,
		secret:     secret,
		exceptions: exceptions,
	}
}

// ServeHTTP implements the http.Handler interface and does the actual basic authenticate
// check for every request
func (hl *AuthHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	if !hl.authenticated(req) {
		InternalErrorOnErrorHandler(writer, req, hl.challengeAuthentication)
		return
	}

	hl.wrapped.ServeHTTP(writer, req)
}

// Sends 401 and authentication challenge in the writer
func (hl *AuthHandler) challengeAuthentication(
	writer http.ResponseWriter,
	req *http.Request,
) error {
	accepts := strings.Split(req.Header.Get("Accept"), ",")

	if contains(accepts, "text/html") {
		return hl.redirectToLogin(writer, req)
	}

	if contains(accepts, "application/json") {
		writer.Header().Set("Content-Type", "application/json; charset=utf8")
		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = writer.Write([]byte(authRequiredJSON))
		return nil
	}

	return hl.basicAuthChallenge(writer, req)
}

func (hl *AuthHandler) basicAuthChallenge(w http.ResponseWriter, r *http.Request) error {
	tmpl, err := hl.templates.Get("unauthorized.html")
	if err != nil {
		return err
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Euterpe Music Server"`)
	w.WriteHeader(http.StatusUnauthorized)

	return tmpl.Execute(w, nil)
}

func (hl *AuthHandler) redirectToLogin(w http.ResponseWriter, r *http.Request) error {
	returnTo := r.URL.Query().Get(returnToQueryParam)
	if returnTo == "" {
		returnTo = r.RequestURI
	}

	query := url.Values{}
	query.Set(returnToQueryParam, returnTo)

	w.Header().Set("Location", fmt.Sprintf("/login/?%s", query.Encode()))
	w.WriteHeader(http.StatusFound)

	return nil
}

// Compares the authentication header with the stored user and passwords
// and returns true if they pass.
func (hl *AuthHandler) authenticated(r *http.Request) bool {
	for _, path := range hl.exceptions {
		if strings.HasPrefix(r.URL.Path, path) {
			return true
		}
	}

	authHeader := r.Header.Get("Authorization")

	if strings.HasPrefix(authHeader, "Bearer ") {
		return hl.withJWT(strings.TrimPrefix(authHeader, "Bearer "))
	}

	if strings.HasPrefix(authHeader, "Basic ") {
		return hl.withBasicAuth(strings.TrimPrefix(authHeader, "Basic "))
	}

	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		return hl.withJWT(cookie.Value)
	}

	if queryToken := r.URL.Query().Get("token"); queryToken != "" {
		return hl.withJWT(queryToken)
	}

	return false
}

func (hl *AuthHandler) withBasicAuth(encoded string) bool {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return false
	}

	pair := strings.SplitN(string(b), ":", 2)

	if len(pair) != 2 {
		return false
	}

	cfg := config.Auth{
		User:     hl.username,
		Password: hl.password,
	}

	return checkLoginCreds(pair[0], pair[1], cfg)
}

func (hl *AuthHandler) withJWT(token string) bool {
	var jot jwt.Payload

	alg := jwt.NewHS256([]byte(hl.secret))
	exp := jwt.ExpirationTimeValidator(time.Now())
	validatePayload := jwt.ValidatePayload(&jot, exp)

	_, err := jwt.Verify([]byte(token), alg, &jot, validatePayload)
	return err == nil
}

func contains(haystack []string, needle string) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}

	return false
}
