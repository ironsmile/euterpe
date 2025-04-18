package webutils

import (
	"encoding/json"
	"log"
	"net/http"
)

// JSONError writes a JSON object with an error message and sets the HTTP status code.
func JSONError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(http.StatusBadRequest)
	resp := jsonErrorMessage{
		Error: message,
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(&resp); err != nil {
		log.Printf("error writing body in browse handler: %s", err)
	}
}

type jsonErrorMessage struct {
	Error string `json:"error"`
}
