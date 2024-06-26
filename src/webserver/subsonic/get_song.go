package subsonic

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) getSong(w http.ResponseWriter, req *http.Request) {
	idString := req.Form.Get("id")
	subsonicID, err := strconv.ParseInt(idString, 10, 64)
	if idString == "" || err != nil || !isTrackID(subsonicID) {
		resp := responseError(errCodeNotFound, "song not found")
		encodeResponse(w, req, resp)
		return
	}

	trackID := toTrackDBID(subsonicID)

	track, err := s.lib.GetTrack(req.Context(), trackID)
	if errors.Is(err, library.ErrNotFound) {
		resp := responseError(errCodeNotFound, "song not found")
		encodeResponse(w, req, resp)
		return
	}

	resp := songResponse{
		baseResponse: responseOk(),
		Song:         trackToChild(track, s.lastModified),
	}

	encodeResponse(w, req, resp)
}

type songResponse struct {
	baseResponse

	Song xsdChild `xml:"song" json:"song"`
}
