package subsonic

import (
	"net/http"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) search3(w http.ResponseWriter, req *http.Request) {
	reqQuery := req.URL.Query()
	searchQuery := reqQuery.Get("query")
	musicFolderID := reqQuery.Get("musicFolderId")
	if musicFolderID != "" && musicFolderExists(musicFolderID) {
		resp := responseError(errCodeNotFound, "music folder not found")
		encodeResponse(w, req, resp)
		return
	}
	songCount := parseIntOrDefault(reqQuery.Get("songCount"), 20)
	songOffset := parseIntOrDefault(reqQuery.Get("songOffset"), 0)

	resp := search3Response{
		baseResponse: responseOk(),
	}

	results := s.lib.Search(library.SearchArgs{
		Query:  searchQuery,
		Offset: songOffset,
		Count:  songCount,
	})
	for _, track := range results {
		resp.Result.Songs = append(
			resp.Result.Songs,
			trackToDirChild(track, s.lastModified),
		)
	}

	//!TODO: explictly search for albums and artists.

	encodeResponse(w, req, resp)
}

type search3Response struct {
	baseResponse

	Result search3Result `xml:"searchResult3" json:"searchResult3"`
}

type search3Result struct {
	Songs  []directoryChildEntry `xml:"song" json:"song"`
	Albums []albumID3Entry       `xml:"album" json:"album"`
}
