package webserver

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ironsmile/euterpe/src/library"
)

// AlbumHandler is a http.Handler which will find and serve a zip of the
// album by the album ID.
type AlbumHandler struct {
	library library.Library
}

// ServeHTTP is required by the http.Handler's interface
func (fh AlbumHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	WithInternalError(fh.find)(writer, req)
}

// Actually searches through the library for this album
// Will serve it as zip file with name "[AlbumName].zip". The zip will contain
// all the files for this album.
func (fh AlbumHandler) find(writer http.ResponseWriter, req *http.Request) error {

	vars := mux.Vars(req)
	idString, ok := vars["albumID"]

	if !ok {
		http.NotFoundHandler().ServeHTTP(writer, req)
		return nil
	}

	id, err := strconv.Atoi(idString)

	if err != nil {
		http.Error(
			writer,
			fmt.Sprintf(
				"Parsing albumID in request path failed: %s",
				err,
			),
			http.StatusBadRequest,
		)
		return nil
	}

	albumFiles := fh.library.GetAlbumFiles(req.Context(), int64(id))

	if len(albumFiles) < 1 {
		http.NotFoundHandler().ServeHTTP(writer, req)
		return nil
	}

	writer.Header().Add("Content-Disposition",
		fmt.Sprintf(`filename="%s.zip"`, albumFiles[0].Album))

	var files []string

	for _, track := range albumFiles {
		files = append(files, fh.library.GetFilePath(req.Context(), track.ID))
	}

	written, err := fh.writeZipContents(writer, files)
	if err != nil && written == 0 {
		// Return the error only in case there have been no bytes written in
		// the response. Only then will the server be able to respond with
		// any other status code. In case we're written some bytes and there
		// are errors clients are left to fend on their own.
		//
		//!IDEA: maybe define a Trailer header which says "request went OK" for
		// generated ZIP files. Since clients have no way of knowing the ZIP's
		// Content-Length at the beginning of the download this might be a good
		// way to signal to them the error.
		return err
	}

	return nil
}

// Zips all files in `files` and writes the output in the `writer`. The name of
// every file is its filepath.Base.
func (fh AlbumHandler) writeZipContents(writer io.Writer, files []string) (int64, error) {

	var written int64
	zipWriter := zip.NewWriter(writer)

	for _, file := range files {
		fh, err := os.Open(file)

		if err != nil {
			_ = zipWriter.Close()
			return written, err
		}

		defer fh.Close()

		zfh, err := zipWriter.Create(filepath.Base(file))
		if err != nil {
			_ = zipWriter.Close()
			return written, err
		}

		n, err := io.Copy(zfh, fh)
		if err != nil {
			_ = zipWriter.Close()
			return written, err
		}

		written += n
	}

	return written, zipWriter.Close()
}

// NewAlbumHandler returns a new Album handler. It needs a library to search in
func NewAlbumHandler(lib library.Library) *AlbumHandler {
	fh := new(AlbumHandler)
	fh.library = lib
	return fh
}
