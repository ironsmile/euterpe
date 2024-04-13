package subsonic

import (
    "net/http"
    "net/url"
    "strconv"
)

func (s *subsonic) getArtistInfo2(w http.ResponseWriter, req *http.Request) {
    idString := req.URL.Query().Get("id")
    subsonicID, err := strconv.ParseInt(idString, 10, 64)
    if idString == "" || err != nil || !isArtistID(subsonicID) {
        resp := responseError(70, "artist not found")
        encodeResponse(w, req, resp)
        return
    }

    artistID := toArtistDBID(subsonicID)

    albums := s.lib.GetArtistAlbums(artistID)
    if len(albums) == 0 {
        resp := responseError(70, "artist not found")
        encodeResponse(w, req, resp)
        return
    }

    query := make(url.Values)
    query.Set("id", artistCoverArtID(artistID))
    query.Set("size", "150")
    artURL := url.URL{
        Path:     "/getCoverArt",
        RawQuery: query.Encode(),
    }

    resp := artistInfo2Response{
        baseResponse: responseOk(),
        ArtistInfo2: aristInfo2Element{
            SmallImageURL:  artURL.String(),
            MediumImageURL: artURL.String(),
        },
    }

    query.Set("size", "600")
    artURL.RawQuery = query.Encode()

    resp.ArtistInfo2.LargeImageURL = artURL.String()

    encodeResponse(w, req, resp)
}

type artistInfo2Response struct {
    baseResponse

    ArtistInfo2 aristInfo2Element `xml:"artistInfo2"`
}

type aristInfo2Element struct {
    SmallImageURL  string `xml:"smallImageUrl" json:"smallImageUrl"`
    MediumImageURL string `xml:"mediumImageUrl" json:"mediumImageUrl"`
    LargeImageURL  string `xml:"largeImageUrl" json:"largeImageUrl"`
}
