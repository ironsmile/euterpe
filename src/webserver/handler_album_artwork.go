package webserver

import (
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ironsmile/httpms/src/library"
)

// AlbumArtworkHandler is a http.Handler which will find and serve the artwork of
// a particular album.
type AlbumArtworkHandler struct {
	artworkFinder library.ArtworkFinder
}

// ServeHTTP is required by the http.Handler's interface
func (fh AlbumArtworkHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	WithInternalError(fh.find)(writer, req)
}

// Actually searches through the library for the artwork of an album and serves
// it as a raw image
func (fh AlbumArtworkHandler) find(writer http.ResponseWriter, req *http.Request) error {

	vars := mux.Vars(req)
	idString, ok := vars["albumID"]

	if !ok {
		http.NotFoundHandler().ServeHTTP(writer, req)
		return nil
	}

	id, err := strconv.Atoi(idString)

	if err != nil {
		http.NotFoundHandler().ServeHTTP(writer, req)
		return nil
	}

	imgReader, err := fh.artworkFinder.GetAlbumArtwork(int64(id))

	if err != nil && err == library.ErrArtworkNotFound {
		http.NotFoundHandler().ServeHTTP(writer, req)
		return nil
	}

	if err != nil {
		return err
	}

	defer imgReader.Close()

	_, err = io.Copy(writer, imgReader)

	if err != nil {
		log.Printf("еrror sending HTTP data for artwork %d: %s", id, err)
	}

	return nil
}

// NewAlbumArtworkHandler returns a new Album artwork handler.
// It needs an implementaion of the ArtworkFinder.
func NewAlbumArtworkHandler(artworkFinder library.ArtworkFinder) *AlbumArtworkHandler {
	aah := new(AlbumArtworkHandler)
	aah.artworkFinder = artworkFinder
	return aah
}
