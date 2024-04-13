package subsonic

import (
	"net/http"
	"strconv"
)

func (s *subsonic) getArtist(w http.ResponseWriter, req *http.Request) {
	idString := req.URL.Query().Get("id")
	subsonicID, err := strconv.ParseInt(idString, 10, 64)
	if idString == "" || err != nil || !isArtistID(subsonicID) {
		resp := responseError(70, "artist not found")
		encodeResponse(w, req, resp)
		return
	}

	artistID := toArtistDBID(subsonicID)

	entry, err := s.getArtistDirectory(req.Context(), artistID)
	if err != nil {
		resp := responseError(0, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	artEtr := artistEntry{
		ID:         entry.ID,
		ParentID:   entry.ParentID,
		Name:       entry.Name,
		AlbumCount: entry.AlbumCount,
		SongCount:  entry.SongCount,
		CoverArtID: entry.CoverArtID,
		Children:   entry.Children,
	}

	resp := artistResponse{
		baseResponse: responseOk(),
		Artist:       artEtr,
	}

	encodeResponse(w, req, resp)
}

type artistResponse struct {
	baseResponse

	Artist artistEntry `xml:"artist" json:"artist"`
}

type artistEntry struct {
	ID         int64  `xml:"id,attr" json:"id,string"`
	ParentID   int64  `xml:"parent,attr,omitempty" json:"parent,string,omitempty"`
	Name       string `xml:"name,attr" json:"name"`
	AlbumCount int64  `xml:"albumCount,attr,omitempty" json:"albumCount,omitempty"`
	SongCount  int64  `xml:"songCount,attr,omitempty" json:"songCount,omitempty"`
	CoverArtID string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`

	Children []directoryChildEntry `xml:"album" json:"album"`
}
