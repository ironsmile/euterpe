package subsonic

import "net/http"

func (s *subsonic) getUser(w http.ResponseWriter, req *http.Request) {
	username := req.Form.Get("username")
	if username == "" {
		resp := responseError(errCodeMissingParameter, "missing username parameter")
		encodeResponse(w, req, resp)
		return
	}

	if username != s.auth.User {
		resp := responseError(errCodeNotFound, "user not found")
		encodeResponse(w, req, resp)
		return
	}

	resp := getUserResponse{
		baseResponse: responseOk(),

		User: xsdUser{
			Username:     s.auth.User,
			Scrobbling:   true,
			AdminRole:    true,
			SettingsRole: true,
			DownloadRole: true,
			UploadRole:   true,
			PlaylistRole: true,
			CoverArtRole: true,
			CommentRole:  true,
			PodcastRole:  true,
			StreamRole:   true,
			JukeboxRole:  true,
			ShareRole:    true,
			Folders: []int64{
				combinedMusicFolderID,
			},
		},
	}

	encodeResponse(w, req, resp)
}

type getUserResponse struct {
	baseResponse

	User xsdUser `xml:"user" json:"user"`
}
