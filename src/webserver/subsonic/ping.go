package subsonic

import (
	"net/http"
)

func (s *subsonic) apiPing(w http.ResponseWriter, req *http.Request) {
	encodeResponse(w, req, pingResponse{
		baseResponse: responseOk(),
	})
}

type pingResponse struct {
	baseResponse
}
