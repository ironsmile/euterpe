package subsonic

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ironsmile/euterpe/src/playlists"
)

func (s *subsonic) getPlaylist(w http.ResponseWriter, req *http.Request) {
	idString := req.Form.Get("id")
	if idString == "" {
		resp := responseError(errCodeMissingParameter, "playlist ID is required")
		encodeResponse(w, req, resp)
		return
	}

	playlistID, err := strconv.ParseInt(idString, 10, 64)
	if err != nil {
		resp := responseError(errCodeNotFound, "playlist not found")
		encodeResponse(w, req, resp)
		return
	}

	playlist, err := s.playlists.Get(req.Context(), playlistID)
	if err != nil && errors.Is(err, playlists.ErrNotFound) {
		resp := responseError(errCodeNotFound, "playlist not found")
		encodeResponse(w, req, resp)
		return
	} else if err != nil {
		resp := responseError(
			errCodeGeneric,
			fmt.Sprintf("failed to get created playlist: %s", err),
		)
		encodeResponse(w, req, resp)
		return
	}

	resp := playlistWithSongsResponse{
		baseResponse: responseOk(),
		Playlist:     toXsdPlaylistWithSongs(playlist, s.auth.User, s.lastModified),
	}

	encodeResponse(w, req, resp)
}
