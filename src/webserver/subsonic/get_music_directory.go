package subsonic

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
)

func (s *subsonic) getMusicDirectory(w http.ResponseWriter, req *http.Request) {
	dirIDString := req.URL.Query().Get("id")
	if dirIDString == "" {
		resp := responseError(10, "`id` was not present")
		encodeResponse(w, resp)
		return
	}
	dirID, err := strconv.ParseInt(dirIDString, 10, 64)
	if err != nil {
		resp := responseError(70, fmt.Sprintf("malformed `id`: %s", err))
		w.WriteHeader(http.StatusBadRequest)
		encodeResponse(w, resp)
		return
	}

	var (
		entry directoryEntry
	)

	if isArtistID(dirID) {
		entry, err = s.getArtistDirectory(req.Context(), toArtistDBID(dirID))
	} else if isAlbumID(dirID) {
		entry, err = s.getAlbumDirectory(req.Context(), toAlbumDBID(dirID))
	} else {
		resp := responseError(70, "no directory with this ID exists")
		encodeResponse(w, resp)
		return
	}

	if err != nil {
		resp := responseError(0, err.Error())
		encodeResponse(w, resp)
		return
	}

	resp := directoryResponse{
		baseResponse: responseOk(),
		Directory:    entry,
	}

	encodeResponse(w, resp)
}

func (s *subsonic) getArtistDirectory(
	_ context.Context,
	artistID int64,
) (directoryEntry, error) {
	artistSubsonicID := artistFSID(artistID)
	albums := s.lib.GetArtistAlbums(artistID)

	resp := directoryEntry{
		ID: artistSubsonicID,
	}

	for _, album := range albums {
		if resp.Name == "" {
			resp.Name = album.Artist
			resp.ParentID = combinedMusicFolderID
		}

		resp.Children = append(resp.Children, directoryChildEntry{
			ID:         albumFSID(album.ID),
			ParentID:   artistSubsonicID,
			Title:      album.Name,
			Artist:     album.Artist,
			IsDir:      true,
			CoverArtID: artistCoverArtID(artistID),
		})
	}

	return resp, nil
}

func (s *subsonic) getAlbumDirectory(
	_ context.Context,
	albumID int64,
) (directoryEntry, error) {
	albumSubsonicID := albumFSID(albumID)
	tracks := s.lib.GetAlbumFiles(albumID)

	resp := directoryEntry{
		ID: albumSubsonicID,
	}

	for _, track := range tracks {
		if resp.Name == "" {
			resp.Name = track.Album
			resp.ParentID = track.ArtistID
		}

		resp.Children = append(resp.Children, directoryChildEntry{
			ID:         trackFSID(track.ID),
			ParentID:   albumSubsonicID,
			Title:      track.Title,
			Artist:     track.Artist,
			Album:      track.Album,
			IsDir:      false,
			CoverArtID: strconv.FormatInt(albumID, 10),
			Track:      track.TrackNumber,
			Duration:   track.Duration / 1000,
			Suffix:     track.Format,
			Path: filepath.Join(
				track.Artist,
				track.Album,
				fmt.Sprintf("%s.%s", track.Title, track.Format),
			),
		})
	}

	return resp, nil
}

type directoryResponse struct {
	baseResponse

	Directory directoryEntry
}

type directoryEntry struct {
	XMLName  xml.Name `xml:"directory"`
	ID       int64    `xml:"id,attr"`
	ParentID int64    `xml:"parent,attr"`
	Name     string   `xml:"name,attr"`

	Children []directoryChildEntry
}

type directoryChildEntry struct {
	XMLName     xml.Name `xml:"child"`
	ID          int64    `xml:"id,attr"`
	ParentID    int64    `xml:"parent,attr"`
	Title       string   `xml:"title,attr,omitempty"`
	Artist      string   `xml:"artist,attr,omitempty"`
	Album       string   `xml:"album,attr,omitempty"`
	IsDir       bool     `xml:"isDir,attr"`
	CoverArtID  string   `xml:"coverArt,attr,omitempty"`
	Track       int64    `xml:"track,attr,omitempty"`    // position in album, I suppose
	Duration    int64    `xml:"duration,attr,omitempty"` // in seconds
	Year        int16    `xml:"year,attr,omitempty"`
	Genre       string   `xml:"genre,attr,omitempty"`
	Size        int64    `xml:"size,attr,omitempty"` // in bytes
	ContentType string   `xml:"contentType,attr,omitempty"`
	Suffix      string   `xml:"suffix,attr,omitempty"`
	BitRate     string   `xml:"bitRate,attr,omitempty"`
	Path        string   `xml:"path,attr,omitempty"` // on the file system I suppose
}
