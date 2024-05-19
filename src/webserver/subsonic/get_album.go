package subsonic

import (
	"net/http"
	"strconv"
)

func (s *subsonic) getAlbum(w http.ResponseWriter, req *http.Request) {
	idString := req.Form.Get("id")
	subsonicID, err := strconv.ParseInt(idString, 10, 64)
	if idString == "" || err != nil || !isAlbumID(subsonicID) {
		resp := responseError(errCodeNotFound, "album not found")
		encodeResponse(w, req, resp)
		return
	}

	albumID := toAlbumDBID(subsonicID)

	album, err := s.lib.GetAlbum(req.Context(), albumID)
	if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	alEntry := xsdAlbumWithSongsID3{
		xsdAlbumID3: dbAlbumToAlbumID3Entry(album),
	}

	tracks := s.lib.GetAlbumFiles(req.Context(), albumID)
	for _, track := range tracks {
		alEntry.Children = append(alEntry.Children, trackToChild(
			track,
			s.getLastModified(),
		))
	}

	resp := albumResponse{
		baseResponse: responseOk(),
		Album:        alEntry,
	}

	encodeResponse(w, req, resp)
}

type albumResponse struct {
	baseResponse

	Album xsdAlbumWithSongsID3 `xml:"album" json:"album"`
}
