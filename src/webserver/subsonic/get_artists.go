package subsonic

import (
    "net/http"

    "github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) getArtists(w http.ResponseWriter, req *http.Request) {
    indexes := artistsList{}

    var (
        page         uint = 0
        seenArtists  int
        currentIndex artistElement
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
                currentIndex = artistElement{
                    Name: forIndex,
                }
            }

            currentIndex.Children = append(
                currentIndex.Children,
                artistIndexElement{
                    ID:         artistFSID(artist.ID),
                    Name:       artist.Name,
                    CoverArt:   artistCoverArtID(artist.ID),
                    AlbumCount: artist.AlbumCount,
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

    resp := artistsResponse{
        baseResponse: responseOk(),
        AristsList:   indexes,
    }

    encodeResponse(w, req, resp)
}

type artistsResponse struct {
    baseResponse

    AristsList artistsList `xml:"artists" json:"artists"`
}

type artistsList struct {
    IgnoredArticles string          `xml:"ignoredArticles,attr,omitempty" json:"ignoredArticles"`
    Children        []artistElement `xml:"index" json:"index"`
}

type artistElement struct {
    Name     string               `xml:"name,attr" json:"name"`
    Children []artistIndexElement `xml:"artist" json:"artist"`
}

type artistIndexElement struct {
    ID         int64  `xml:"id,attr" json:"id,string"`
    Name       string `xml:"name,attr" json:"name"`
    CoverArt   string `xml:"coverArt,attr" json:"coverArt"`
    AlbumCount int64  `xml:"albumCount,attr,omitempty" json:"albumCount,omitempty"`
}
