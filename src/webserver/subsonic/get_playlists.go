package subsonic

import (
	"net/http"
)

func (s *subsonic) getPlaylists(w http.ResponseWriter, req *http.Request) {
	username := req.Form.Get("username")
	if username != "" && username != s.auth.User {
		resp := responseError(errCodeNotFound, "username not found")
		encodeResponse(w, req, resp)
		return
	}

	playlists, err := s.playlists.GetAll(req.Context())
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
