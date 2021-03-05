package webserver

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
)

// NewTemplateHandler returns a handler which will execute the page template inside
// the layout template.
func NewTemplateHandler(tpl *template.Template, title string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Title string
			Req   *http.Request
			Menu  []menu
		}{
			Title: title,
			Req:   r,
			Menu: []menu{
				{
					Name:   "Player",
					URI:    "/",
					Active: r.URL.Path == "/",
				},
				{
					Name:   "Add Device",
					URI:    "/add_device/",
					Active: r.URL.Path == "/add_device/",
				},
			},
		}
		if err := tpl.Execute(w, data); err != nil {
			errorMessage := fmt.Sprintf("Error executing template: %s.\n", err)
			log.Print(errorMessage)
			http.Error(w, errorMessage, http.StatusInternalServerError)
		}
	})
}

type menu struct {
	URI    string
	Name   string
	Active bool
}
