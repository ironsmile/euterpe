package webserver

import (
	"fmt"
	"html/template"
	"log"
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

// HandlerFuncWithError is similar to http.HandlerFunc but returns an error when
// the handling of the request failed.
type HandlerFuncWithError func(http.ResponseWriter, *http.Request) error

// InternalErrorOnErrorHandler is used to wrap around handlers-like functions which just
// return error. This function actually writes the HTTP error and renders the error in
// the html.
func InternalErrorOnErrorHandler(writer http.ResponseWriter, req *http.Request,
	fnc HandlerFuncWithError) {
	withErrorHandling := WithInternalError(fnc)
	withErrorHandling(writer, req)
}

// WithInternalError converts HandlerFuncWithError to http.HandlerFunc by making sure
// all errors returned are flushed to the writer and Internal Server Error HTTP status
// is sent.
func WithInternalError(fnc HandlerFuncWithError) http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		err := fnc(writer, req)
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			if _, err := writer.Write([]byte(err.Error())); err != nil {
				log.Printf("error writing body in InternalErrorHandler: %s", err)
			}
		}
	}
}
