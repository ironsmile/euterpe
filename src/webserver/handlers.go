package webserver

import (
	"encoding/base64"
	"fmt"
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

	if err == false || len(auth) != 1 {
		hl.challengeAuthentication(writer)
		return
	}

	if hl.authenticate(auth[0]) == false {
		hl.challengeAuthentication(writer)
		return
	}

	hl.wrapped.ServeHTTP(writer, req)
}

// Sends 401 and authentication challenge in the writer
func (hl BasicAuthHandler) challengeAuthentication(writer http.ResponseWriter) {
	tmpl, err := getTemplate("unauthorized.html")

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(err.Error()))
		return
	}

	writer.Header().Set("WWW-Authenticate", `Basic realm="HTTPMS"`)
	writer.WriteHeader(http.StatusUnauthorized)

	err = tmpl.Execute(writer, nil)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(err.Error()))
		return
	}
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

// Handler responsible for search requests. It will use the Library to
// return a list of matched files to the interface.
type SearchHandler struct{}

// This method is required by the http.Handler's interface
func (sh SearchHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {

	fullPath := fmt.Sprintf("%s?%s", req.URL.Path, req.URL.RawQuery)

	tmpl, err := getTemplate("test.html")

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(err.Error()))
		return
	}

	variables := map[string]string{
		"FullPath":     fullPath,
		"TemplateFile": "test.html",
	}

	err = tmpl.Execute(writer, variables)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(err.Error()))
		return
	}
}
