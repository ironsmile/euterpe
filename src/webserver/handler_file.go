package webserver

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/ironsmile/euterpe/src/library"
)

// FileHandler will find and serve a media file by its ID
type FileHandler struct {
	library library.Library
}

// ServeHTTP is required by the http.Handler's interface
func (fh FileHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	InternalErrorOnErrorHandler(writer, req, fh.find)
}

// Actually searches through the library for this file and serves it
// if it is found. Returns 404 if not (duh)
// Uses http.FileServer for serving the found files
func (fh FileHandler) find(writer http.ResponseWriter, req *http.Request) error {
	vars := mux.Vars(req)

	id, err := strconv.Atoi(vars["fileID"])
	if err != nil {
		http.NotFoundHandler().ServeHTTP(writer, req)
		return nil
	}

	if fh.library == nil {
		return fmt.Errorf("Library for FileHandler is nil")
	}

	filePath := fh.library.GetFilePath(int64(id))
	fileReader, err := os.Open(filePath)
	if err != nil {
		http.NotFoundHandler().ServeHTTP(writer, req)
		return nil
	}
	defer fileReader.Close()

	modTime := time.Time{}
	st, err := fileReader.Stat()
	if err == nil {
		modTime = st.ModTime()
	}

	err = fh.library.RecordTrackPlay(req.Context(), int64(id), time.Now())
	if err != nil {
		log.Printf("failed to update track %d stats: %s", id, err)
	}

	baseName := filepath.Base(filePath)
	writer.Header().Add("Content-Disposition",
		fmt.Sprintf("filename=\"%s\"", baseName))
	http.ServeContent(writer, req, baseName, modTime, fileReader)
	return nil
}

// NewFileHandler returns a new File handler will will be resposible for serving a file
// from the library identified from its ID.
func NewFileHandler(lib library.Library) *FileHandler {
	fh := new(FileHandler)
	fh.library = lib
	return fh
}
