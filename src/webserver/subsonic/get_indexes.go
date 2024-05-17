package subsonic

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) getIndexes(w http.ResponseWriter, req *http.Request) {
	musicFolderID := req.Form.Get("musicFolderId")
	ifModifiedSince := req.Form.Get("ifModifiedSince")

	if musicFolderID != "" && !musicFolderExists(musicFolderID) {
		resp := responseError(errCodeNotFound, "Unknown music folder ID")
		encodeResponse(w, req, resp)
		return
	}

	if ifModifiedSince != "" {
		t, err := strconv.ParseInt(ifModifiedSince, 10, 64)
		if err != nil {
			resp := responseError(
				0,
				fmt.Sprintf("ifModifiedSince must be an int: %s", err),
			)
			encodeResponse(w, req, resp)
			return
		}
		ifModfiedSinceTime := time.Unix(t/1000, (t%1000)*1e6)

		if ifModfiedSinceTime.After(s.getLastModified()) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	indexes := xsdIndexes{
		LastModified: s.getLastModified().UnixMilli(),
	}

	artURL, artURLQuery := s.getAristImageURL(req, 0)

	var (
		page         uint = 0
		seenArtists  int
		currentIndex xsdIndex
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
				currentIndex = xsdIndex{
					Name: forIndex,
				}
			}

			artURLQuery.Set("id", artistCoverArtID(artist.ID))
			artURL.RawQuery = artURLQuery.Encode()
			currentIndex.Children = append(
				currentIndex.Children,
				toXSDArtist(artist, artURL),
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

	resp := indexesResponse{
		baseResponse: responseOk(),
		IndexesList:  indexes,
	}

	encodeResponse(w, req, resp)
}

type indexesResponse struct {
	baseResponse

	IndexesList xsdIndexes `xml:"indexes" json:"indexes"`
}
