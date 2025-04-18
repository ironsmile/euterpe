package subsonic

import (
	"net/http"

	"github.com/ironsmile/euterpe/src/playlists"
)

func (s *subsonic) getPlaylists(w http.ResponseWriter, req *http.Request) {
	username := req.Form.Get("username")
	if username != "" && username != s.auth.User {
		resp := responseError(errCodeNotFound, "username not found")
		encodeResponse(w, req, resp)
		return
	}

	playlists, err := s.playlists.List(req.Context(), playlists.ListArgs{
		Offset: 0,
		Count:  0, // 0 means "all"
	})
	if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	resp := playlistsResponse{
		baseResponse: responseOk(),
	}

	for _, playlist := range playlists {
		resp.Playlists.Children = append(
			resp.Playlists.Children,
			toXsdPlaylist(playlist, s.auth.User),
		)
	}

	encodeResponse(w, req, resp)
}

type playlistsResponse struct {
	baseResponse

	Playlists xsdPlaylists `xml:"playlists" json:"playlists"`
}
