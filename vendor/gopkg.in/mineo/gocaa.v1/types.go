package caa

// CoverArtInfo is the unmarshaled representation of a JSON file in the Cover Art Archive.
// See https://musicbrainz.org/doc/Cover_Art_Archive/API#Cover_Art_Archive_Metadata for an example.
type CoverArtInfo struct {
	Images  []CoverArtImageInfo
	Release string
}

// CoverArtImageInfo is the unmarshaled representation of a single images metadata in a CAA JSON file.
// See https://musicbrainz.org/doc/Cover_Art_Archive/API#Cover_Art_Archive_Metadata for an example.
type CoverArtImageInfo struct {
	Approved   bool
	Back       bool
	Comment    string
	Edit       int
	Front      bool
	ID         string
	Image      string
	Thumbnails ThumbnailMap
	Types      []string
}

// CoverArtImage is a wrapper around an image from the CAA, containing its binary data and mimetype information.
type CoverArtImage struct {
	Data     []byte
	Mimetype string
}

// ThumbnailMap maps thumbnail names to their URLs. The only valid keys are
// "large" and "small", "250", "500" and "1200".
type ThumbnailMap map[string]string
