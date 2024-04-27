package subsonic

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) getAlbumList2(w http.ResponseWriter, req *http.Request) {
	browseType := req.URL.Query().Get("type")
	sizeString := req.URL.Query().Get("size")
	offsetString := req.URL.Query().Get("offset")

	browseArgs := library.BrowseArgs{}
	switch browseType {
	case "":
		resp := responseError(errCodeMissingParameter, "`type` parameter is missing")
		encodeResponse(w, req, resp)
		return
	case "random":
		browseArgs.OrderBy = library.OrderByRandom
	case "newest":
		browseArgs.OrderBy = library.OrderByID
		browseArgs.Order = library.OrderDesc
	case "alphabeticalByName":
		browseArgs.OrderBy = library.OrderByName
		browseArgs.Order = library.OrderAsc
	case "starred":
		fallthrough
	case "frequent":
		fallthrough
	case "recent":
		fallthrough
	case "alphabeticalByArtist":
		fallthrough
	case "byYear":
		fallthrough
	case "byGenre":
		resp := responseError(
			errCodeGeneric,
			fmt.Sprintf("ordering by `%s` is not yet supported", browseType),
		)
		encodeResponse(w, req, resp)
		return
	default:
		resp := responseError(errCodeMissingParameter, "unknown `type` parameter used")
		encodeResponse(w, req, resp)
		return
	}

	perPage, err := strconv.ParseUint(sizeString, 10, 32)
	if err == nil && perPage > 0 && perPage <= 500 {
		browseArgs.PerPage = uint(perPage)
	} else {
		browseArgs.PerPage = 10
	}

	offset, err := strconv.ParseUint(offsetString, 10, 32)
	if err == nil && offset > 0 {
		browseArgs.Page = uint(offset) / browseArgs.PerPage
	}

	albums, _ := s.libBrowser.BrowseAlbums(browseArgs)

	var albumList []albumID3Entry
	for _, album := range albums {
		entry := albumID3Entry{
			ID:         albumFSID(album.ID),
			Name:       album.Name,
			Artist:     album.Artist,
			SongCount:  album.SongCount,
			CoverArtID: albumConverArtID(album.ID),

			//!TODO: add ParentID and AlbumID
		}

		albumList = append(albumList, entry)
	}

	resp := albumList2Response{
		baseResponse: responseOk(),
		AlbumList2: albumList2Element{
			Children: albumList,
		},
	}

	encodeResponse(w, req, resp)
}

type albumList2Response struct {
	baseResponse

	AlbumList2 albumList2Element `xml:"albumList2" json:"albumList2"`
}

type albumList2Element struct {
	Children []albumID3Entry `xml:"album" json:"album"`
}
