package webserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ironsmile/euterpe/src/version"
	"github.com/ironsmile/euterpe/src/webserver/webutils"
)

type aboutHandler struct {
	resp aboutResponse
}

// NewAboutHandler returns the HTTP handler which shows a JSON with information
// about the server.
func NewAboutHandler() http.Handler {
	return &aboutHandler{
		resp: aboutResponse{
			ServerVersion: version.Version,
		},
	}
}

func (h *aboutHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	if err := enc.Encode(h.resp); err != nil {
		msg := fmt.Sprintf("Failed to encode JSON response: %s", err)
		webutils.JSONError(w, msg, http.StatusInternalServerError)
		return
	}
}

type aboutResponse struct {
	ServerVersion string `json:"server_version"`
}
