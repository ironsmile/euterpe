// This module contains the webserver whcih deals with processing requests
// from the user, presenting him with the interface of the application.
package webserver

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// The configuration which should be supplied to the webserver
type ServerConfig struct {
	Address   string          // Address on which the server will listen. See http/Server.Addr
	Root      string          // The http root directory containing the interface files
	SSL       bool            // Should it use SSL when serving
	SSLCert   string          // The SSL certificate. Only makes sens if SSL is true
	SSLKey    string          // The SSL key. Only makes sense if SSL is true
	WaitGroup *sync.WaitGroup // Should someone needs to sych with the server's stop
	Auth      bool            // Should the server require HTTP auth
	AuthUser  string          // HTTP basic authenticate username
	AuthPass  string          // HTTP basic authenticate password
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

// The function that actually starts the webserver. It attaches all the handlers
// and starts the webserver while consulting the ServerConfig supplied.
func Serve(cfg ServerConfig) {

	http.Handle("/", http.FileServer(http.Dir(cfg.Root)))
	http.Handle("/search/", SearchHandler{})

	s := &http.Server{
		Addr:           cfg.Address,
		Handler:        nil,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if cfg.SSL {
		log.Fatal(s.ListenAndServeTLS(cfg.SSLCert, cfg.SSLKey))
	} else {
		log.Fatal(s.ListenAndServe())
	}

	cfg.WaitGroup.Done()
}
