package webserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/ironsmile/euterpe/src/playlists"
)

// playlistHandler will handle the REST methods for a single playlist.
//
// The playlist operations are as follows:
//
// * Getting playlist info (GET)
// * Removing the playlist (DELETE)
// * Completely replacing the tracks in the playlist (PUT)
// * Change playlist information and/or reordering tracks (PATCH)
type playlistHandler struct {
	playlists playlists.Playlister
}

// NewSinglePlaylistHandler returns an HTTP handler for interacting with a single
// playlist identified by its ID.
func NewSinglePlaylistHandler(playlister playlists.Playlister) http.Handler {
	return &playlistHandler{
		playlists: playlister,
	}
}

// ServeHTTP is required by the http.Handler's interface
func (h *playlistHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	playlistID, err := strconv.ParseInt(vars["playlistID"], 10, 64)
	if err != nil {
		http.NotFound(w, req)
		return
	}

	if req.Method == http.MethodPut {
		h.replacePlaylist(w, req, playlistID)
		return
	} else if req.Method == http.MethodPatch {
		h.changePlaylist(w, req, playlistID)
		return
	} else if req.Method == http.MethodDelete {
		h.deletePlaylist(w, req, playlistID)
		return
	}

	h.getPlaylist(w, req, playlistID)
}

func (h *playlistHandler) replacePlaylist(
	w http.ResponseWriter,
	req *http.Request,
	playlistID int64,
) {
	var params playlistRequest
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&params); err != nil {
		http.Error(
			w,
			fmt.Sprintf("cannot parse request body: %s", err),
			http.StatusBadRequest,
		)
		return
	}

	updateReq := playlists.UpdateArgs{
		Name:            params.Name,
		Desc:            params.Desc,
		AddTracks:       params.AddTracksByID,
		RemoveAllTracks: true,
	}

	err := h.playlists.Update(req.Context(), playlistID, updateReq)
	if errors.Is(err, playlists.ErrNotFound) {
		http.NotFound(w, req)
		return
	} else if err != nil {
		http.Error(
			w,
			fmt.Sprintf("error replacing the playlist: %s", err),
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *playlistHandler) changePlaylist(
	w http.ResponseWriter,
	req *http.Request,
	playlistID int64,
) {
	var params playlistRequest
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&params); err != nil {
		http.Error(
			w,
			fmt.Sprintf("cannot parse request body: %s", err),
			http.StatusBadRequest,
		)
		return
	}

	updateReq := playlists.UpdateArgs{
		Name:         params.Name,
		Desc:         params.Desc,
		AddTracks:    params.AddTracksByID,
		RemoveTracks: params.RemoveIndeces,
	}

	for _, moveReq := range params.MoveTracks {
		updateReq.MoveTracks = append(updateReq.MoveTracks, playlists.MoveArgs{
			FromIndex: moveReq.From,
			ToIndex:   moveReq.To,
		})
	}

	err := h.playlists.Update(req.Context(), playlistID, updateReq)
	if errors.Is(err, playlists.ErrNotFound) {
		http.NotFound(w, req)
		return
	} else if err != nil {
		http.Error(
			w,
			fmt.Sprintf("error updating the playlist: %s", err),
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *playlistHandler) deletePlaylist(
	w http.ResponseWriter,
	req *http.Request,
	playlistID int64,
) {
	err := h.playlists.Delete(req.Context(), playlistID)
	if errors.Is(err, playlists.ErrNotFound) {
		http.NotFound(w, req)
		return
	} else if err != nil {
		http.Error(
			w,
			fmt.Sprintf("error deleting a playlist: %s", err),
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *playlistHandler) getPlaylist(
	w http.ResponseWriter,
	req *http.Request,
	playlistID int64,
) {
	pl, err := h.playlists.Get(req.Context(), playlistID)
	if errors.Is(err, playlists.ErrNotFound) {
		http.NotFound(w, req)
		return
	} else if err != nil {
		http.Error(
			w,
			fmt.Sprintf("error getting a playlist: %s", err),
			http.StatusInternalServerError,
		)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(toAPIplaylist(pl)); err != nil {
		http.Error(
			w,
			fmt.Sprintf("Encoding playlist response failed: %s", err),
			http.StatusInternalServerError,
		)
	}
}
