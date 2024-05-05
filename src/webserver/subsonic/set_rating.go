package subsonic

import (
	"net/http"
	"strconv"
)

func (s *subsonic) setRating(w http.ResponseWriter, req *http.Request) {
	id, err := strconv.ParseInt(req.Form.Get("id"), 10, 64)
	if err != nil {
		resp := responseError(
			errCodeMissingParameter,
			"Bad parameter `id`. It must be an integer.",
		)
		encodeResponse(w, req, resp)
		return
	}
	rating, err := strconv.ParseInt(req.Form.Get("rating"), 10, 8)
	if err != nil || rating < 0 || rating > 5 {
		resp := responseError(
			errCodeMissingParameter,
			"Bad parameter `rating`. It must be an integer in [0-5] range.",
		)
		encodeResponse(w, req, resp)
		return
	}

	if isTrackID(id) {
		s.setRatingTrack(w, req, toTrackDBID(id), uint8(rating))
	} else if isAlbumID(id) {
		s.setRatingAlbum(w, req, toAlbumDBID(id), uint8(rating))
	} else if isArtistID(id) {
		s.setRatingArtist(w, req, toArtistDBID(id), uint8(rating))
	} else {
		resp := responseError(
			errCodeNotFound,
			"Nothing with this ID was found",
		)
		encodeResponse(w, req, resp)
	}
}

func (s *subsonic) setRatingTrack(
	w http.ResponseWriter,
	req *http.Request,
	trackID int64, // database ID
	rating uint8,
) {
	err := s.lib.SetTrackRating(req.Context(), trackID, rating)
	if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	encodeResponse(w, req, responseOk())
}

func (s *subsonic) setRatingAlbum(
	w http.ResponseWriter,
	req *http.Request,
	albumID int64, // database ID
	rating uint8,
) {
	err := s.lib.SetAlbumRating(req.Context(), albumID, rating)
	if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	encodeResponse(w, req, responseOk())
}

func (s *subsonic) setRatingArtist(
	w http.ResponseWriter,
	req *http.Request,
	artistID int64, // database ID
	rating uint8,
) {
	err := s.lib.SetArtistRating(req.Context(), artistID, rating)
	if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	encodeResponse(w, req, responseOk())
}
