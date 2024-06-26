package subsonic

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ironsmile/euterpe/src/library"
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

	artist, err := s.lib.GetArtist(req.Context(), artistID)
	if err != nil && errors.Is(err, library.ErrArtistNotFound) {
		resp := responseError(errCodeNotFound, "artist not found")
		encodeResponse(w, req, resp)
		return
	} else if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	resp := s.getArtistInfoBase(req, artist)
	encodeResponse(w, req, resp)
}

func (s *subsonic) getArtistInfoBase(
	req *http.Request,
	artist library.Artist,
) artistInfo2Response {
	artURL, query := s.getAristImageURL(req, artist.ID)

	resp := artistInfo2Response{
		baseResponse: responseOk(),
		ArtistInfo2: xsdArtistInfoBase{
			LastfmURL: "https://last.fm/music/" + url.PathEscape(artist.Name),
		},
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

	return resp
}

// getAristImageURL returns a URL for artist image with query parameters
// for access set from the request.
// artistID is an ID from the database.
func (s *subsonic) getAristImageURL(
	req *http.Request,
	artistID int64,
) (url.URL, url.Values) {
	query := make(url.Values)
	query.Set("id", artistCoverArtID(artistID))
	setQueryFromReq(query, req)
	artURL := url.URL{
		Scheme:   getProtoFromRequest(req),
		Host:     getHostFromRequest(req),
		Path:     s.prefix + "/getCoverArt",
		RawQuery: query.Encode(),
	}

	return artURL, query
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

	ArtistInfo2 xsdArtistInfoBase `xml:"artistInfo2"`
}
