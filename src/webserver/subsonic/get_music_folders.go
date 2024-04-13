package subsonic

import (
	"net/http"
)

func (s *subsonic) getMusicFolders(w http.ResponseWriter, req *http.Request) {
	resp := musicFoldersResponse{
		baseResponse: responseOk(),
		MusicFolders: musicFolders{
			Children: []musicFolder{
				{
					ID:   combinedMusicFolderID,
					Name: "Combined Music Library",
				},
			},
		},
	}

	encodeResponse(w, req, resp)
}

type musicFoldersResponse struct {
	baseResponse

	MusicFolders musicFolders `xml:"musicFolders" json:"musicFolders"`
}

type musicFolders struct {
	Children []musicFolder `xml:"musicFolder" json:"musicFolder"`
}

type musicFolder struct {
	ID   int64  `xml:"id,attr" json:"id,string"`
	Name string `xml:"name,attr" json:"name"`
}
