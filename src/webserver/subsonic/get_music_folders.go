package subsonic

import (
	"encoding/xml"
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

	encodeResponse(w, resp)
}

type musicFoldersResponse struct {
	baseResponse

	MusicFolders musicFolders `xml:"musicFolders"`
}

type musicFolders struct {
	XMLName  xml.Name `xml:"musicFolders"`
	Children []musicFolder
}

type musicFolder struct {
	XMLName xml.Name `xml:"musicFolder"`
	ID      int64    `xml:"id,attr"`
	Name    string   `xml:"name,attr"`
}
