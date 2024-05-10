package subsonic

import (
	"net/http"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) getStarred(w http.ResponseWriter, req *http.Request) {
	resp := starredResponse{
		baseResponse: responseOk(),
	}

	browseArgs := library.BrowseArgs{
		PerPage: 500,
		Order:   library.OrderDesc,
		OrderBy: library.OrderByFavourites,
	}

	artURL, query := s.getAristImageURL(req, 0)
	for {
		artists, _ := s.libBrowser.BrowseArtists(browseArgs)
		if len(artists) == 0 {
			break
		}

		var lastStarred bool
		for _, artist := range artists {
			if artist.Favourite == 0 {
				lastStarred = true
				break
			}

			query.Set("id", artistCoverArtID(artist.ID))
			artURL.RawQuery = query.Encode()

			resp.Starred.Artists = append(resp.Starred.Artists, toXSDArtist(
				artist,
				artURL,
			))
		}

		if lastStarred {
			break
		}

		browseArgs.Offset = uint64(len(resp.Starred.Artists))
	}

	browseArgs.Offset = 0
	for {
		albums, _ := s.libBrowser.BrowseAlbums(browseArgs)
		if len(albums) == 0 {
			break
		}

		var lastStarred bool
		for _, album := range albums {
			if album.Favourite == 0 {
				lastStarred = true
				break
			}

			resp.Starred.Albums = append(
				resp.Starred.Albums,
				albumToChild(album, 0, s.getLastModified()),
			)
		}

		if lastStarred {
			break
		}

		browseArgs.Offset = uint64(len(resp.Starred.Albums))
	}

	//!TODO: add songs

	encodeResponse(w, req, resp)
}

type starredResponse struct {
	baseResponse

	Starred xsdStarred `xml:"starred" json:"starred"`
}
