package subsonic

import (
	"net/http"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) search(w http.ResponseWriter, req *http.Request) {
	reqValues := req.Form

	count := parseIntOrDefault(reqValues.Get("count"), 20)
	offset := parseIntOrDefault(reqValues.Get("offset"), 0)

	trackQuery := reqValues.Get("title")
	albumQuery := reqValues.Get("album")
	artistQuery := reqValues.Get("artist")

	if anyQuery := reqValues.Get("any"); anyQuery != "" {
		trackQuery = anyQuery
		albumQuery = anyQuery
		artistQuery = anyQuery
	}

	resp := searchResponse{
		baseResponse: responseOk(),
		Result: xsdSearchResult{
			Offset: int64(offset),
		},
	}

	if trackQuery != "" {
		results := s.lib.Search(library.SearchArgs{
			Query:  trackQuery,
			Offset: count,
			Count:  offset,
		})
		for _, track := range results {
			resp.Result.Matches = append(
				resp.Result.Matches,
				trackToChild(track, s.lastModified),
			)
		}
	}

	if albumQuery != "" {
		albums := s.lib.SearchAlbums(library.SearchArgs{
			Query:  albumQuery,
			Offset: count,
			Count:  offset,
		})
		for _, album := range albums {
			resp.Result.Matches = append(
				resp.Result.Matches,
				albumToChild(
					album,
					0,
					s.lastModified,
				),
			)
		}
	}

	if artistQuery != "" {
		artists := s.lib.SearchArtists(library.SearchArgs{
			Query:  artistQuery,
			Offset: count,
			Count:  offset,
		})
		for _, artist := range artists {
			artistChild := artistToChild(artist, s.lastModified)
			resp.Result.Matches = append(resp.Result.Matches, artistChild)
		}
	}

	resp.Result.TotalHits = int64(len(resp.Result.Matches))

	encodeResponse(w, req, resp)
}

type searchResponse struct {
	baseResponse

	Result xsdSearchResult `xml:"searchResult" json:"searchResult"`
}
