package subsonic

import (
	"net/http"
	"strconv"
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

	albums := s.lib.GetArtistAlbums(req.Context(), artistID)
	if len(albums) == 0 {
		resp := responseError(errCodeNotFound, "artist not found")
		encodeResponse(w, req, resp)
		return
	}

	baseArtistInfo := s.getArtistInfoBase(req, artistID)
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
