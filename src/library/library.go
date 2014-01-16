// This module deals with the actual media library. It is creates the Library type.
//
// Every media receives an ID in the library. The main thing a search result returns
// is the tracks' IDs. They are used to get the media, again using the Library. That
// way the real location of the file is never revealed to the interface.
package library

// Contains a result for a search term. Contains all the neccessery information to
// uniquely identify a media in the library.
type SearchResult struct {
	ID          int64  // ID in the library for a media file
	Artist      string // Meta info: Artist
	Collection  string // Meta info: Album for music
	Title       string // Meta info: the title of this media file
	TrackNumber int64  // Meta info: track number for music
}

// This type represents the media library which is played using the HTTPMS.
// It is responsible for scaning the library directories, watching for new files,
// actually searching for a media by a search term and finding the exact file path
// in the file system for a media.
type Library interface {

	// Adds a new path to the library paths. If it hasn't been scanned yet a new scan
	// will be started.
	AddLibraryPath(string)

	// Search the library using a search string. It will match against Artist, Collection
	// and Title. Will OR the results. So it is "return anything whcih Artist maches or
	// Collection matches or Title matches"
	Search(string) []SearchResult

	// Returns the real filesystem path. Requires the media ID.
	GetFilePath(int64) string

	// Starts a background library scan. Will scan all paths if
	// they are not scanned already
	Scan() error

	// Adds this media (file) to the library
	AddMedia(string) error

	// Makes sure the library is initialied. This method will be called once on
	// every start of the httpms
	Initialize() error

	// Makes the library forget everything. Also Closes the library.
	Truncate() error

	// Frees all resources this library object is using.
	// Any operations (except Truncate) on closed library will result in panic.
	Close()
}
