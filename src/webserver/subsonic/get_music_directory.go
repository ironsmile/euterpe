package subsonic

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/ironsmile/euterpe/src/library"
)

func (s *subsonic) getMusicDirectory(w http.ResponseWriter, req *http.Request) {
	dirIDString := req.URL.Query().Get("id")
	if dirIDString == "" {
		resp := responseError(70, "directory not found")
		encodeResponse(w, req, resp)
		return
	}
	dirID, err := strconv.ParseInt(dirIDString, 10, 64)
	if err != nil {
		resp := responseError(70, fmt.Sprintf("malformed `id`: %s", err))
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
		resp := responseError(70, "no directory with this ID exists")
		encodeResponse(w, req, resp)
		return
	}

	if err != nil {
		resp := responseError(0, err.Error())
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

		resp.Children = append(resp.Children, directoryChildEntry{
			ID:         albumFSID(album.ID),
			ParentID:   artistSubsonicID,
			MediaType:  "album",
			Title:      album.Name,
			Name:       album.Name,
			Album:      album.Name,
			AlbumID:    albumFSID(album.ID),
			Artist:     album.Artist,
			ArtistID:   artistSubsonicID,
			IsDir:      true,
			CoverArtID: albumConverArtID(album.ID),
			SongCount:  album.SongCount,
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
		ID:         albumSubsonicID,
		SongCount:  int64(len(tracks)),
		CoverArtID: albumConverArtID(albumID),
	}

	for _, track := range tracks {
		if resp.Name == "" {
			resp.Name = track.Album
			resp.ParentID = artistFSID(track.ArtistID)
			resp.Artist = track.Artist
		}

		if resp.Artist != track.Artist {
			resp.Artist = "Various Artists"
		}

		resp.Children = append(resp.Children, directoryChildEntry{
			ID:         trackFSID(track.ID),
			ParentID:   albumSubsonicID,
			MediaType:  "song",
			Title:      track.Title,
			Name:       track.Title,
			Artist:     track.Artist,
			ArtistID:   artistFSID(track.ArtistID),
			Album:      track.Album,
			AlbumID:    albumSubsonicID,
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
					Name:      artist.Name,
					Artist:    artist.Name,
					Title:     artist.Name,
					ID:        artistFSID(artist.ID),
					MediaType: "artist",
					ParentID:  combinedMusicFolderID,
					IsDir:     true,
				},
			)
		}

		page++
	}

	return resp, nil
}

type directoryResponse struct {
	baseResponse

	Directory directoryEntry `xml:"directory" json:"directory"`
}

type directoryEntry struct {
	ID         int64  `xml:"id,attr" json:"id,string"`
	ParentID   int64  `xml:"parent,attr" json:"parent,string"`
	Artist     string `xml:"-" json:"-"`
	Name       string `xml:"name,attr" json:"name"`
	AlbumCount int64  `xml:"albumCount,attr,omitempty" json:"albumCount,omitempty"`
	SongCount  int64  `xml:"songCount,attr,omitempty" json:"songCount,omitempty"`
	CoverArtID string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`

	Children []directoryChildEntry `xml:"child" json:"child"`
}

type directoryChildEntry struct {
	ID          int64  `xml:"id,attr" json:"id,string"`
	ParentID    int64  `xml:"parent,attr,omitempty" json:"parent,omitempty,string"`
	MediaType   string `xml:"mediaType,attr,omitempty" json:"mediaType"`
	Title       string `xml:"title,attr,omitempty" json:"title"`
	Name        string `xml:"name,attr,omitempty" json:"-"`
	Artist      string `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	ArtistID    int64  `xml:"artstId,attr,omitempty" json:"artistId,omitempty,string"`
	Album       string `xml:"album,attr,omitempty" json:"album"`
	AlbumID     int64  `xml:"-" json:"albumId,omitempty,string"`
	IsDir       bool   `xml:"isDir,attr" json:"isDir"`
	IsVideo     bool   `xml:"isVideo,attr,omitempty" json:"isVideo"`
	CoverArtID  string `xml:"coverArt,attr,omitempty" json:"coverArt"`
	Track       int64  `xml:"track,attr,omitempty" json:"track,omitempty"`       // position in album, I suppose
	Duration    int64  `xml:"duration,attr,omitempty" json:"duration,omitempty"` // in seconds
	Year        int16  `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre       string `xml:"genre,attr,omitempty" json:"gener,omitempty"`
	Size        int64  `xml:"size,attr,omitempty" json:"size,omitempty"` // in bytes
	ContentType string `xml:"contentType,attr,omitempty" json:"contentType,omitempty"`
	SongCount   int64  `xml:"songCount,attr,omitempty" json:"songCount,omitempty"`
	Suffix      string `xml:"suffix,attr,omitempty" json:"suffix,omitempty"`
	BitRate     string `xml:"bitRate,attr,omitempty" json:"bitRate,omitempty"`
	Path        string `xml:"path,attr,omitempty" json:"path,omitempty"` // on the file system I suppose
}
