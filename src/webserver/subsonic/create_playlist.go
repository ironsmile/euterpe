package subsonic

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ironsmile/euterpe/src/playlists"
)

func (s *subsonic) createPlaylist(w http.ResponseWriter, req *http.Request) {
	if playlistID := req.Form.Get("playlistId"); playlistID != "" {
		s.updatePlaylistFromCreate(w, req)
	} else {
		s.createNewPlaylist(w, req)
	}
}

func (s *subsonic) createNewPlaylist(w http.ResponseWriter, req *http.Request) {
	name := req.Form.Get("name")
	if name == "" {
		resp := responseError(errCodeMissingParameter, "playlist name is required")
		encodeResponse(w, req, resp)
		return
	}

	trackIDs, err := querySongsToDBTrackIDs(req)
	if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	id, err := s.playlists.Create(req.Context(), name, trackIDs)
	if err != nil {
		resp := responseError(
			errCodeGeneric,
			fmt.Sprintf("failed to create playlist: %s", err),
		)
		encodeResponse(w, req, resp)
		return
	}

	s.respondCreatedPlaylist(w, req, id)
}

func (s *subsonic) updatePlaylistFromCreate(w http.ResponseWriter, req *http.Request) {
	playlistID, err := strconv.ParseInt(req.Form.Get("playlistId"), 10, 64)
	if err != nil {
		resp := responseError(errCodeNotFound, "playlist not found")
		encodeResponse(w, req, resp)
		return
	}

	trackIDs, err := querySongsToDBTrackIDs(req)
	if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	playlistUpdate := playlists.UpdateArgs{
		Name:            req.Form.Get("name"),
		RemoveAllTracks: true,
		AddTracks:       trackIDs,
	}

	if err := s.playlists.Update(req.Context(), playlistID, playlistUpdate); err != nil {
		resp := responseError(
			errCodeGeneric,
			fmt.Sprintf("failed to update playlist: %s", err),
		)
		encodeResponse(w, req, resp)
		return
	}

	s.respondCreatedPlaylist(w, req, playlistID)
}

// querySongsToDBTrackIDs converts a "songId" input query array into track IDs in the
// database.
func querySongsToDBTrackIDs(req *http.Request) ([]int64, error) {
	var trackIDs []int64
	for _, songIDstr := range req.Form["songId"] {
		songID, err := strconv.ParseInt(songIDstr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse songID: %w", err)
		}

		if !isTrackID(songID) {
			return nil, fmt.Errorf("song %d not found", songID)
		}

		trackIDs = append(trackIDs, toTrackDBID(songID))
	}

	return trackIDs, nil
}

func (s *subsonic) respondCreatedPlaylist(
	w http.ResponseWriter,
	req *http.Request,
	id int64,
) {
	playlist, err := s.playlists.Get(req.Context(), id)
	if err != nil {
		resp := responseError(
			errCodeGeneric,
			fmt.Sprintf("failed to get created playlist: %s", err),
		)
		encodeResponse(w, req, resp)
		return
	}

	resp := playlistWithSongsResponse{
		baseResponse: responseOk(),
		Playlist:     toXsdPlaylistWithSongs(playlist, s.auth.User, s.lastModified),
	}

	encodeResponse(w, req, resp)
}

type playlistWithSongsResponse struct {
	baseResponse

	Playlist xsdPlaylistWithSongs `xml:"playlist" json:"playlist"`
}
