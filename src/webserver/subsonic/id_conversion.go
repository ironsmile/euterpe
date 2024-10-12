package subsonic

import (
	"fmt"
	"strconv"
)

const (
	// combinedMusicFolderID is selected so that the library will be allowed to have up
	// to one billion (1e9) artists, one billion (1e9) albums and `max(int64) - 2e9`
	// songs.
	combinedMusicFolderID int64 = 1e9
)

// trackFSID converts a trackID to the imaginary subsonic file structure ID for the
// track. It it supposed to be in the imaginary directory of its album.
func trackFSID(trackID int64) int64 {
	return trackID + 2*combinedMusicFolderID
}

// toTrackDBID converts an imaginary subsonic FS ID back to the ID of the track in
// the database.
func toTrackDBID(trackFSID int64) int64 {
	return trackFSID - 2*combinedMusicFolderID
}

// isTrackID returns true if id is an subsonic FS ID of a track.
func isTrackID(id int64) bool {
	return id > combinedMusicFolderID*2
}

// albumFSID converts an album ID to the imaginary subsonic file structure ID for this
// album where it is supposed to be inside the directory of its artist.
func albumFSID(albumID int64) int64 {
	return albumID
}

// toAlbumID converts from the imaginary subsonic FS album ID to the one in the
// database.
func toAlbumDBID(albumFSID int64) int64 {
	return albumFSID
}

// isAlbumID returns true if a given subsonic FS ID is an ID of an album.
func isAlbumID(id int64) bool {
	return id < combinedMusicFolderID
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
// exposed subsonic API. artistID is an ID from the database.
func artistCoverArtID(artistID int64) string {
	return fmt.Sprintf("%s%d", coverArtistPrefix, artistID)
}

// albumConverArtID converts the albumID to an ID for a cover image in the
// exposed subsonic API. albumID is an ID from the database.
func albumConverArtID(albumID int64) string {
	return fmt.Sprintf("%s%d", coverAlbumPrefix, albumID)
}

// musicFolderExists returns true if musicFolderID is a real music folder
// which could be found on the server.
//
// For the moment the only possible value is the string representation
// of the combaindMusicFolderID.
func musicFolderExists(musicFolderID string) bool {
	combindIDstr := strconv.FormatInt(combinedMusicFolderID, 10)
	return combindIDstr == musicFolderID
}

const (
	coverAlbumPrefix    = "al-"
	coverArtistPrefix   = "ar-"
	coverPlaylistPrefix = "pl-"
)
