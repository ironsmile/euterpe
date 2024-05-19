package subsonic

import "net/http"

func (s *subsonic) getAlbumInfo(w http.ResponseWriter, req *http.Request) {
	s.getAlbumInfo2(w, req)
}
