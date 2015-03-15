/*
   This HTTP handler echoes Terry's name.

       "A man is not dead while his name is still spoken."

   As proposed in in http://www.gnuterrypratchett.com/
*/

package webserver

import (
	"net/http"
)

// TerryHandler adds the X-Clacks-Overhead header. It wraps around the actual handler.
type TerryHandler struct {
	wrapped http.Handler
}

// ServeHTTP satisfies the http.Handler interface.
func (th TerryHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Set("X-Clacks-Overhead", "GNU Terry Pratchett")
	th.wrapped.ServeHTTP(writer, req)
}

// NewTerryHandler returns a new TerryHandler, ready for use.
func NewTerryHandler(handler http.Handler) http.Handler {
	return &TerryHandler{wrapped: handler}
}
