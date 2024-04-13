package subsonic

import (
    "net/http"
    "strconv"
)

func (s *subsonic) getAlbum(w http.ResponseWriter, req *http.Request) {
    idString := req.URL.Query().Get("id")
    subsonicID, err := strconv.ParseInt(idString, 10, 64)
    if idString == "" || err != nil || !isAlbumID(subsonicID) {
        resp := responseError(70, "album not found")
        encodeResponse(w, req, resp)
        return
    }

    albumID := toAlbumDBID(subsonicID)

    entry, err := s.getAlbumDirectory(req.Context(), albumID)
    if err != nil {
        resp := responseError(0, err.Error())
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
    ID            int64  `xml:"id,attr" json:"id,string"`
    ParentID      int64  `xml:"parent,attr,omitempty" json:"parent,string,omitempty"`
    Name          string `xml:"name,attr" json:"name"`
    Artist        string `xml:"artist,attr" json:"artist"`
    ArtistID      int64  `xml:"artistId,attr" json:"artistId,string"`
    SongCount     int64  `xml:"songCount,attr,omitempty" json:"songCount,omitempty"`
    CoverArtID    string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
    IsCompilation bool   `xml:"isCompilation,attr" json:"isCompilation"`

    Children []directoryChildEntry `xml:"song" json:"song"`
}
