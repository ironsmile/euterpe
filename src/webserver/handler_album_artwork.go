package webserver

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/ironsmile/euterpe/src/library"
)

// AlbumArtworkHandler is a http.Handler which will find and serve the artwork of
// a particular album.
type AlbumArtworkHandler struct {
	artworkManager library.ArtworkManager
	rootFS         fs.FS
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

	imgSize := library.OriginalImage
	if req.URL.Query().Get("size") == "small" {
		imgSize = library.SmallImage
	}

	imgReader, err := aah.artworkManager.FindAndSaveAlbumArtwork(ctx, id, imgSize)

	if err == library.ErrArtworkNotFound || os.IsNotExist(err) {
		writer.WriteHeader(http.StatusNotFound)
		notFoundImage, err := aah.rootFS.Open(aah.notFoundPath)
		if err == nil {
			defer notFoundImage.Close()
			_, _ = io.Copy(writer, notFoundImage)
		} else {
			log.Printf("Error opening not-found image: %s\n", err)
			fmt.Fprintln(writer, "404 image not found")
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
		_, _ = writer.Write([]byte("Uploaded artwork is too large."))
		return nil
	} else if _, ok := err.(*library.ArtworkError); ok {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(err.Error()))
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
	httpRootFS fs.FS,
	notFoundImagePath string,
) *AlbumArtworkHandler {

	return &AlbumArtworkHandler{
		rootFS:         httpRootFS,
		artworkManager: am,
		notFoundPath:   notFoundImagePath,
	}
}
