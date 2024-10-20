package webserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ironsmile/euterpe/src/version"
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

func (h *aboutHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Add("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(writer)
	if err := enc.Encode(h.resp); err != nil {
		writer.Header().Set("Content-Type", "plain/text; charset=utf-8")
		msg := fmt.Sprintf("Failed to encode JSON response: %s", err)
		http.Error(writer, msg, http.StatusInternalServerError)
		return
	}
}

type aboutResponse struct {
	ServerVersion string `json:"server_version"`
}
