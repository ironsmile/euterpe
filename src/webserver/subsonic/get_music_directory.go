package subsonic

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) getMusicDirectory(w http.ResponseWriter, req *http.Request) {
	dirIDString := req.Form.Get("id")
	if dirIDString == "" {
		resp := responseError(errCodeNotFound, "directory not found")
		encodeResponse(w, req, resp)
		return
	}
	dirID, err := strconv.ParseInt(dirIDString, 10, 64)
	if err != nil {
		resp := responseError(errCodeNotFound, fmt.Sprintf("malformed `id`: %s", err))
		encodeResponse(w, req, resp)
		return
	}

	var (
		entry directoryEntry
	)

	if dirID == combinedMusicFolderID {
		entry, err = s.getRootDirectory(req.Context())
	} else if isArtistID(dirID) {
		entry, err = s.getArtistDirectory(req.Context(), toArtistDBID(dirID))
	} else if isAlbumID(dirID) {
		entry, err = s.getAlbumDirectory(req.Context(), toAlbumDBID(dirID))
	} else {
		resp := responseError(errCodeNotFound, "no directory with this ID exists")
		encodeResponse(w, req, resp)
		return
	}

	if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	resp := directoryResponse{
		baseResponse: responseOk(),
		Directory:    entry,
	}

	encodeResponse(w, req, resp)
}

func (s *subsonic) getArtistDirectory(
	_ context.Context,
	artistID int64,
) (directoryEntry, error) {
	artistSubsonicID := artistFSID(artistID)
	albums := s.lib.GetArtistAlbums(artistID)

	resp := directoryEntry{
		ID:         artistSubsonicID,
		AlbumCount: int64(len(albums)),
		CoverArtID: artistCoverArtID(artistID),
	}

	for _, album := range albums {
		if resp.Name == "" {
			resp.Name = album.Artist
		}
		resp.PlayCount += album.Plays

		resp.Children = append(resp.Children, albumToDirChild(
			album,
			artistID,
			s.lastModified,
		))
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
		ID:         albumSubsonicID,
		SongCount:  int64(len(tracks)),
		CoverArtID: albumConverArtID(albumID),
	}

	var totalDur int64
	for _, track := range tracks {
		if resp.Name == "" {
			resp.Name = track.Album
			resp.ParentID = artistFSID(track.ArtistID)
			resp.Artist = track.Artist
		}

		if resp.Artist != track.Artist {
			resp.Artist = "Various Artists"
		}

		resp.PlayCount += track.Plays
		totalDur += track.Duration / 1000

		resp.Children = append(resp.Children, trackToDirChild(track, s.lastModified))
	}

	resp.Duration = totalDur

	return resp, nil
}

func (s *subsonic) getRootDirectory(
	_ context.Context,
) (directoryEntry, error) {
	var (
		page uint = 0
		resp      = directoryEntry{
			ID:       combinedMusicFolderID,
			ParentID: 0,
			Name:     "Combined Music Library",
		}
	)
	for {
		artists, _ := s.libBrowser.BrowseArtists(library.BrowseArgs{
			Page:    page,
			PerPage: 500,
			Order:   library.OrderAsc,
			OrderBy: library.OrderByName,
		})

		if len(artists) == 0 {
			break
		}

		for _, artist := range artists {
			if artist.Name == "" {
				continue
			}

			resp.Children = append(
				resp.Children,
				directoryChildEntry{
					Name:          artist.Name,
					Artist:        artist.Name,
					Title:         artist.Name,
					ID:            artistFSID(artist.ID),
					MediaType:     "artist",
					DirectoryType: "music",
					ParentID:      combinedMusicFolderID,
					IsDir:         true,
					Created:       s.lastModified,
				},
			)
		}

		page++
	}

	return resp, nil
}

type directoryResponse struct {
	baseResponse

	Directory directoryEntry `xml:"directory,omitempty" json:"directory,omitempty"`
}

type directoryEntry struct {
	ID         int64  `xml:"id,attr" json:"id,string"`
	ParentID   int64  `xml:"parent,attr,omitempty" json:"parent,omitempty,string"`
	Artist     string `xml:"-" json:"-"`
	Name       string `xml:"name,attr" json:"name"`
	AlbumCount int64  `xml:"-" json:"albumCount,omitempty"`
	SongCount  int64  `xml:"-" json:"songCount,omitempty"`
	CoverArtID string `xml:"-" json:"coverArt,omitempty"`
	Duration   int64  `xml:"-" json:"duration,omitempty"` // in seconds
	PlayCount  int64  `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	UserRating uint8  `xml:"userRating,attr,omitempty" json:"userRating,omitempty"`

	Children []directoryChildEntry `xml:"child" json:"child"`
}

type directoryChildEntry struct {
	ID            int64     `xml:"id,attr" json:"id,string"`
	ParentID      int64     `xml:"parent,attr,omitempty" json:"parent,omitempty,string"`
	MediaType     string    `xml:"-" json:"mediaType"`
	DirectoryType string    `xml:"type,attr,omitempty" json:"type,omitempty"`
	Title         string    `xml:"title,attr,omitempty" json:"title"`
	Name          string    `xml:"-" json:"-"`
	Artist        string    `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	ArtistID      int64     `xml:"-" json:"artistId,omitempty,string"`
	Album         string    `xml:"album,attr,omitempty" json:"album"`
	AlbumID       int64     `xml:"albumId,attr,omitempty" json:"albumId,omitempty,string"`
	IsDir         bool      `xml:"isDir,attr" json:"isDir"`
	IsVideo       bool      `xml:"isVideo,attr,omitempty" json:"isVideo"`
	CoverArtID    string    `xml:"coverArt,attr,omitempty" json:"coverArt"`
	Track         int64     `xml:"track,attr,omitempty" json:"track,omitempty"`       // position in album, I suppose
	Duration      int64     `xml:"duration,attr,omitempty" json:"duration,omitempty"` // in seconds
	Year          int16     `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre         string    `xml:"genre,attr,omitempty" json:"gener,omitempty"`
	Size          int64     `xml:"size,attr,omitempty" json:"size,omitempty"` // in bytes
	ContentType   string    `xml:"contentType,attr,omitempty" json:"contentType,omitempty"`
	SongCount     int64     `xml:"-" json:"songCount,omitempty"`
	PlayCount     int64     `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	UserRating    uint8     `xml:"userRating,attr,omitempty" json:"userRating,omitempty"`
	Suffix        string    `xml:"suffix,attr,omitempty" json:"suffix,omitempty"`
	BitRate       string    `xml:"bitRate,attr,omitempty" json:"bitRate,omitempty"`
	Path          string    `xml:"path,attr,omitempty" json:"path,omitempty"` // on the file system I suppose
	Created       time.Time `xml:"created,attr,omitempty" json:"created,omitempty"`
}

func trackToDirChild(track library.TrackInfo, created time.Time) directoryChildEntry {
	return directoryChildEntry{
		ID:            trackFSID(track.ID),
		ParentID:      albumFSID(track.AlbumID),
		MediaType:     "song",
		DirectoryType: "music",
		Title:         track.Title,
		Name:          track.Title,
		Artist:        track.Artist,
		ArtistID:      artistFSID(track.ArtistID),
		Album:         track.Album,
		AlbumID:       albumFSID(track.AlbumID),
		IsDir:         false,
		CoverArtID:    albumConverArtID(track.AlbumID),
		Track:         track.TrackNumber,
		Duration:      track.Duration / 1000,
		Suffix:        track.Format,
		Path: filepath.Join(
			track.Artist,
			track.Album,
			fmt.Sprintf("%s.%s", track.Title, track.Format),
		),
		Created:    created,
		PlayCount:  track.Plays,
		UserRating: track.Rating,

		// Here we take advantage of the knowledge that the track.Format is just
		// the file name extension.
		ContentType: mime.TypeByExtension(filepath.Ext("." + track.Format)),
	}
}

// albumToDirChild converts a library Album to a directory child entry.
// artistID is a in-db library ID.
//
// If artistID is empty then ParentID and ArtistID properties of the child
// will not be set.
func albumToDirChild(
	album library.Album,
	artistID int64,
	created time.Time,
) directoryChildEntry {
	entry := directoryChildEntry{
		ID:            albumFSID(album.ID),
		MediaType:     "album",
		DirectoryType: "music",
		Title:         album.Name,
		Name:          album.Name,
		Album:         album.Name,
		AlbumID:       albumFSID(album.ID),
		Artist:        album.Artist,
		IsDir:         true,
		CoverArtID:    albumConverArtID(album.ID),
		SongCount:     album.SongCount,
		Created:       created,
		Duration:      album.Duration / 1000,
	}

	if artistID != 0 {
		artistSubsonicID := artistFSID(artistID)
		entry.ParentID = artistSubsonicID
		entry.ArtistID = artistSubsonicID
	}

	return entry
}

func artistToDirChild(
	artist library.Artist,
	created time.Time,
) directoryChildEntry {
	return directoryChildEntry{
		ID:            albumFSID(artist.ID),
		MediaType:     "artist",
		DirectoryType: "music",
		Name:          artist.Name,
		AlbumID:       albumFSID(artist.ID),
		Artist:        artist.Name,
		IsDir:         true,
		CoverArtID:    artistCoverArtID(artist.ID),
		Created:       created,
	}
}
