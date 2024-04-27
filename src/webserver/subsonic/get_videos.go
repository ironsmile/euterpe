package subsonic

import (
    "net/http"
)

func (s *subsonic) getVideos(w http.ResponseWriter, req *http.Request) {
    resp := videosResponse{
        baseResponse: responseOk(),
    }

    encodeResponse(w, req, resp)
}

type videosResponse struct {
    baseResponse

    Videos struct{} `xml:"videos" json:"videos"`
}
