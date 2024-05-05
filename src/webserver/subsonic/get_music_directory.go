package subsonic

import (
	"fmt"
	"net/http"
	"strconv"

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
		entry xsdDirectory
	)

	if dirID == combinedMusicFolderID {
		entry, err = s.getRootDirectory(req)
	} else if isArtistID(dirID) {
		entry, err = s.getArtistDirectory(req, toArtistDBID(dirID))
	} else if isAlbumID(dirID) {
		entry, err = s.getAlbumDirectory(req, toAlbumDBID(dirID))
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
	req *http.Request,
	artistID int64,
) (xsdDirectory, error) {
	ctx := req.Context()

	artist, err := s.lib.GetArtist(ctx, artistID)
	if err != nil {
		return xsdDirectory{}, fmt.Errorf("getting artist: %w", err)
	}
	artistSubsonicID := artistFSID(artistID)
	albums := s.lib.GetArtistAlbums(artistID)

	artURL, _ := s.getAristImageURL(req, artistID)

	resp := xsdDirectory{
		ID:             artistSubsonicID,
		Name:           artist.Name,
		ParentID:       combinedMusicFolderID,
		AlbumCount:     int64(len(albums)),
		CoverArtID:     artistCoverArtID(artistID),
		ArtistImageURL: artURL.String(),
		Starred:        artist.Favourite,
		UserRating:     artist.Rating,
	}

	for _, album := range albums {
		resp.PlayCount += album.Plays

		resp.Children = append(resp.Children, albumToChild(
			album,
			artistID,
			s.lastModified,
		))
	}

	return resp, nil
}

func (s *subsonic) getAlbumDirectory(
	req *http.Request,
	albumID int64,
) (xsdDirectory, error) {
	album, err := s.lib.GetAlbum(req.Context(), albumID)
	if err != nil {
		return xsdDirectory{}, fmt.Errorf("getting album failed: %w", err)
	}

	albumSubsonicID := albumFSID(albumID)
	tracks := s.lib.GetAlbumFiles(albumID)

	resp := xsdDirectory{
		ID:         albumSubsonicID,
		Name:       album.Name,
		Artist:     album.Artist,
		SongCount:  int64(len(tracks)),
		CoverArtID: albumConverArtID(albumID),
		Starred:    album.Favourite,
		UserRating: album.Rating,
		Duration:   album.Duration,
		PlayCount:  album.Plays,
	}

	for _, track := range tracks {
		if resp.ParentID == 0 {
			resp.ParentID = artistFSID(track.ArtistID)
		}

		resp.Children = append(resp.Children, trackToChild(track, s.lastModified))
	}

	return resp, nil
}

func (s *subsonic) getRootDirectory(
	_ *http.Request,
) (xsdDirectory, error) {
	var (
		page uint = 0
		resp      = xsdDirectory{
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
				xsdChild{
					ID:            artistFSID(artist.ID),
					ParentID:      combinedMusicFolderID,
					Name:          artist.Name,
					Artist:        artist.Name,
					Title:         artist.Name,
					MediaType:     "artist",
					DirectoryType: "music",
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

	Directory xsdDirectory `xml:"directory,omitempty" json:"directory,omitempty"`
}
