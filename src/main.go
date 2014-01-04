package src

import "net/http"
import "log"
import "time"
import "fmt"
import "os"
import "html/template"

type SearchHandler struct{}

func (sh SearchHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {

	fullPath := fmt.Sprintf("%s?%s", req.URL.Path, req.URL.RawQuery)

	templateDir := fmt.Sprintf("%s/src/github.com/ironsmile/httpms/templates",
		os.ExpandEnv("$GOPATH"))
	templateFile := fmt.Sprintf("%s/test2.html", templateDir)

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

func Main() {

	http.Handle("/", http.FileServer(http.Dir("/home/iron4o/workspace")))
	http.Handle("/search/", SearchHandler{})

	s := &http.Server{
		Addr:           ":8080",
		Handler:        nil,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
