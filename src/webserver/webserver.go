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

type ServerConfig struct {
	Address   string
	Root      string
	SSL       bool
	SSLCert   string
	SSLKey    string
	WaitGroup *sync.WaitGroup
}

type SearchHandler struct{}

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
