package webserver

import (
	"errors"
	"fmt"
	"html/template"
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
		return "", errors.New(fmt.Sprintf("%s is not a directory", theDirectory))
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
