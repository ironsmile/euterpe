package subsonic

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) getIndexes(w http.ResponseWriter, req *http.Request) {
	musicFolderID := req.URL.Query().Get("musicFolderId")
	ifModifiedSince := req.URL.Query().Get("ifModifiedSince")
	combindIDstr := strconv.FormatInt(combinedMusicFolderID, 10)

	if musicFolderID != "" && musicFolderID != combindIDstr {
		resp := responseError(70, "Unknown music folder ID")
		encodeResponse(w, resp)
		return
	}

	if ifModifiedSince != "" {
		t, err := strconv.ParseInt(ifModifiedSince, 10, 64)
		if err != nil {
			resp := responseError(
				0,
				fmt.Sprintf("ifModifiedSince must be an int: %s", err),
			)
			encodeResponse(w, resp)
			return
		}
		ifModfiedSinceTime := time.Unix(t/1000, (t%1000)*1e6)

		if ifModfiedSinceTime.After(s.getLastModified()) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	indexes := indexesList{
		LastModified: s.getLastModified().UnixMilli(),
	}

	var (
		page         uint = 0
		seenArtists  int
		currentIndex indexElement
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
				currentIndex = indexElement{
					Name: forIndex,
				}
			}

			currentIndex.Children = append(
				currentIndex.Children,
				indexArtistElement{
					ID:   artistFSID(artist.ID),
					Name: artist.Name,
				},
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

	encodeResponse(w, resp)
}

type indexesResponse struct {
	baseResponse

	IndexesList indexesList
}

type indexesList struct {
	XMLName         xml.Name `xml:"indexes"`
	LastModified    int64    `xml:"lastModified,attr"`
	IgnoredArticles string   `xml:"ignoredArticles,attr"`
	Children        []indexElement
}

type indexElement struct {
	XMLName  xml.Name `xml:"index"`
	Name     string   `xml:"name,attr"`
	Children []indexArtistElement
}

type indexArtistElement struct {
	XMLName xml.Name `xml:"artst"`
	ID      int64    `xml:"id,attr"`
	Name    string   `xml:"name,attr"`
}
