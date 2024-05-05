package subsonic

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) getAlbumList2(w http.ResponseWriter, req *http.Request) {
	browseType := req.Form.Get("type")
	sizeString := req.Form.Get("size")
	offsetString := req.Form.Get("offset")

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
	case "frequent":
		browseArgs.OrderBy = library.OrderByFrequentlyPlayed
		browseArgs.Order = library.OrderDesc
	case "recent":
		browseArgs.OrderBy = library.OrderByRecentlyPlayed
		browseArgs.Order = library.OrderDesc
	case "starred":
		browseArgs.OrderBy = library.OrderByFavourites
		browseArgs.Order = library.OrderDesc
	case "alphabeticalByArtist":
		browseArgs.OrderBy = library.OrderByArtistName
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

	var albumList []xsdAlbumID3
	for _, album := range albums {
		albumList = append(albumList, dbAlbumToAlbumID3Entry(album))
	}

	resp := albumList2Response{
		baseResponse: responseOk(),
		AlbumList2: xsdAlbumList2{
			Children: albumList,
		},
	}

	encodeResponse(w, req, resp)
}

type albumList2Response struct {
	baseResponse

	AlbumList2 xsdAlbumList2 `xml:"albumList2" json:"albumList2"`
}
