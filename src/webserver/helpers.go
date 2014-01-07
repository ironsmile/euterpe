package webserver

import (
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

// Returns the root directory in which HTML templates reside
func templateDir() (string, error) {
	gopath := os.ExpandEnv("$GOPATH")
	relPath := filepath.FromSlash("src/github.com/ironsmile/httpms/templates")
	for _, path := range strings.Split(gopath, ":") {
		tmplPath := filepath.Join(path, relPath)
		entry, err := os.Stat(tmplPath)
		if err != nil {
			continue
		}

		if !entry.IsDir() {
			return "", errors.New(fmt.Sprintf("%s is not a directory", tmplPath))
		}
		return tmplPath, nil
	}
	return "", errors.New("Template directory was not found")
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
