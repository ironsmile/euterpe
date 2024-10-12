package subsonic

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ironsmile/euterpe/src/playlists"
)

func (s *subsonic) updatePlaylist(w http.ResponseWriter, req *http.Request) {
	idString := req.Form.Get("playlistId")
	if idString == "" {
		resp := responseError(errCodeMissingParameter, "playlist ID is required")
		encodeResponse(w, req, resp)
		return
	}

	id, err := strconv.ParseInt(idString, 10, 64)
	if err != nil {
		resp := responseError(errCodeNotFound, "playlist not found")
		encodeResponse(w, req, resp)
		return
	}

	updateArgs := playlists.UpdateArgs{
		Name: req.Form.Get("name"),
		Desc: req.Form.Get("comment"),
	}

	if public := req.Form.Get("public"); public == "true" {
		t := true
		updateArgs.Public = &t
	} else if public == "false" {
		t := false
		updateArgs.Public = &t
	}

	for _, removeIndexStr := range req.Form["songIndexToRemove"] {
		removeIndex, err := strconv.ParseInt(removeIndexStr, 10, 64)
		if err != nil {
			resp := responseError(errCodeGeneric,
				fmt.Sprintf("maformed index ID: %s", err),
			)
			encodeResponse(w, req, resp)
			return
		}

		updateArgs.RemoveTracks = append(updateArgs.RemoveTracks, removeIndex)
	}

	for _, songIDStr := range req.Form["songIdToAdd"] {
		songID, err := strconv.ParseInt(songIDStr, 10, 64)
		if err != nil {
			resp := responseError(errCodeGeneric,
				fmt.Sprintf("maformed song ID: %s", err),
			)
			encodeResponse(w, req, resp)
			return
		}

		if !isTrackID(songID) {
			resp := responseError(errCodeGeneric,
				fmt.Sprintf("there is no song with ID %d", songID),
			)
			encodeResponse(w, req, resp)
			return
		}

		updateArgs.AddTracks = append(updateArgs.AddTracks, toTrackDBID(songID))
	}

	err = s.playlists.Update(req.Context(), id, updateArgs)
	if err != nil && errors.Is(err, playlists.ErrNotFound) {
		resp := responseError(errCodeNotFound, "playlist not found")
		encodeResponse(w, req, resp)
		return
	} else if err != nil {
		resp := responseError(errCodeGeneric, err.Error())
		encodeResponse(w, req, resp)
		return
	}

	encodeResponse(w, req, responseOk())
}
