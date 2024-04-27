package subsonic

import (
    "net/http"
)

func (s *subsonic) getVideoInfo(w http.ResponseWriter, req *http.Request) {
    resp := responseError(70, "video not found")
    encodeResponse(w, req, resp)
}
