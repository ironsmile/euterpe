package subsonic

import (
    "net/http"
)

func (s *subsonic) getGenres(w http.ResponseWriter, req *http.Request) {
    resp := genresResponse{
        baseResponse: responseOk(),
    }

    encodeResponse(w, req, resp)
}

type genresResponse struct {
    baseResponse

    Genres struct{} `xml:"genres" json:"genres"`
}
