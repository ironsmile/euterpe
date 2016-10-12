package webserver

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ironsmile/httpms/src/helpers"
)

// Returns the root directory in which HTML templates reside
func templateDir() (string, error) {
	projRoot, err := helpers.ProjectRoot()

	if err != nil {
		return "", err
	}

	theDirectory := filepath.Join(projRoot, "templates")

	st, err := os.Stat(theDirectory)

	if err != nil {
		return "", err
	}

	if !st.IsDir() {
		return "", fmt.Errorf("%s is not a directory", theDirectory)
	}

	return theDirectory, nil
}

// Returns a *template.Template 'object' by its file name
func getTemplate(templateFileName string) (*template.Template, error) {
	templateDir, err := templateDir()

	if err != nil {
		return nil, err
	}

	templateFile := filepath.Join(templateDir, templateFileName)

	_, err = os.Stat(templateFile)

	if err != nil {
		return nil, err
	}

	tmpl, err := template.ParseFiles(templateFile)

	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

// Used to wrap around handlers-like functions which just return error.
// This function actually writes the HTTP error and renders the error in the html
func InternalErrorOnErrorHandler(writer http.ResponseWriter, req *http.Request,
	fnc func(http.ResponseWriter, *http.Request) error) {
	err := fnc(writer, req)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(err.Error()))
	}
}
