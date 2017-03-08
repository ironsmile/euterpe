// Go wrapper for taglib

// Generate stringer method for types
//go:generate stringer -type=TagName

package taglib

// #cgo pkg-config: taglib
// #cgo LDFLAGS: -ltag_c
// #include <stdlib.h>
// #include <tag_c.h>
import "C"

import (
	"errors"
	"strconv"
	"sync"
	"time"
	"unsafe"
)

type TagName int

// Tag names
const (
	Album TagName = iota
	Artist
	Bitrate
	Channels
	Comments
	Genre
	Length
	Samplerate
	Title
	Track
	Year
)

var (
	ErrInvalid = errors.New("invalid file")
	glock      = sync.Mutex{}
)

// Returns a string with this tag's comment.
func (file *File) Tag(tagname TagName) (tagvalue string) {
	switch tagname {
	case Album:
		return file.Album()
	case Artist:
		return file.Artist()
	case Bitrate:
		return strconv.Itoa(file.Bitrate())
	case Channels:
		return strconv.Itoa(file.Channels())
	case Comments:
		return file.Comment()
	case Genre:
		return file.Genre()
	case Length:
		return file.Length().String()
	case Samplerate:
		return strconv.Itoa(file.Samplerate())
	case Title:
		return file.Title()
	case Track:
		return strconv.Itoa(file.Track())
	case Year:
		return strconv.Itoa(file.Year())
	}
	return ""
}

// Sets the tag.
func (file *File) SetTag(tagname TagName, tagvalue string) {
	switch tagname {
	case Album:
		file.SetAlbum(tagvalue)
	case Artist:
		file.SetArtist(tagvalue)
	case Comments:
		file.SetComment(tagvalue)
	case Genre:
		file.SetGenre(tagvalue)
	case Title:
		file.SetTitle(tagvalue)
	case Track:
		intValue, convErr := strconv.Atoi(tagvalue)
		if convErr == nil {
			file.SetTrack(intValue)
		}
	case Year:
		intValue, convErr := strconv.Atoi(tagvalue)
		if convErr == nil {
			file.SetYear(intValue)
		}
	}
}

type File struct {
	fp    *C.TagLib_File
	tag   *C.TagLib_Tag
	props *C.TagLib_AudioProperties
}

// Reads and parses a music file. Returns an error if the provided filename is
// not a valid file.
func Read(filename string) (*File, error) {
	glock.Lock()
	defer glock.Unlock()

	cs := C.CString(filename)
	defer C.free(unsafe.Pointer(cs))

	fp := C.taglib_file_new(cs)
	if fp == nil || C.taglib_file_is_valid(fp) == 0 {
		return nil, ErrInvalid
	}

	return &File{
		fp:    fp,
		tag:   C.taglib_file_tag(fp),
		props: C.taglib_file_audioproperties(fp),
	}, nil
}

// Close and free the file.
func (file *File) Close() {
	glock.Lock()
	defer glock.Unlock()

	C.taglib_file_free(file.fp)
	file.fp = nil
	file.tag = nil
	file.props = nil
}

func convertAndFree(cs *C.char) string {
	if cs == nil {
		return ""
	}

	defer C.free(unsafe.Pointer(cs))
	return C.GoString(cs)
}

// Returns a string with this tag's title.
func (file *File) Title() string {
	glock.Lock()
	defer glock.Unlock()

	return convertAndFree(C.taglib_tag_title(file.tag))
}

// Returns a string with this tag's artist.
func (file *File) Artist() string {
	glock.Lock()
	defer glock.Unlock()

	return convertAndFree(C.taglib_tag_artist(file.tag))
}

// Returns a string with this tag's album name.
func (file *File) Album() string {
	glock.Lock()
	defer glock.Unlock()

	return convertAndFree(C.taglib_tag_album(file.tag))
}

// Returns a string with this tag's comment.
func (file *File) Comment() string {
	glock.Lock()
	defer glock.Unlock()

	return convertAndFree(C.taglib_tag_comment(file.tag))
}

// Returns a string with this tag's genre.
func (file *File) Genre() string {
	glock.Lock()
	defer glock.Unlock()

	return convertAndFree(C.taglib_tag_genre(file.tag))
}

// Returns the tag's year or 0 if year is not set.
func (file *File) Year() int {
	glock.Lock()
	defer glock.Unlock()

	return int(C.taglib_tag_year(file.tag))
}

// Returns the tag's track number or 0 if track number is not set.
func (file *File) Track() int {
	glock.Lock()
	defer glock.Unlock()

	return int(C.taglib_tag_track(file.tag))
}

// Returns the length of the file.
func (file *File) Length() time.Duration {
	glock.Lock()
	defer glock.Unlock()

	length := C.taglib_audioproperties_length(file.props)
	return time.Duration(length) * time.Second
}

// Returns the bitrate of the file in kb/s.
func (file *File) Bitrate() int {
	glock.Lock()
	defer glock.Unlock()

	return int(C.taglib_audioproperties_bitrate(file.props))
}

// Returns the sample rate of the file in Hz.
func (file *File) Samplerate() int {
	glock.Lock()
	defer glock.Unlock()

	return int(C.taglib_audioproperties_samplerate(file.props))
}

// Returns the number of channels in the audio stream.
func (file *File) Channels() int {
	glock.Lock()
	defer glock.Unlock()

	return int(C.taglib_audioproperties_channels(file.props))
}

func init() {
	glock.Lock()
	defer glock.Unlock()

	C.taglib_set_string_management_enabled(0)
}

// Saves the \a file to disk.
func (file *File) Save() error {
	var err error
	glock.Lock()
	defer glock.Unlock()
	if C.taglib_file_save(file.fp) != 1 {
		err = errors.New("Cannot save file")
	}
	return err
}

// Sets the tag's title.
func (file *File) SetTitle(s string) {
	glock.Lock()
	defer glock.Unlock()
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	C.taglib_tag_set_title(file.tag, cs)

}

// Sets the tag's artist.
func (file *File) SetArtist(s string) {
	glock.Lock()
	defer glock.Unlock()
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	C.taglib_tag_set_artist(file.tag, cs)
}

// Sets the tag's album.
func (file *File) SetAlbum(s string) {
	glock.Lock()
	defer glock.Unlock()
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	C.taglib_tag_set_album(file.tag, cs)
}

// Sets the tag's comment.
func (file *File) SetComment(s string) {
	glock.Lock()
	defer glock.Unlock()
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	C.taglib_tag_set_comment(file.tag, cs)
}

// Sets the tag's genre.
func (file *File) SetGenre(s string) {
	glock.Lock()
	defer glock.Unlock()
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	C.taglib_tag_set_genre(file.tag, cs)
}

// Sets the tag's year.  0 indicates that this field should be cleared.
func (file *File) SetYear(i int) {
	glock.Lock()
	defer glock.Unlock()
	ci := C.uint(i)
	C.taglib_tag_set_year(file.tag, ci)
}

// Sets the tag's track number.  0 indicates that this field should be cleared.
func (file *File) SetTrack(i int) {
	glock.Lock()
	defer glock.Unlock()
	ci := C.uint(i)
	C.taglib_tag_set_track(file.tag, ci)
}

