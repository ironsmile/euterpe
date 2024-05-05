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

	entry, err := s.getAlbumDirectory(req, albumID)
	if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	artEtr := xsdAlbumWithSongsID3{
		xsdAlbumID3: xsdAlbumID3{
			ID:         entry.ID,
			Artist:     entry.Artist,
			ArtistID:   entry.ParentID,
			Name:       entry.Name,
			SongCount:  entry.SongCount,
			CoverArtID: entry.CoverArtID,
			Created:    s.lastModified,
			Duration:   entry.Duration,
			PlayCount:  entry.PlayCount,
			Starred:    entry.Starred,
		},
		Children: entry.Children,
	}

	resp := albumResponse{
		baseResponse: responseOk(),
		Album:        artEtr,
	}

	encodeResponse(w, req, resp)
}

type albumResponse struct {
	baseResponse

	Album xsdAlbumWithSongsID3 `xml:"album" json:"album"`
}
