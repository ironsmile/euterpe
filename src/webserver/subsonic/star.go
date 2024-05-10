package subsonic

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) star(w http.ResponseWriter, req *http.Request) {
	favs, ok := s.parseStarArguments(w, req)
	if !ok {
		return
	}

	if err := s.lib.RecordFavourite(req.Context(), favs); err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	encodeResponse(w, req, responseOk())
}

// parseStarArguments parses the HTTP request for the favourite arguments defined in the
// Subsonic API for "star" and "unstar" methods. If any errors are found it responds
// into `w` appropriately and the second return is "false". If everything went find
// the second return is "true".
func (s *subsonic) parseStarArguments(
	w http.ResponseWriter,
	req *http.Request,
) (library.Favourites, bool) {
	favs := library.Favourites{}

	respondUnknownID := func(idVal string) {
		resp := responseError(
			errCodeGeneric,
			fmt.Sprintf("unknown ID type for `%s`", idVal),
		)
		encodeResponse(w, req, resp)
	}

	responedWrongID := func(idVal string, err error) {
		resp := responseError(
			errCodeGeneric,
			fmt.Sprintf("cannot parse ID: `%s`: %s", idVal, err),
		)
		encodeResponse(w, req, resp)
	}

	for _, idString := range req.Form["id"] {
		id, err := strconv.ParseInt(idString, 10, 64)
		if err != nil {
			responedWrongID(idString, err)
			return favs, false
		}

		if isTrackID(id) {
			favs.TrackIDs = append(favs.TrackIDs, toTrackDBID(id))
		} else if isAlbumID(id) {
			favs.AlbumIDs = append(favs.AlbumIDs, toAlbumDBID(id))
		} else if isArtistID(id) {
			favs.ArtistIDs = append(favs.ArtistIDs, toArtistDBID(id))
		} else {
			respondUnknownID(idString)
			return favs, false
		}
	}

	for _, albumIDStr := range req.Form["albumId"] {
		albumID, err := strconv.ParseInt(albumIDStr, 10, 64)
		if err != nil {
			responedWrongID(albumIDStr, err)
			return favs, false
		}
		if !isAlbumID(albumID) {
			respondUnknownID(albumIDStr)
			return favs, false
		}

		favs.AlbumIDs = append(favs.AlbumIDs, toAlbumDBID(albumID))
	}

	for _, artistIDStr := range req.Form["artistId"] {
		artistID, err := strconv.ParseInt(artistIDStr, 10, 64)
		if err != nil {
			responedWrongID(artistIDStr, err)
			return favs, false
		}
		if !isArtistID(artistID) {
			respondUnknownID(artistIDStr)
			return favs, false
		}

		favs.ArtistIDs = append(favs.ArtistIDs, toArtistDBID(artistID))
	}

	if len(favs.TrackIDs) == 0 && len(favs.AlbumIDs) == 0 && len(favs.ArtistIDs) == 0 {
		resp := responseError(
			errCodeMissingParameter,
			"missing arguments: id, artistId or albumId",
		)
		encodeResponse(w, req, resp)
		return favs, false
	}

	return favs, true
}
