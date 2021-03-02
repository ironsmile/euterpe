package webserver

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
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

// FSTemplates is Templates implementation which uses fs.FS to
// extract template data.
type FSTemplates struct {
	fs fs.FS
}

// Get implements Templates for the box in FSTemplates.
func (t *FSTemplates) Get(path string) (*template.Template, error) {
	tpl := template.New(path)

	parsed, err := tpl.ParseFS(t.fs, path)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %s", err)
	}

	return parsed, nil
}

// All implements the Templates interface.
func (t *FSTemplates) All() (*AllTemplates, error) {
	layout, err := t.Get("layout.html")
	if err != nil {
		return nil, fmt.Errorf("parsing layout: %s", err)
	}

	pfh, err := t.fs.Open("player.html")
	if err != nil {
		return nil, fmt.Errorf("could not find player.html template: %s", err)
	}
	defer pfh.Close()
	tplContents, err := io.ReadAll(pfh)
	if err != nil {
		return nil, fmt.Errorf("error reading player.html: %s", err)
	}

	index := template.Must(layout.Clone())
	if _, err = index.New("content").Parse(string(tplContents)); err != nil {
		return nil, fmt.Errorf("finding index template: %s", err)
	}

	adfh, err := t.fs.Open("add_device.html")
	if err != nil {
		return nil, fmt.Errorf("could not find add_device.html template: %s", err)
	}
	defer adfh.Close()
	tplContents, err = io.ReadAll(adfh)
	if err != nil {
		return nil, fmt.Errorf("error reading add_device.html: %s", err)
	}

	addDevice := template.Must(layout.Clone())
	if _, err := addDevice.New("content").Parse(string(tplContents)); err != nil {
		return nil, fmt.Errorf("finding add_device template: %s", err)
	}

	return &AllTemplates{
		index:     index,
		addDevice: addDevice,
	}, nil
}

// NewFSTemplates returns a new FSTemplates which will use the argument
// fs.FS for finding and reading files.
func NewFSTemplates(fs fs.FS) *FSTemplates {
	return &FSTemplates{
		fs: fs,
	}
}

// AllTemplates is a structure which contains all parsed templates for different pages.
// They are ready for usage in http handlers which return HTML.
type AllTemplates struct {
	index     *template.Template
	addDevice *template.Template
}
