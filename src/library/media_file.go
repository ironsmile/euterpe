package library

import (
	"fmt"
	"os"
	"time"

	"github.com/dhowden/tag"
	taglib "github.com/wtolson/go-taglib"
)

// MediaFile is an interface which a media object should satisfy in order to be inserted
// in the library database.
type MediaFile interface {

	// Artist returns a string which represents the artist responsible for this media file
	Artist() string

	// Album returns a string for the name of the album this media file is part of
	Album() string

	// Title returns the name of this piece of media
	Title() string

	// Track returns the media track number in its album
	Track() int

	// Length returns the duration of this piece of media
	Length() time.Duration

	// Year returns the four-digit year at which this media was recorded.
	Year() int

	// Returns the bitrate of the file in kb/s.
	Bitrate() int
}

// TaglibRead is a function which uses taglib to read a file.
type TaglibRead func(filename string) (*taglib.File, error)

// parseFileTags reads a file and returns its metadata tags as a MediaFile object.
func parseFileTags(readFunc TaglibRead, fileName string) (MediaFile, error) {
	file, tglErr := readFunc(fileName)
	if tglErr == nil {
		defer file.Close()
		return medaFileFromTaglib(file), nil
	}

	mf, tagErr := mediaFileFromTag(fileName)
	if tagErr != nil {
		return nil, fmt.Errorf(
			"failed to parse file with both tagging libs: (tag: %s, taglib: %w)",
			tagErr, tglErr,
		)
	}

	return mf, nil
}

type mediaFile struct {
	artist  string
	album   string
	title   string
	track   int
	length  time.Duration
	year    int
	bitrate int
}

func (f *mediaFile) Artist() string        { return f.artist }
func (f *mediaFile) Album() string         { return f.album }
func (f *mediaFile) Title() string         { return f.title }
func (f *mediaFile) Track() int            { return f.track }
func (f *mediaFile) Length() time.Duration { return f.length }
func (f *mediaFile) Year() int             { return f.year }
func (f *mediaFile) Bitrate() int          { return f.bitrate }

// medaFileFromTaglib returns a MediaFile from a taglib parsed file.
func medaFileFromTaglib(file *taglib.File) MediaFile {
	return &mediaFile{
		artist:  file.Artist(),
		album:   file.Album(),
		title:   file.Title(),
		track:   file.Track(),
		length:  file.Length(),
		year:    file.Year(),
		bitrate: file.Bitrate(),
	}
}

func mediaFileFromTag(fileName string) (MediaFile, error) {
	fh, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer fh.Close()

	md, err := tag.ReadFrom(fh)
	if err != nil {
		return nil, fmt.Errorf("parsing tags: %w", err)
	}

	track, _ := md.Track()
	file := &mediaFile{
		artist: md.Artist(),
		album:  md.Album(),
		title:  md.Title(),
		track:  track,
		year:   md.Year(),
	}

	return file, nil
}
