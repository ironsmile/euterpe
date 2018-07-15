package webserver

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ironsmile/httpms/src/library"
)

// AlbumArtworkHandler is a http.Handler which will find and serve the artwork of
// a particular album.
type AlbumArtworkHandler struct {
	artworkFinder library.ArtworkFinder
	notFoundPath  string
}

// ServeHTTP is required by the http.Handler's interface
func (aah AlbumArtworkHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	WithInternalError(aah.find)(writer, req)
}

// Actually searches through the library for the artwork of an album and serves
// it as a raw image
func (aah AlbumArtworkHandler) find(writer http.ResponseWriter, req *http.Request) error {

	vars := mux.Vars(req)
	idString, ok := vars["albumID"]

	if !ok {
		http.NotFoundHandler().ServeHTTP(writer, req)
		return nil
	}

	id, err := strconv.Atoi(idString)

	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(writer, "Bad request. Parsing albumID: %s\n", err)
		return nil
	}

	imgReader, err := aah.artworkFinder.FindAndSaveAlbumArtwork(int64(id))

	if err != nil && err == library.ErrArtworkNotFound {
		notFoundImage, err := os.Open(aah.notFoundPath)
		if err == nil {
			defer notFoundImage.Close()
			//!TODO: return Status Code Not Found here. But unfortunately
			// because of the gzip handler on WriteHeader here the gzip
			// headers could not be send as well. We need some deferred response
			// writer here. One which caches its WriteHeader status code and
			// sends it only once Write is called.
			// writer.WriteHeader(http.StatusNotFound)
			_, _ = io.Copy(writer, notFoundImage)
		} else {
			log.Printf("Error opening not-found image: %s\n", err)
			writer.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(writer, "404 page not found")
		}
		return nil
	}

	if err != nil {
		log.Printf("Error finding album artwork: %s\n", err)
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
func NewAlbumArtworkHandler(
	artworkFinder library.ArtworkFinder,
	notFoundImagePath string,
) *AlbumArtworkHandler {
	return &AlbumArtworkHandler{
		artworkFinder: artworkFinder,
		notFoundPath:  notFoundImagePath,
	}
}
