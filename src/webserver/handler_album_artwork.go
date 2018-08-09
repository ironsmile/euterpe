package webserver

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"

	"github.com/ironsmile/httpms/src/library"
)

// AlbumArtworkHandler is a http.Handler which will find and serve the artwork of
// a particular album.
type AlbumArtworkHandler struct {
	artworkManager library.ArtworkManager
	rootBox        packr.Box
	notFoundPath   string
}

// ServeHTTP is required by the http.Handler's interface
func (aah AlbumArtworkHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	idString, ok := vars["albumID"]
	if !ok {
		http.NotFoundHandler().ServeHTTP(writer, req)
		return
	}

	id, err := strconv.ParseInt(idString, 10, 64)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(writer, "Bad request. Parsing albumID: %s\n", err)
		return
	}

	if req.Method == http.MethodDelete {
		err = aah.remove(writer, req, id)
	} else if req.Method == http.MethodPut {
		err = aah.upload(writer, req, id)
	} else {
		err = aah.find(writer, req, id)
	}

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		if _, err := writer.Write([]byte(err.Error())); err != nil {
			log.Printf("error writing body in AlbumArtworkHandler: %s", err)
		}
	}
}

// Actually searches through the library for the artwork of an album and serves
// it as a raw image
func (aah AlbumArtworkHandler) find(
	writer http.ResponseWriter,
	req *http.Request,
	id int64,
) error {

	ctx, cancel := context.WithTimeout(req.Context(), 5*time.Minute)
	defer cancel()

	imgReader, err := aah.artworkManager.FindAndSaveAlbumArtwork(ctx, id)

	if err != nil && err == library.ErrArtworkNotFound || os.IsNotExist(err) {
		notFoundImage, err := aah.rootBox.Open(aah.notFoundPath)
		if err == nil {
			defer notFoundImage.Close()
			// !TODO: return Status Code Not Found here. But unfortunately
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
		log.Printf("Error finding album %d artwork: %s\n", id, err)
		return err
	}

	defer imgReader.Close()

	writer.Header().Set("Cache-Control", "max-age=604800")
	_, err = io.Copy(writer, imgReader)

	if err != nil {
		log.Printf("еrror sending HTTP data for artwork %d: %s", id, err)
	}

	return nil
}

func (aah AlbumArtworkHandler) remove(
	writer http.ResponseWriter,
	req *http.Request,
	id int64,
) error {
	if err := aah.artworkManager.RemoveAlbumArtwork(req.Context(), id); err != nil {
		return err
	}

	writer.WriteHeader(http.StatusNoContent)
	return nil
}

func (aah AlbumArtworkHandler) upload(
	writer http.ResponseWriter,
	req *http.Request,
	id int64,
) error {
	err := aah.artworkManager.SaveAlbumArtwork(req.Context(), id, req.Body)
	if err == library.ErrArtworkTooBig {
		writer.WriteHeader(413)
		writer.Write([]byte("Uploaded artwork is too large."))
		return nil
	} else if _, ok := err.(*library.ArtworkError); ok {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(err.Error()))
		return nil
	} else if err != nil {
		return err
	}

	writer.WriteHeader(http.StatusCreated)
	return nil
}

// NewAlbumArtworkHandler returns a new Album artwork handler.
// It needs an implementaion of the ArtworkManager.
func NewAlbumArtworkHandler(
	am library.ArtworkManager,
	rootBox packr.Box,
	notFoundImagePath string,
) *AlbumArtworkHandler {

	return &AlbumArtworkHandler{
		rootBox:        rootBox,
		artworkManager: am,
		notFoundPath:   notFoundImagePath,
	}
}
