package webserver

import (
	"fmt"
	"html/template"

	"github.com/gobuffalo/packr"
)

// Templates is a type which knows how to find and parse HTML templates
// by their name.
type Templates interface {

	// Get find and parses a template based on its file name.
	Get(path string) (*template.Template, error)
}

// PackrTemplates is Templates implementation which uses packr.Box to
// extract template data.
type PackrTemplates struct {
	box packr.Box
}

// Get implements Templates for the box in PackrTemplate.
func (t *PackrTemplates) Get(path string) (*template.Template, error) {
	tplString, err := t.box.MustString(path)
	if err != nil {
		return nil, fmt.Errorf("finding template in box: %s", err)
	}

	tpl := template.New(path)

	parsed, err := tpl.Parse(tplString)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %s", err)
	}

	return parsed, nil
}

// NewPackrTemplates returns a new PackrTemplates which will use the argument
// box for finding and reading files.
func NewPackrTemplates(box packr.Box) *PackrTemplates {
	return &PackrTemplates{
		box: box,
	}
}
