package webserver

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
)

// Handler wrapper
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
	writer.Header().Set("WWW-Authenticate", `Basic realm="HTTPMS"`)
	writer.WriteHeader(http.StatusUnauthorized)
	writer.Write([]byte("Authentication required"))
}

// Compares the authentication header with the stored user and passwords
// and returns true if they pass
func (hl BasicAuthHandler) authenticate(auth string) bool {
	return false
}

// Handler responsible for search requests. It will use the Library to
// return a list of matched files to the interface.
type SearchHandler struct{}

// This method is required by the http.Handler's interface
func (sh SearchHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {

	fullPath := fmt.Sprintf("%s?%s", req.URL.Path, req.URL.RawQuery)

	templateDir := fmt.Sprintf("%s/src/github.com/ironsmile/httpms/templates",
		os.ExpandEnv("$GOPATH"))
	templateFile := fmt.Sprintf("%s/test.html", templateDir)

	_, err := os.Stat(templateFile)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(fmt.Sprintf("%s does not exist", templateFile)))
		return
	}

	tmpl, err := template.ParseFiles(templateFile)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(fmt.Sprintf("Error parsing template %s", templateFile)))
		return
	}

	variables := map[string]string{
		"FullPath":     fullPath,
		"TemplateFile": templateFile,
	}

	err = tmpl.Execute(writer, variables)

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte("Error executing template"))
		return
	}
}
