package subsonic

import (
    "net/http"
    "net/url"
    "strconv"
)

func (s *subsonic) getArtistInfo2(w http.ResponseWriter, req *http.Request) {
    idString := req.Form.Get("id")
    subsonicID, err := strconv.ParseInt(idString, 10, 64)
    if idString == "" || err != nil || !isArtistID(subsonicID) {
        resp := responseError(errCodeNotFound, "artist not found")
        encodeResponse(w, req, resp)
        return
    }

    artistID := toArtistDBID(subsonicID)

    albums := s.lib.GetArtistAlbums(artistID)
    if len(albums) == 0 {
        resp := responseError(errCodeNotFound, "artist not found")
        encodeResponse(w, req, resp)
        return
    }

    query := make(url.Values)
    query.Set("id", artistCoverArtID(artistID))
    setQueryFromReq(query, req)
    artURL := url.URL{
        Scheme:   getProtoFromRequest(req),
        Host:     getHostFromRequest(req),
        Path:     s.prefix + "/getCoverArt",
        RawQuery: query.Encode(),
    }

    resp := artistInfo2Response{
        baseResponse: responseOk(),
        ArtistInfo2:  aristInfo2Element{},
    }

    query.Set("size", "150")
    artURL.RawQuery = query.Encode()
    resp.ArtistInfo2.SmallImageURL = artURL.String()

    query.Set("size", "300")
    artURL.RawQuery = query.Encode()
    resp.ArtistInfo2.MediumImageURL = artURL.String()

    query.Set("size", "600")
    artURL.RawQuery = query.Encode()
    resp.ArtistInfo2.LargeImageURL = artURL.String()

    encodeResponse(w, req, resp)
}

func setQueryFromReq(query url.Values, req *http.Request) {
    reqQuery := req.Form

    if v := reqQuery.Get("c"); v != "" {
        query.Set("c", v)
    }
    if v := reqQuery.Get("s"); v != "" {
        query.Set("s", v)
    }
    if v := reqQuery.Get("t"); v != "" {
        query.Set("t", v)
    }
    if v := reqQuery.Get("p"); v != "" {
        query.Set("p", v)
    }
    if v := reqQuery.Get("v"); v != "" {
        query.Set("v", v)
    }
    if v := reqQuery.Get("u"); v != "" {
        query.Set("u", v)
    }
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
