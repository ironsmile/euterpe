package subsonic

import (
	"net/http"
	"strconv"
)

func (s *subsonic) createPlaylist(w http.ResponseWriter, req *http.Request) {
	if playlistID := req.Form.Get("playlistId"); playlistID != "" {
		s.cretePlaylistUpdate(w, req)
	} else {
		s.cretePlaylistNew(w, req)
	}
}

func (s *subsonic) cretePlaylistNew(w http.ResponseWriter, req *http.Request) {
	name := req.Form.Get("name")
	if name == "" {
		resp := responseError(errCodeMissingParameter, "playlist name is required")
		encodeResponse(w, req, resp)
		return
	}

}

func (s *subsonic) cretePlaylistUpdate(w http.ResponseWriter, req *http.Request) {
	playlistID, err := strconv.ParseInt(req.Form.Get("playlistId"), 10, 64)
	if err != nil {
		resp := responseError(errCodeNotFound, "playlist not found")
		encodeResponse(w, req, resp)
		return
	}

	_ = playlistID
}
