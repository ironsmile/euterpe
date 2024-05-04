package subsonic

import (
	"net/http"
	"strconv"
	"time"
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

	entry, err := s.getAlbumDirectory(req.Context(), albumID)
	if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	artEtr := albumEntry{
		ID:         entry.ID,
		Artist:     entry.Artist,
		ArtistID:   entry.ParentID,
		Name:       entry.Name,
		SongCount:  entry.SongCount,
		CoverArtID: entry.CoverArtID,
		Children:   entry.Children,
		Created:    s.lastModified,
		Duration:   entry.Duration,
		PlayCount:  entry.PlayCount,
	}

	resp := albumResponse{
		baseResponse: responseOk(),
		Album:        artEtr,
	}

	encodeResponse(w, req, resp)
}

type albumResponse struct {
	baseResponse

	Album albumEntry `xml:"album" json:"album"`
}

type albumEntry struct {
	ID            int64     `xml:"id,attr" json:"id,string"`
	ParentID      int64     `xml:"parent,attr,omitempty" json:"parent,string,omitempty"`
	Name          string    `xml:"name,attr" json:"name"`
	Artist        string    `xml:"artist,attr" json:"artist"`
	ArtistID      int64     `xml:"artistId,attr" json:"artistId,string"`
	SongCount     int64     `xml:"songCount,attr" json:"songCount"`
	CoverArtID    string    `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	IsCompilation bool      `xml:"-" json:"isCompilation"`
	Created       time.Time `xml:"created,attr" json:"created,omitempty"`
	Duration      int64     `xml:"duration,attr" json:"duration,omitempty"`
	PlayCount     int64     `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`

	Children []directoryChildEntry `xml:"song" json:"song"`
}
