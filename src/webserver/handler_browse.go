package webserver

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/ironsmile/euterpe/src/library"
	"github.com/ironsmile/euterpe/src/webserver/webutils"
)

// BrowseHandler is a http.Handler which will allow you to browse through artists or
// albums with the help of pagination.
type BrowseHandler struct {
	browser library.Browser
}

// ServeHTTP is required by the http.Handler's interface
func (bh BrowseHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	InternalErrorOnErrorHandler(writer, req, bh.browse)
}

// Actually generates a browse results using the library
func (bh BrowseHandler) browse(writer http.ResponseWriter, req *http.Request) error {
	writer.Header().Add("Content-Type", "application/json; charset=utf-8")

	if err := req.ParseForm(); err != nil {
		bh.badRequest(writer, err.Error())
		return nil
	}

	var page, perPage int = 1, 10
	pageStr := req.Form.Get("page")
	perPageStr := req.Form.Get("per-page")
	browseBy := req.Form.Get("by")
	orderBy := strings.TrimSpace(strings.ToLower(req.Form.Get("order-by")))
	order := strings.TrimSpace(strings.ToLower(req.Form.Get("order")))

	possibleTypes := []string{"artist", "album", "song"}
	if browseBy != "" && !slices.Contains(possibleTypes, browseBy) {
		bh.badRequest(writer,
			fmt.Sprintf(
				"Wrong 'by' parameter. Must be one of %s",
				strings.Join(possibleTypes, ", "),
			),
		)
		return nil
	}

	possibleOrders := []string{"id", "name", "random", "frequency", "recency", "year"}
	if orderBy != "" && !slices.Contains(possibleOrders, orderBy) {
		bh.badRequest(writer,
			fmt.Sprintf("Wrong 'order-by' parameter - '%s'. ", orderBy)+
				fmt.Sprintf("Must be one of %s", strings.Join(possibleOrders, ", ")))
		return nil
	}

	if order != "" && order != "asc" && order != "desc" {
		bh.badRequest(writer, "Wrong 'order-type' parameter. Must be 'asc' or 'desc'")
		return nil
	}

	if pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)

		if err != nil {
			bh.badRequest(writer, fmt.Sprintf(`Wrong "page" parameter: %s`, err))
			return nil
		}
	}

	if perPageStr != "" {
		var err error
		perPage, err = strconv.Atoi(perPageStr)

		if err != nil {
			bh.badRequest(writer, fmt.Sprintf(`Wrong "per-page" parameter: %s`, err))
			return nil
		}
	}

	if page < 1 || perPage < 1 {
		bh.badRequest(writer, `"page" and "per-page" must be integers greater than one`)
		return nil
	}

	if browseBy == "artist" {
		return bh.browseArtists(writer, page, perPage, orderBy, order)
	} else if browseBy == "song" {
		return bh.browseSongs(writer, page, perPage, orderBy, order)
	}

	return bh.browseAlbums(writer, page, perPage, orderBy, order)
}

func (bh BrowseHandler) browseAlbums(
	writer http.ResponseWriter,
	page, perPage int,
	orderBy, order string,
) error {
	browseArgs := getBrowseArgs(page, perPage, orderBy, order)
	albums, count := bh.browser.BrowseAlbums(browseArgs)
	prevPage, nextPage := getBrowsePrevNextPageURI(
		"album",
		page,
		perPage,
		count,
		orderBy,
		order,
	)

	retData := struct {
		Data       []library.Album `json:"data"`
		Next       string          `json:"next"`
		Previous   string          `json:"previous"`
		PagesCount int             `json:"pages_count"`
	}{
		Data:       albums,
		PagesCount: int(math.Ceil(float64(count) / float64(perPage))),
		Next:       nextPage,
		Previous:   prevPage,
	}

	enc := json.NewEncoder(writer)
	return enc.Encode(retData)
}

func (bh BrowseHandler) browseArtists(
	writer http.ResponseWriter,
	page, perPage int,
	orderBy, order string,
) error {
	browseArgs := getBrowseArgs(page, perPage, orderBy, order)
	unsupportedBrowseBy := []library.BrowseOrderBy{
		library.OrderByRecentlyPlayed,
		library.OrderByFrequentlyPlayed,
		library.OrderByYear,
	}
	if slices.Contains(unsupportedBrowseBy, browseArgs.OrderBy) {
		return fmt.Errorf(
			"this type of order (%d) is not supported for artists",
			browseArgs.OrderBy,
		)
	}

	artists, count := bh.browser.BrowseArtists(browseArgs)
	prevPage, nextPage := getBrowsePrevNextPageURI(
		"artist",
		page,
		perPage,
		count,
		orderBy,
		order,
	)

	retData := struct {
		Data       []library.Artist `json:"data"`
		Next       string           `json:"next"`
		Previous   string           `json:"previous"`
		PagesCount int              `json:"pages_count"`
	}{
		Data:       artists,
		PagesCount: int(math.Ceil(float64(count) / float64(perPage))),
		Next:       nextPage,
		Previous:   prevPage,
	}

	enc := json.NewEncoder(writer)
	return enc.Encode(retData)
}

func (bh BrowseHandler) browseSongs(
	writer http.ResponseWriter,
	page, perPage int,
	orderBy, order string,
) error {
	if orderBy == "" {
		orderBy = "id"
	}

	browseArgs := getBrowseArgs(page, perPage, orderBy, order)
	tracks, count := bh.browser.BrowseTracks(browseArgs)
	prevPage, nextPage := getBrowsePrevNextPageURI(
		"track",
		page,
		perPage,
		count,
		orderBy,
		order,
	)

	retData := struct {
		Data       []library.TrackInfo `json:"data"`
		Next       string              `json:"next"`
		Previous   string              `json:"previous"`
		PagesCount int                 `json:"pages_count"`
	}{
		Data:       tracks,
		PagesCount: int(math.Ceil(float64(count) / float64(perPage))),
		Next:       nextPage,
		Previous:   prevPage,
	}

	enc := json.NewEncoder(writer)
	return enc.Encode(retData)
}

func (bh BrowseHandler) badRequest(writer http.ResponseWriter, message string) {
	webutils.JSONError(writer, message, http.StatusBadRequest)
}

func getBrowseArgs(page, perPage int, orderBy, order string) library.BrowseArgs {
	// In the API we count starting from 1. But actually for the library function
	// pages are counted from 0 which is much easier for implementing.
	browsePage := uint64(page - 1)

	browseArgs := library.BrowseArgs{
		Offset:  browsePage * uint64(perPage),
		PerPage: uint(perPage),
	}

	switch orderBy {
	case "id":
		browseArgs.OrderBy = library.OrderByID
	case "random":
		browseArgs.OrderBy = library.OrderByRandom
	case "frequency":
		browseArgs.OrderBy = library.OrderByFrequentlyPlayed
	case "recency":
		browseArgs.OrderBy = library.OrderByRecentlyPlayed
	case "year":
		browseArgs.OrderBy = library.OrderByYear
	default:
		browseArgs.OrderBy = library.OrderByName
	}

	switch order {
	case "desc":
		browseArgs.Order = library.OrderDesc
	default:
		browseArgs.Order = library.OrderAsc
	}

	return browseArgs
}

func getBrowsePrevNextPageURI(
	by string,
	page, perPage, count int,
	orderBy,
	order string,
) (string, string) {
	orderArg := ""
	orderByArg := ""

	if order != "" {
		orderArg = fmt.Sprintf("&order=%s", order)
	}

	if orderBy != "" {
		orderByArg = fmt.Sprintf("&order-by=%s", orderBy)
	}

	prevPage := ""

	if page-1 > 0 {
		prevPage = fmt.Sprintf(
			"/v1/browse?by=%s&page=%d&per-page=%d%s%s",
			by,
			page-1,
			perPage,
			orderArg,
			orderByArg,
		)
	}

	nextPage := ""

	if page*perPage < count {
		nextPage = fmt.Sprintf(
			"/v1/browse?by=%s&page=%d&per-page=%d%s%s",
			by,
			page+1,
			perPage,
			orderArg,
			orderByArg,
		)
	}

	return prevPage, nextPage
}

// NewBrowseHandler returns a new Browse handler. It needs a library.Browser to browse
// through.
func NewBrowseHandler(browser library.Browser) *BrowseHandler {
	return &BrowseHandler{
		browser: browser,
	}
}
