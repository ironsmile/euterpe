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

	albumCount := parseIntOrDefault(reqQuery.Get("albumCount"), 20)
	albumOffset := parseIntOrDefault(reqQuery.Get("albumOffset"), 0)

	albums := s.lib.SearchAlbums(library.SearchArgs{
		Query:  searchQuery,
		Offset: albumOffset,
		Count:  albumCount,
	})
	for _, album := range albums {
		resp.Result.Albums = append(resp.Result.Albums, dbAlbumToAlbumID3Entry(album))
	}

	artistCount := parseIntOrDefault(reqQuery.Get("artistCount"), 20)
	artistOffset := parseIntOrDefault(reqQuery.Get("artistOffset"), 0)

	artists := s.lib.SearchArtists(library.SearchArgs{
		Query:  searchQuery,
		Offset: artistOffset,
		Count:  artistCount,
	})
	for _, artist := range artists {
		resp.Result.Artists = append(
			resp.Result.Artists,
			dbArtistToArtistID3Entry(artist),
		)
	}

	encodeResponse(w, req, resp)
}

type search3Response struct {
	baseResponse

	Result search3Result `xml:"searchResult3" json:"searchResult3"`
}

type search3Result struct {
	Artists []artistID3Entry      `xml:"artist" json:"artist"`
	Albums  []albumID3Entry       `xml:"album" json:"album"`
	Songs   []directoryChildEntry `xml:"song" json:"song"`
}