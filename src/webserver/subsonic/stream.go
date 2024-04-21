package subsonic

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

func (s *subsonic) stream(w http.ResponseWriter, req *http.Request) {
	idString := req.URL.Query().Get("id")
	trackID, err := strconv.ParseInt(idString, 10, 64)
	if idString == "" || err != nil {
		resp := responseError(70, "track not found")
		encodeResponse(w, req, resp)
		return
	}

	//!TODO: support maximum bitrate and and transcoding. Once done, a separate
	// endpoint must be created for the "/download" endpoint.

	filePath := s.lib.GetFilePath(trackID)

	_, err = os.Stat(filePath)
	if err != nil {
		http.NotFoundHandler().ServeHTTP(w, req)
		return
	}

	baseName := filepath.Base(filePath)

	w.Header().Add("Content-Disposition",
		fmt.Sprintf("filename=\"%s\"", baseName))

	req.URL.Path = "/" + baseName
	http.FileServer(http.Dir(filepath.Dir(filePath))).ServeHTTP(w, req)
}
