package subsonic

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) getRandomSongs(w http.ResponseWriter, req *http.Request) {
	size := parseIntOrDefault(req.URL.Query().Get("size"), 10)
	genre := req.URL.Query().Get("genre")
	fromYear := req.URL.Query().Get("fromYear")
	toYear := req.URL.Query().Get("toYear")
	musicFolderID := req.URL.Query().Get("musicFolderID")

	// Ignored search filters:
	_ = musicFolderID
	_ = genre

	if size > 500 {
		size = 500
	}

	browseArgs := library.BrowseArgs{
		OrderBy: library.OrderByRandom,
		PerPage: uint(size),
	}

	if fromYear != "" {
		fromYearInt, err := strconv.ParseInt(fromYear, 10, 64)
		if err != nil {
			resp := responseError(errCodeGeneric, fmt.Sprintf("wrong fromYear: %s", err))
			encodeResponse(w, req, resp)
			return
		}
		browseArgs.FromYear = &fromYearInt
	}

	if toYear != "" {
		toYearInt, err := strconv.ParseInt(toYear, 10, 64)
		if err != nil {
			resp := responseError(errCodeGeneric, fmt.Sprintf("wrong toYear: %s", err))
			encodeResponse(w, req, resp)
			return
		}
		browseArgs.ToYear = &toYearInt
	}

	songs, _ := s.libBrowser.BrowseTracks(browseArgs)

	resp := getRandomSongsResponse{
		baseResponse: responseOk(),
	}

	for _, song := range songs {
		resp.RandomSongs.Songs = append(
			resp.RandomSongs.Songs,
			trackToChild(song, s.getLastModified()),
		)
	}

	encodeResponse(w, req, resp)
}

type getRandomSongsResponse struct {
	baseResponse

	RandomSongs xsdSongs `xml:"randomSongs" json:"randomSongs"`
}
