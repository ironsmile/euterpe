package subsonic

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"

	"github.com/ironsmile/euterpe/src/version"
)

func encodeResponse(w http.ResponseWriter, req *http.Request, resp any) {
	if req.URL.Query().Get("f") == "json" {
		encodeResponseJSON(w, req, resp)
		return
	}

	encodeResponseXML(w, req, resp)
}

func encodeResponseJSON(w http.ResponseWriter, _ *http.Request, resp any) {
	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(jsonResponse{Response: resp}); err != nil {
		errMsg := fmt.Sprintf("failed to encode JSON: %s", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
}

func encodeResponseXML(w http.ResponseWriter, _ *http.Request, resp any) {
	w.Header().Set("Content-Type", "application/xml")

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")

	if err := enc.Encode(resp); err != nil {
		errMsg := fmt.Sprintf("failed to encode XML: %s", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
}

type jsonResponse struct {
	Response any `json:"subsonic-response"`
}

type baseResponse struct {
	XMLName       xml.Name `xml:"subsonic-response" json:"-"`
	XMLNS         string   `xml:"xmlns,attr" json:"-"`
	Status        string   `xml:"status,attr" json:"status"`
	Version       string   `xml:"version,attr" json:"version"`
	Type          string   `xml:"type,attr" json:"type"`
	ServerVersion string   `xml:"serverVersion,attr" json:"serverVersion"`
	OpenSubsonic  bool     `xml:"openSubsonic,attr" json:"openSubsonic"`
}

func responseOk() baseResponse {
	return baseResponse{
		XMLNS:         `http://subsonic.org/restapi`,
		Status:        "ok",
		Version:       "1.16.1",
		Type:          "euterpe",
		ServerVersion: version.Version,
		OpenSubsonic:  true,
	}
}

func responseFailed() baseResponse {
	return baseResponse{
		XMLNS:         `http://subsonic.org/restapi`,
		Status:        "failed",
		Version:       "1.16.1",
		Type:          "euterpe",
		ServerVersion: version.Version,
		OpenSubsonic:  true,
	}
}

type errorResponse struct {
	baseResponse

	Error errorElement `xml:"error" json:"error"`
}

type errorElement struct {
	Code    apiErrorCode `xml:"code,attr" json:"code"`
	Message string       `xml:"message,attr" json:"message"`
}

func responseError(code apiErrorCode, msg string) errorResponse {
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

// artistCoverArtID converts the artistID to an ID for cover image in the
// exposed subsonic API.
func artistCoverArtID(artistID int64) string {
	return fmt.Sprintf("%s%d", coverArtistPrefix, artistID)
}

// albumConverArtID converts the albumID to an ID for a cover image in the
// exposed subsonic API.
func albumConverArtID(albumID int64) string {
	return fmt.Sprintf("%s%d", coverAlbumPrefix, albumID)
}

// getProtoFromRequest returns the original request scheme used for accessing
// the server. It takes into account the X-Forwarded-Proto and the Forwarded
// HTTP headers.
func getProtoFromRequest(req *http.Request) string {
	proto := "http"
	if forwadedProto := req.Header.Get("X-Forwarded-Proto"); forwadedProto == "https" {
		proto = "https"
	}

	if forwarded := req.Header.Get("Forwarded"); forwarded != "" {
		vals := splitForwarded(forwarded)
		if forwardedProto, ok := vals["proto"]; ok && forwardedProto == "https" {
			proto = "https"
		}
	}

	return proto
}

// getHostFromRequest returns the original request Host used for accessing the
// server. It takes into account the X-Forwarded-Host and Forwarded HTTP headers.
func getHostFromRequest(req *http.Request) string {
	host := req.Host
	if forwadedHost := req.Header.Get("X-Forwarded-Host"); forwadedHost != "" {
		host = forwadedHost
	}

	if forwarded := req.Header.Get("Forwarded"); forwarded != "" {
		vals := splitForwarded(forwarded)
		if forwardedHost, ok := vals["host"]; ok {
			host = forwardedHost
		}
	}

	return host
}

func splitForwarded(val string) map[string]string {
	vals := make(map[string]string)

	// Example:
	// Forwarded: by=<identifier>;for=<identifier>;host=<host>;proto=<http|https>
	pairs := strings.Split(val, ";")
	for _, pair := range pairs {
		k, v, ok := strings.Cut(pair, "=")
		if !ok || v == "" {
			continue
		}

		vals[k] = strings.Trim(v, `"`)
	}

	return vals
}

const (
	coverAlbumPrefix  = "al-"
	coverArtistPrefix = "ar-"
)
