package subsonic

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func (s *subsonic) stream(w http.ResponseWriter, req *http.Request) {
	idString := req.Form.Get("id")
	trackID, err := strconv.ParseInt(idString, 10, 64)
	if idString == "" || err != nil {
		resp := responseError(errCodeNotFound, "track not found")
		encodeResponse(w, req, resp)
		return
	}

	//!TODO: support maximum bitrate and and transcoding. Once done, a separate
	// endpoint must be created for the "/download" endpoint.

	filePath := s.lib.GetFilePath(trackID)

	fh, err := os.Open(filePath)
	if err != nil {
		http.NotFoundHandler().ServeHTTP(w, req)
		return
	}
	defer fh.Close()

	modTime := time.Time{}
	st, err := fh.Stat()
	if err == nil {
		modTime = st.ModTime()
	}

	baseName := filepath.Base(filePath)

	w.Header().Add("Content-Disposition",
		fmt.Sprintf("filename=\"%s\"", baseName))

	http.ServeContent(w, req, baseName, modTime, fh)
}
