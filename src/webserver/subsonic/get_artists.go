package subsonic

import (
	"net/http"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) getArtists(w http.ResponseWriter, req *http.Request) {
	indexes := xsdArtistsID3{}

	artURL, artQuery := s.getAristImageURL(req, 0)
	var (
		page         uint = 0
		seenArtists  int
		currentIndex xsdIndexID3
	)
	for {
		artists, totalCount := s.libBrowser.BrowseArtists(library.BrowseArgs{
			Page:    page,
			PerPage: 500,
			Order:   library.OrderAsc,
			OrderBy: library.OrderByName,
		})

		if len(artists) == 0 {
			break
		}

		for _, artist := range artists {
			if artist.Name == "" {
				continue
			}
			forIndex := string([]rune(artist.Name)[0:1])

			if currentIndex.Name == "" {
				currentIndex.Name = forIndex
			}

			if forIndex != currentIndex.Name {
				indexes.Children = append(indexes.Children, currentIndex)
				currentIndex = xsdIndexID3{
					Name: forIndex,
				}
			}

			artQuery.Set("id", artistCoverArtID(artist.ID))
			artURL.RawQuery = artQuery.Encode()
			currentIndex.Children = append(
				currentIndex.Children,
				dbArtistToArtistID3(artist, artURL),
			)
		}

		page++
		seenArtists += len(artists)
		if seenArtists >= totalCount {
			break
		}
	}

	if len(currentIndex.Children) > 0 {
		indexes.Children = append(indexes.Children, currentIndex)
	}

	resp := artistsResponse{
		baseResponse: responseOk(),
		AristsList:   indexes,
	}

	encodeResponse(w, req, resp)
}

type artistsResponse struct {
	baseResponse

	AristsList xsdArtistsID3 `xml:"artists" json:"artists"`
}
