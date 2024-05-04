package subsonic

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ironsmile/euterpe/src/library"
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

	entry, err := s.getArtistDirectory(req.Context(), artistID)
	if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	artEtr := artistID3Entry{
		ID:         entry.ID,
		ParentID:   entry.ParentID,
		Name:       entry.Name,
		AlbumCount: entry.AlbumCount,
		SongCount:  entry.SongCount,
		CoverArtID: entry.CoverArtID,
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

	Artist artistID3Entry `xml:"artist" json:"artist"`
}

type artistID3Entry struct {
	ID             int64  `xml:"id,attr" json:"id,string"`
	ParentID       int64  `xml:"parent,attr,omitempty" json:"parent,string,omitempty"`
	Name           string `xml:"name,attr" json:"name"`
	AlbumCount     int64  `xml:"albumCount,attr,omitempty" json:"albumCount,omitempty"`
	SongCount      int64  `xml:"songCount,attr,omitempty" json:"songCount,omitempty"`
	CoverArtID     string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	ArtistImageURL string `xml:"artistImageUrl,attr,omitempty" json:"artistImageUrl,omitempty"`

	Children []albumID3Entry `xml:"album,omitempty" json:"album,omitempty"`
}

type albumID3Entry struct {
	ID         int64     `xml:"id,attr" json:"id,string"`
	Name       string    `xml:"name,attr" json:"name"`
	Artist     string    `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	ArtistID   int64     `xml:"artistId,attr,omitempty" json:"artistId,omitempty,string"`
	CoverArtID string    `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Duration   int64     `xml:"duration,attr" json:"duration"` // in seconds
	Year       int16     `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre      string    `xml:"genre,attr,omitempty" json:"gener,omitempty"`
	SongCount  int64     `xml:"songCount,attr" json:"songCount"`
	Created    time.Time `xml:"created,attr" json:"created"`
}

func toAlbumID3Entry(child directoryChildEntry) albumID3Entry {
	return albumID3Entry{
		ID:         child.ID,
		Name:       child.Name,
		Artist:     child.Artist,
		ArtistID:   child.ArtistID,
		CoverArtID: child.CoverArtID,
		Duration:   child.Duration,
		Year:       child.Year,
		Genre:      child.Genre,
		SongCount:  child.SongCount,
		Created:    child.Created,
	}
}

func dbArtistToArtistID3Entry(artist library.Artist) artistID3Entry {
	return artistID3Entry{
		ID:         artistFSID(artist.ID),
		Name:       artist.Name,
		AlbumCount: artist.AlbumCount,
		CoverArtID: artistCoverArtID(artist.ID),
	}
}
