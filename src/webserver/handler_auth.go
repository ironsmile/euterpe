package webserver

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gbrlsnchs/jwt"
)

const (
	authRequiredJSON = `{"error": "authentication required"}`
)

// AuthHandler is a handler wrapper used for authenticaation. Its only job is
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

	w.Header().Set("WWW-Authenticate", `Basic realm="HTTPMS"`)
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

	return pair[0] == hl.username && pair[1] == hl.password
}

func (hl *AuthHandler) withJWT(token string) bool {
	jot, err := jwt.FromString(token)
	if err != nil {
		return false
	}

	if err := jot.Verify(jwt.HS256(hl.secret)); err != nil {
		return false
	}

	alg := jwt.AlgorithmValidator(jwt.MethodHS256)
	exp := jwt.ExpirationTimeValidator(time.Now())

	if err := jot.Validate(alg, exp); err != nil {
		return false
	}

	return true
}

func contains(haystack []string, needle string) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}

	return false
}
