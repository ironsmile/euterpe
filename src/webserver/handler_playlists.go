package webserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ironsmile/euterpe/src/library"
	"github.com/ironsmile/euterpe/src/playlists"
)

// playlistsHandler will list playlists (GET) and create a new one (POST).
type playlistsHandler struct {
	playlists playlists.Playlister
}

// NewPlaylistsHandler returns an http.Handler which supports listing all playlists
// with a GET request and creating a new playlist with a POST request.
func NewPlaylistsHandler(playlister playlists.Playlister) http.Handler {
	return &playlistsHandler{
		playlists: playlister,
	}
}

// ServeHTTP is required by the http.Handler's interface
func (plh playlistsHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodPost {
		plh.create(w, req)
		return
	}

	plh.listAll(w, req)
}

func (plh playlistsHandler) create(w http.ResponseWriter, req *http.Request) {
	listReq := playlistRequest{}
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&listReq); err != nil {
		http.Error(
			w,
			fmt.Sprintf("Cannot decode playlist JSON: %s", err),
			http.StatusBadRequest,
		)
		return
	}

	newID, err := plh.playlists.Create(req.Context(), listReq.Name, listReq.AddTrackByID)
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Failed to create playlist: %s", err),
			http.StatusInternalServerError,
		)
		return
	}

	resp := createPlaylistResponse{
		CreatedPlaylistID: newID,
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		http.Error(
			w,
			fmt.Sprintf("Playlist created but cannot write response JSON: %s", err),
			http.StatusInternalServerError,
		)
		return
	}
}

func (plh playlistsHandler) listAll(w http.ResponseWriter, req *http.Request) {
	resp := playlistsResponse{}
	playlists, err := plh.playlists.GetAll(req.Context())
	if err != nil {
		http.Error(
			w,
			fmt.Sprintf("Getting playlists failed: %s", err),
			http.StatusInternalServerError,
		)
		return
	}

	for _, pl := range playlists {
		resp.Playlists = append(resp.Playlists, playlist{
			ID:          pl.ID,
			Name:        pl.Name,
			Desc:        pl.Desc,
			TracksCount: pl.TracksCount,
			Duration:    pl.Duration.Milliseconds(),
			CreatedAt:   pl.CreatedAt.Unix(),
			UpdatedAt:   pl.UpdatedAt.Unix(),
		})
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		http.Error(
			w,
			fmt.Sprintf("Encoding playlists response failed: %s", err),
			http.StatusInternalServerError,
		)
	}
}

type playlistsResponse struct {
	Playlists []playlist `json:"playlists"`
}

type createPlaylistResponse struct {
	CreatedPlaylistID int64 `json:"created_playlsit_id"`
}

type playlist struct {
	ID          int64               `json:"id"`
	Name        string              `json:"name"`
	Desc        string              `json:"description,omitempty"`
	TracksCount int64               `json:"tracks_count"`
	Duration    int64               `json:"duration"`   // Playlist duration in millisecs.
	CreatedAt   int64               `json:"created_at"` // Unix timestamp in seconds.
	UpdatedAt   int64               `json:"updated_at"` // Unix timestamp in seconds.
	Tracks      []library.TrackInfo `json:"tracks,omitempty"`
}

type playlistRequest struct {
	Name           string  `json:"name"`
	Desc           string  `json:"description"`
	AddTrackByID   []int64 `json:"add_tracks_by_id"`
	RemoveIndecies []int64 `json:"remove_indecies"`
}
