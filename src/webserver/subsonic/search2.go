package subsonic

import (
	"net/http"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) search2(w http.ResponseWriter, req *http.Request) {
	reqValues := req.Form
	searchQuery := reqValues.Get("query")
	musicFolderID := reqValues.Get("musicFolderId")
	if musicFolderID != "" && musicFolderExists(musicFolderID) {
		resp := responseError(errCodeNotFound, "music folder not found")
		encodeResponse(w, req, resp)
		return
	}
	songCount := parseIntOrDefault(reqValues.Get("songCount"), 20)
	songOffset := parseIntOrDefault(reqValues.Get("songOffset"), 0)

	resp := search2Response{
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

	albumCount := parseIntOrDefault(reqValues.Get("albumCount"), 20)
	albumOffset := parseIntOrDefault(reqValues.Get("albumOffset"), 0)

	albums := s.lib.SearchAlbums(library.SearchArgs{
		Query:  searchQuery,
		Offset: albumOffset,
		Count:  albumCount,
	})
	for _, album := range albums {
		resp.Result.Albums = append(
			resp.Result.Albums,
			albumToDirChild(
				album,
				0,
				s.lastModified,
			),
		)
	}

	artistCount := parseIntOrDefault(reqValues.Get("artistCount"), 20)
	artistOffset := parseIntOrDefault(reqValues.Get("artistOffset"), 0)

	artURL, query := s.getAristImageURL(req, 0)
	artists := s.lib.SearchArtists(library.SearchArgs{
		Query:  searchQuery,
		Offset: artistOffset,
		Count:  artistCount,
	})
	for _, artist := range artists {
		query.Set("id", artistCoverArtID(artist.ID))
		artURL.RawQuery = query.Encode()

		resp.Result.Artists = append(
			resp.Result.Artists,
			directoryEntry{
				ID:             artistFSID(artist.ID),
				Artist:         artist.Name,
				Name:           artist.Name,
				AlbumCount:     artist.AlbumCount,
				CoverArtID:     artistCoverArtID(artist.ID),
				ArtistImageURL: artURL.String(),
			},
		)
	}

	encodeResponse(w, req, resp)
}

type search2Response struct {
	baseResponse

	Result search2Result `xml:"searchResult2" json:"searchResult2"`
}

type search2Result struct {
	Artists []directoryEntry      `xml:"artist" json:"artist"`
	Albums  []directoryChildEntry `xml:"album" json:"album"`
	Songs   []directoryChildEntry `xml:"song" json:"song"`
}
