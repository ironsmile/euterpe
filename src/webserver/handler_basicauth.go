package webserver

import (
	"encoding/base64"
	"net/http"
	"strings"
)

// Handler wrapper used for basic authenticate. Its only job is to do the
// authentication and then pass the work to the Handler it wraps around
type BasicAuthHandler struct {
	wrapped  http.Handler // The actual handler that does the APP Logic job
	username string       // Username to be used for basic authenticate
	password string       // Password to be used for basic authenticate
}

// Implements the http.Handler interface and does the actual basic authenticate
// check for every request
func (hl BasicAuthHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	auth, err := req.Header["Authorization"]

	if err == false || len(auth) < 1 || hl.authenticate(auth[0]) == false {
		InternalErrorOnErrorHandler(writer, req, hl.challengeAuthentication)
		return
	}

	hl.wrapped.ServeHTTP(writer, req)
}

// Sends 401 and authentication challenge in the writer
func (hl BasicAuthHandler) challengeAuthentication(writer http.ResponseWriter,
	req *http.Request) error {
	tmpl, err := getTemplate("unauthorized.html")

	if err != nil {
		return err
	}

	writer.Header().Set("WWW-Authenticate", `Basic realm="HTTPMS"`)
	writer.WriteHeader(http.StatusUnauthorized)

	err = tmpl.Execute(writer, nil)

	if err != nil {
		return err
	}

	return nil
}

// Compares the authentication header with the stored user and passwords
// and returns true if they pass.
func (hl BasicAuthHandler) authenticate(auth string) bool {

	s := strings.SplitN(auth, " ", 2)

	if len(s) != 2 || s[0] != "Basic" {
		return false
	}

	b, err := base64.StdEncoding.DecodeString(s[1])

	if err != nil {
		return false
	}

	pair := strings.SplitN(string(b), ":", 2)

	if len(pair) != 2 {
		return false
	}

	return pair[0] == hl.username && pair[1] == hl.password
}
