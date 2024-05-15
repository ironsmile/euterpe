package subsonic

import (
	"net/http"
	"strings"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) getTopSongs(w http.ResponseWriter, req *http.Request) {
	count := parseIntOrDefault(req.Form.Get("count"), 50)
	artistName := req.Form.Get("artist")
	if artistName == "" {
		resp := responseError(errCodeMissingParameter, "The 'artist' param is missing")
		encodeResponse(w, req, resp)
		return
	}
	if count > 500 {
		count = 500
	}

	artists := s.lib.SearchArtists(library.SearchArgs{
		Query: artistName,
		Count: 10,
	})

	var (
		artist library.Artist
		found  bool
	)
	for _, foundArtist := range artists {
		if strings.EqualFold(artistName, foundArtist.Name) {
			found = true
			artist = foundArtist
			break
		}
	}

	if !found {
		resp := responseError(errCodeNotFound, "No such artist was found")
		encodeResponse(w, req, resp)
		return
	}

	topSongs, _ := s.libBrowser.BrowseTracks(library.BrowseArgs{
		OrderBy:  library.OrderByFrequentlyPlayed,
		Order:    library.OrderDesc,
		PerPage:  uint(count),
		ArtistID: artist.ID,
	})

	resp := topSonxResponse{
		baseResponse: responseOk(),
	}

	for _, song := range topSongs {
		resp.TopSongs.Songs = append(resp.TopSongs.Songs, trackToChild(
			song,
			s.getLastModified(),
		))
	}

	encodeResponse(w, req, resp)
}

type topSonxResponse struct {
	baseResponse

	TopSongs xsdTopSongs `xml:"topSongs" json:"topSongs"`
}
