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

	// All returns a struct which contains all templates in
	// non-exported attributes.
	All() (*AllTemplates, error)
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

// All implements the Templates interface.
func (t *PackrTemplates) All() (*AllTemplates, error) {
	layout, err := t.Get("layout.html")
	if err != nil {
		return nil, fmt.Errorf("parsing layout: %s", err)
	}

	tplString, err := t.box.MustString("player.html")
	if err != nil {
		return nil, fmt.Errorf("finding index template in box: %s", err)
	}
	index := template.Must(layout.Clone())
	template.Must(index.New("content").Parse(tplString))

	tplString, err = t.box.MustString("add_device.html")
	if err != nil {
		return nil, fmt.Errorf("finding add_device template in box: %s", err)
	}
	addDevice := template.Must(layout.Clone())
	template.Must(addDevice.New("content").Parse(tplString))

	return &AllTemplates{
		index:     index,
		addDevice: addDevice,
	}, nil
}

// NewPackrTemplates returns a new PackrTemplates which will use the argument
// box for finding and reading files.
func NewPackrTemplates(box packr.Box) *PackrTemplates {
	return &PackrTemplates{
		box: box,
	}
}

// AllTemplates is a structure which contains all parsed templates for different pages.
// They are ready for usage in http handlers which return HTML.
type AllTemplates struct {
	index     *template.Template
	addDevice *template.Template
}
