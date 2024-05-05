package subsonic

import (
	"net/http"
	"strconv"
)

func (s *subsonic) getArtist(w http.ResponseWriter, req *http.Request) {
	idString := req.Form.Get("id")
	subsonicID, err := strconv.ParseInt(idString, 10, 64)
	if idString == "" || err != nil || !isArtistID(subsonicID) {
		resp := responseError(errCodeNotFound, "artist not found")
		encodeResponse(w, req, resp)
		return
	}

	artistID := toArtistDBID(subsonicID)

	entry, err := s.getArtistDirectory(req, artistID)
	if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	artEtr := xsdArtistWithAlbumsID3{
		xsdArtistID3: directoryToArtistID3(entry),
	}

	for _, child := range entry.Children {
		artEtr.Children = append(artEtr.Children, toAlbumID3Entry(child))
	}

	resp := artistResponse{
		baseResponse: responseOk(),
		Artist:       artEtr,
	}

	encodeResponse(w, req, resp)
}

type artistResponse struct {
	baseResponse

	Artist xsdArtistWithAlbumsID3 `xml:"artist" json:"artist"`
}
