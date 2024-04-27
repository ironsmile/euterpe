package subsonic

import (
    "net/http"
)

func (s *subsonic) getVideoInfo(w http.ResponseWriter, req *http.Request) {
    resp := responseError(errCodeNotFound, "video not found")
    encodeResponse(w, req, resp)
}
