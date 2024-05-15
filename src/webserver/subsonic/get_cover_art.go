package subsonic

import (
    "log"
    "net/http"
    "strconv"
    "strings"
)

func (s *subsonic) getCoverArt(w http.ResponseWriter, req *http.Request) {
    id := req.Form.Get("id")
    size := req.Form.Get("size")

    var artworkHandler CoverArtHandler
    if strings.HasPrefix(id, coverAlbumPrefix) {
        artworkHandler = s.albumArtHandler
        id = strings.TrimPrefix(id, coverAlbumPrefix)
    } else if strings.HasPrefix(id, coverArtistPrefix) {
        artworkHandler = s.artistArtHandler
        id = strings.TrimPrefix(id, coverArtistPrefix)
    } else if albumID := isAlbumIDString(id); albumID != "" {
        artworkHandler = s.albumArtHandler
        id = albumID
    } else if artistID := isArtistIDString(id); artistID != "" {
        artworkHandler = s.artistArtHandler
        id = artistID
    } else {
        w.WriteHeader(http.StatusNotFound)
        return
    }

    dbArtID, err := strconv.ParseInt(id, 10, 64)
    if err != nil {
        w.WriteHeader(http.StatusNotFound)
        if _, err := w.Write([]byte(err.Error())); err != nil {
            log.Printf("unrecognized ID format: %s", err)
        }
        return
    }

    if sizePx, err := strconv.ParseInt(size, 10, 64); err == nil && sizePx < 200 {
        query := req.Form
        query.Set("size", "small")
        req.URL.RawQuery = query.Encode()
        req.Method = http.MethodGet
    }

    err = artworkHandler.Find(w, req, dbArtID)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        if _, err := w.Write([]byte(err.Error())); err != nil {
            log.Printf("error writing body in getCoverArt: %s", err)
        }
    }
}

func isAlbumIDString(subsonicID string) string {
    id, err := strconv.ParseInt(subsonicID, 10, 64)
    if err != nil {
        return ""
    }
    if !isAlbumID(id) {
        return ""
    }
    return strconv.FormatInt(toAlbumDBID(id), 10)
}

func isArtistIDString(subsonicID string) string {
    id, err := strconv.ParseInt(subsonicID, 10, 64)
    if err != nil {
        return ""
    }
    if !isArtistID(id) {
        return ""
    }
    return strconv.FormatInt(toArtistDBID(id), 10)
}
