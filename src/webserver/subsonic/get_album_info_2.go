package subsonic

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) getAlbumInfo2(w http.ResponseWriter, req *http.Request) {
	idString := req.Form.Get("id")
	subsonicID, err := strconv.ParseInt(idString, 10, 64)
	if idString == "" || err != nil || !isAlbumID(subsonicID) {
		resp := responseError(errCodeNotFound, "album not found")
		encodeResponse(w, req, resp)
		return
	}

	albumID := toAlbumDBID(subsonicID)

	album, err := s.lib.GetAlbum(req.Context(), albumID)
	if err != nil && errors.Is(err, library.ErrAlbumNotFound) {
		resp := responseError(errCodeNotFound, "album not found")
		encodeResponse(w, req, resp)
		return
	} else if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	artURL, query := s.getAlbumImageURL(req, album.ID)

	resp := albumInfoResponse{
		baseResponse: responseOk(),
		AlbumInfo: xsdAlbumInfo{
			LastfmURL: "https://last.fm/music/" + url.PathEscape(album.Artist) + "/" +
				url.PathEscape(album.Name),
		},
	}

	query.Set("size", "150")
	artURL.RawQuery = query.Encode()
	resp.AlbumInfo.SmallImageURL = artURL.String()

	query.Set("size", "300")
	artURL.RawQuery = query.Encode()
	resp.AlbumInfo.MediumImageURL = artURL.String()

	query.Set("size", "600")
	artURL.RawQuery = query.Encode()
	resp.AlbumInfo.LargeImageURL = artURL.String()

	encodeResponse(w, req, resp)
}

// getAlbumImageURL returns a URL for album image with query parameters
// for access set from the request.
// albumID is an ID from the database.
func (s *subsonic) getAlbumImageURL(
	req *http.Request,
	albumID int64,
) (url.URL, url.Values) {
	query := make(url.Values)
	query.Set("id", albumConverArtID(albumID))
	setQueryFromReq(query, req)
	artURL := url.URL{
		Scheme:   getProtoFromRequest(req),
		Host:     getHostFromRequest(req),
		Path:     s.prefix + "/getCoverArt",
		RawQuery: query.Encode(),
	}

	return artURL, query
}

type albumInfoResponse struct {
	baseResponse

	AlbumInfo xsdAlbumInfo `xml:"albumInfo"`
}
