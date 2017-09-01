package webserver

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ironsmile/httpms/src/library"
)

// AlbumHandler is a http.Handler which will find and serve a zip of the
// album by the album ID.
type AlbumHandler struct {
	library library.Library
}

// ServeHTTP is required by the http.Handler's interface
func (fh AlbumHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	InternalErrorOnErrorHandler(writer, req, fh.find)
}

// Actually searches through the library for this album
// Will serve it as zip file with name "[AlbumName].zip". The zip will contain
// all the files for this album.
func (fh AlbumHandler) find(writer http.ResponseWriter, req *http.Request) error {

	id, err := strconv.Atoi(req.URL.Path)

	if err != nil {
		http.NotFoundHandler().ServeHTTP(writer, req)
		return nil
	}

	albumFiles := fh.library.GetAlbumFiles(int64(id))

	if len(albumFiles) < 1 {
		http.NotFoundHandler().ServeHTTP(writer, req)
		return nil
	}

	writer.Header().Add("Content-Disposition",
		fmt.Sprintf(`filename="%s.zip"`, albumFiles[0].Album))

	var files []string

	for _, track := range albumFiles {
		files = append(files, fh.library.GetFilePath(track.ID))
	}

	fh.writeZipContents(writer, files)

	return nil
}

// Zips all files in `files` and writes the output in the `writer`. The name of
// every file is its filepath.Base.
func (fh AlbumHandler) writeZipContents(writer io.Writer, files []string) error {

	zipWriter := zip.NewWriter(writer)

	for _, file := range files {
		fh, err := os.Open(file)

		if err != nil {
			_ = zipWriter.Close()
			return err
		}

		defer fh.Close()

		zfh, err := zipWriter.Create(filepath.Base(file))
		if err != nil {
			_ = zipWriter.Close()
			return err
		}

		_, err = io.Copy(zfh, fh)
		if err != nil {
			_ = zipWriter.Close()
			return err
		}
	}

	return zipWriter.Close()
}

// NewAlbumHandler returns a new Album handler. It needs a library to search in
func NewAlbumHandler(lib library.Library) *AlbumHandler {
	fh := new(AlbumHandler)
	fh.library = lib
	return fh
}
