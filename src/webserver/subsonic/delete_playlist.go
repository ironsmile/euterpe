package subsonic

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ironsmile/euterpe/src/playlists"
)

func (s *subsonic) deletePlaylist(w http.ResponseWriter, req *http.Request) {
	id, err := strconv.ParseInt(req.Form.Get("id"), 10, 64)
	if err != nil && req.Form.Get("id") == "" {
		resp := responseError(errCodeMissingParameter, "playlist ID is missing in query")
		encodeResponse(w, req, resp)
		return
	} else if err != nil {
		resp := responseError(errCodeNotFound, "playlist not found")
		encodeResponse(w, req, resp)
		return
	}

	err = s.playlists.Delete(req.Context(), id)
	if err != nil && errors.Is(err, playlists.ErrNotFound) {
		resp := responseError(errCodeNotFound, "playlist not found")
		encodeResponse(w, req, resp)
		return
	} else if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	encodeResponse(w, req, responseOk())
}
