package subsonic

import (
	"net/http"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) search(w http.ResponseWriter, req *http.Request) {
	reqQuery := req.URL.Query()

	count := parseIntOrDefault(reqQuery.Get("count"), 20)
	offset := parseIntOrDefault(reqQuery.Get("offset"), 0)

	trackQuery := reqQuery.Get("title")
	albumQuery := reqQuery.Get("album")
	artistQuery := reqQuery.Get("artist")

	if anyQuery := reqQuery.Get("any"); anyQuery != "" {
		trackQuery = anyQuery
		albumQuery = anyQuery
		artistQuery = anyQuery
	}

	resp := searchResponse{
		baseResponse: responseOk(),
		Result: searchResult{
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
				trackToDirChild(track, s.lastModified),
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
				albumToDirChild(
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
			resp.Result.Matches = append(
				resp.Result.Matches,
				artistToDirChild(artist, s.lastModified),
			)
		}
	}

	resp.Result.TotalHits = int64(len(resp.Result.Matches))

	encodeResponse(w, req, resp)
}

type searchResponse struct {
	baseResponse

	Result searchResult `xml:"searchResult" json:"searchResult"`
}

type searchResult struct {
	Offset    int64                 `xml:"offset,attr" json:"offset"`
	TotalHits int64                 `xml:"totalHits,attr" json:"totalHits"`
	Matches   []directoryChildEntry `xml:"match" json:"match"`
}
