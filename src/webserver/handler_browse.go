package webserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/ironsmile/httpms/src/library"
)

// BrowseHandler is a http.Handler which will allow you to browse through artists or albums
// with the help of pagination.
type BrowseHandler struct {
	library library.Library
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

	if browseBy != "" && browseBy != "artist" && browseBy != "album" {
		bh.badRequest(writer, "Wrong 'by' parameter. Must be 'album' or 'artist'")
		return nil
	}

	if pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)

		if err != nil {
			bh.badRequest(writer, fmt.Sprintf(`Wrong "page" parameter: %s`, err.Error()))
			return nil
		}
	}

	if perPageStr != "" {
		var err error
		perPage, err = strconv.Atoi(perPageStr)

		if err != nil {
			bh.badRequest(writer, fmt.Sprintf(`Wrong "perPage" parameter: %s`, err.Error()))
			return nil
		}
	}

	if page < 1 || perPage < 1 {
		bh.badRequest(writer, `"page" and "perPage" must be integers greater than one`)
		return nil
	}

	if browseBy == "artist" {
		return bh.browseArtists(writer, page, perPage)
	}

	return bh.browseAlbums(writer, page, perPage)
}

func (bh BrowseHandler) browseAlbums(writer http.ResponseWriter, page, perPage int) error {
	// In the API we count starting from 1. But actually for the library function pages
	// are counted from 0 which is much easier for implementing.
	albums, count := bh.library.BrowseAlbums(uint(page-1), uint(perPage))

	prevPage := ""

	if page-1 > 0 {
		prevPage = fmt.Sprintf("/browse/?by=album&page=%d&per-page=%d", page-1, perPage)
	}

	nextPage := ""

	if page*perPage < count {
		nextPage = fmt.Sprintf("/browse/?by=album&page=%d&per-page=%d", page+1, perPage)
	}

	retData := struct {
		Data       []library.Album `json:"data"`
		Next       string          `json:"next"`
		Previous   string          `json:"previous"`
		PagesCount int             `json:"pages_count"`
	}{
		Data:       albums,
		PagesCount: count / perPage,
		Next:       nextPage,
		Previous:   prevPage,
	}

	marshalled, err := json.Marshal(retData)

	if err != nil {
		return err
	}

	writer.Write(marshalled)

	return nil
}

func (bh BrowseHandler) browseArtists(writer http.ResponseWriter, page, perPage int) error {
	// In the API we count starting from 1. But actually for the library function pages
	// are counted from 0 which is much easier for implementing.
	artists, count := bh.library.BrowseArtists(uint(page-1), uint(perPage))

	prevPage := ""

	if page-1 > 0 {
		prevPage = fmt.Sprintf("/browse/?by=artist&page=%d&per-page=%d", page-1, perPage)
	}

	nextPage := ""

	if page*perPage < count {
		nextPage = fmt.Sprintf("/browse/?by=artist&page=%d&per-page=%d", page+1, perPage)
	}

	retData := struct {
		Data       []library.Artist `json:"data"`
		Next       string           `json:"next"`
		Previous   string           `json:"previous"`
		PagesCount int              `json:"pages_count"`
	}{
		Data:       artists,
		PagesCount: count / perPage,
		Next:       nextPage,
		Previous:   prevPage,
	}

	marshalled, err := json.Marshal(retData)

	if err != nil {
		return err
	}

	writer.Write(marshalled)

	return nil
}

func (bh BrowseHandler) badRequest(writer http.ResponseWriter, message string) {
	writer.WriteHeader(http.StatusBadRequest)
	msgJSON, _ := json.Marshal(struct {
		Error string `json:"error"`
	}{
		Error: message,
	})
	if _, err := writer.Write([]byte(msgJSON)); err != nil {
		log.Printf("error writing body in browse handler: %s", err)
	}
}

// NewBrowseHandler returns a new Browse handler. It needs a library to browse through.
func NewBrowseHandler(lib library.Library) *BrowseHandler {
	bh := new(BrowseHandler)
	bh.library = lib
	return bh
}
