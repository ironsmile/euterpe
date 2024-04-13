package subsonic

import (
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/ironsmile/euterpe/src/version"
)

func encodeResponse(w http.ResponseWriter, resp any) {
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")

	if err := enc.Encode(resp); err != nil {
		errMsg := fmt.Sprintf("faild to encode XML: %s", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
}

type baseResponse struct {
	XMLName xml.Name `xml:"subsonic-response"`
	Status  string   `xml:"status,attr"`
	Version string   `xml:"version,attr"`
}

func responseOk() baseResponse {
	return baseResponse{
		Status:  "ok",
		Version: version.Version,
	}
}

func responseFailed() baseResponse {
	return baseResponse{
		Status:  "failed",
		Version: version.Version,
	}
}

type errorResponse struct {
	baseResponse

	Error errorElement
}

type errorElement struct {
	XMLName xml.Name `xml:"error"`
	Code    int      `xml:"code,attr"`
	Message string   `xml:"message,attr"`
}

func responseError(code int, msg string) errorResponse {
	return errorResponse{
		baseResponse: responseFailed(),
		Error: errorElement{
			Code:    code,
			Message: msg,
		},
	}
}

const (
	combinedMusicFolderID int64 = 1e9
)

// trackFSID converts a trackID to the imaginary subsonic file structure ID for the
// track. It it supposed to be in the imaginary directory of its album.
func trackFSID(trackID int64) int64 {
	return trackID
}

// toTrackDBID converts an imaginary subsonic FS ID back to the ID of the track in
// the database.
func toTrackDBID(trackFSID int64) int64 {
	return trackFSID
}

// isTrackID returns true if id is an subsonic FS ID of a track.
func isTrackID(id int64) bool {
	return id < combinedMusicFolderID
}

// albumFSID converts an album ID to the imaginary subsonic file structure ID for this
// album where it is supposed to be inside the directory of its artist.
func albumFSID(albumID int64) int64 {
	return 2*combinedMusicFolderID + albumID
}

// toAlbumID converts from the imaginary subsonic FS album ID to the one in the
// database.
func toAlbumDBID(albumFSID int64) int64 {
	return albumFSID - 2*combinedMusicFolderID
}

// isAlbumID returns true if a given subsonic FS ID is an ID of an album.
func isAlbumID(id int64) bool {
	return id > 2*combinedMusicFolderID
}

// artistFSID converts an artist ID to the imaginary subsonic file structure ID for
// this artist where all artists are top level directories in the combined music
// folder.
func artistFSID(artistID int64) int64 {
	return combinedMusicFolderID + artistID
}

// toArtistDBID converts from the imaginary subsonic FS album ID to the one in the
// database.
func toArtistDBID(artistFSID int64) int64 {
	return artistFSID - combinedMusicFolderID
}

// isArtistID returns true if a given subsonic FS ID is an ID of an artist.
func isArtistID(id int64) bool {
	return id > combinedMusicFolderID && id <= 2*combinedMusicFolderID
}

func artistCoverArtID(artistID int64) string {
	return fmt.Sprintf("ar-%d", artistID)
}
