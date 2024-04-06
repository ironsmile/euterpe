package subsonic

import (
	"net/http"
)

func (s *subsonic) apiPing(w http.ResponseWriter, _ *http.Request) {
	encodeResponse(w, responseOk())
}
