package subsonic

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) getArtistInfo(w http.ResponseWriter, req *http.Request) {
	idString := req.Form.Get("id")
	subsonicID, err := strconv.ParseInt(idString, 10, 64)
	if idString == "" || err != nil || !isArtistID(subsonicID) {
		resp := responseError(errCodeNotFound, "artist not found")
		encodeResponse(w, req, resp)
		return
	}

	artistID := toArtistDBID(subsonicID)

	artist, err := s.lib.GetArtist(req.Context(), artistID)
	if errors.Is(err, library.ErrArtistNotFound) {
		resp := responseError(errCodeNotFound, "artist not found")
		encodeResponse(w, req, resp)
		return
	} else if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	baseArtistInfo := s.getArtistInfoBase(req, artist)
	resp := artistInfoResponse{
		baseResponse: baseArtistInfo.baseResponse,
		ArtistInfo:   baseArtistInfo.ArtistInfo2,
	}

	encodeResponse(w, req, resp)
}

type artistInfoResponse struct {
	baseResponse

	// ArtistInfo reuses the aristInfo2Element element since for the moment
	// it is exactly the same as what would've been the artistInfoElement.
	ArtistInfo xsdArtistInfoBase `xml:"artistInfo"`
}
