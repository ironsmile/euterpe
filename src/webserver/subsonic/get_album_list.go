package subsonic

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) getAlbumList(w http.ResponseWriter, req *http.Request) {
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
		fromYear, fromErr := strconv.ParseInt(req.Form.Get("fromYear"), 10, 64)
		toYear, toErr := strconv.ParseInt(req.Form.Get("toYear"), 10, 64)
		if fromErr != nil || toErr != nil {
			resp := responseError(
				errCodeMissingParameter,
				"valid `fromYear` and `toYear` are required when type=byYear",
			)
			encodeResponse(w, req, resp)
			return
		}

		browseArgs.Order = library.OrderAsc
		browseArgs.FromYear = &fromYear
		browseArgs.ToYear = &toYear

		if fromYear > toYear {
			browseArgs.Order = library.OrderDesc
			browseArgs.FromYear = &toYear
			browseArgs.ToYear = &fromYear
		}
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
		browseArgs.Offset = offset
	}

	albums, _ := s.libBrowser.BrowseAlbums(browseArgs)

	var albumList []xsdChild
	for _, album := range albums {
		albumList = append(
			albumList,
			albumToChild(album, 0, s.getLastModified()), //!TODO: add artistID
		)
	}

	resp := albumListResponse{
		baseResponse: responseOk(),
		AlbumList: xsdAlbumList{
			Children: albumList,
		},
	}

	encodeResponse(w, req, resp)
}

type albumListResponse struct {
	baseResponse

	AlbumList xsdAlbumList `xml:"albumList" json:"albumList"`
}
