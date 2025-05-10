package webserver

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/ironsmile/euterpe/src/library"
	"github.com/ironsmile/euterpe/src/playlists"
	"github.com/ironsmile/euterpe/src/webserver/webutils"
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if req.Method == http.MethodPost {
		plh.create(w, req)
		return
	}

	plh.list(w, req)
}

func (plh playlistsHandler) create(w http.ResponseWriter, req *http.Request) {
	listReq := playlistRequest{}
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&listReq); err != nil {
		webutils.JSONError(
			w,
			fmt.Sprintf("Cannot decode playlist JSON: %s", err),
			http.StatusBadRequest,
		)
		return
	}

	newID, err := plh.playlists.Create(req.Context(), listReq.Name, listReq.AddTracksByID)
	if err != nil {
		webutils.JSONError(
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
		webutils.JSONError(
			w,
			fmt.Sprintf("Playlist created but cannot write response JSON: %s", err),
			http.StatusInternalServerError,
		)
		return
	}
}

func (plh playlistsHandler) list(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		webutils.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	var page, perPage int64 = 1, 40
	pageStr := req.Form.Get("page")
	perPageStr := req.Form.Get("per-page")

	if pageStr != "" {
		var err error
		page, err = strconv.ParseInt(pageStr, 10, 64)

		if err != nil {
			webutils.JSONError(
				w,
				fmt.Sprintf(`Wrong "page" parameter: %s`, err),
				http.StatusBadRequest,
			)
			return
		}
	}

	if perPageStr != "" {
		var err error
		perPage, err = strconv.ParseInt(perPageStr, 10, 64)

		if err != nil {
			webutils.JSONError(
				w,
				fmt.Sprintf(`Wrong "per-page" parameter: %s`, err),
				http.StatusBadRequest,
			)
			return
		}
	}

	if page < 1 || perPage < 1 {
		webutils.JSONError(
			w,
			`"page" and "per-page" must be integers greater than one`,
			http.StatusBadRequest,
		)
		return
	}

	playlistsCount, err := plh.playlists.Count(req.Context())
	if err != nil {
		webutils.JSONError(
			w,
			fmt.Sprintf(`Cannot determine the playlists count: %s`, err),
			http.StatusInternalServerError,
		)
		return
	}

	prevPage, nextPage := getPlaylistsPrevNextPageURI(
		page,
		perPage,
		playlistsCount,
	)

	resp := playlistsResponse{
		PagesCount: int(math.Ceil(float64(playlistsCount) / float64(perPage))),
		Next:       nextPage,
		Previous:   prevPage,
		Playlists:  []playlist{},
	}
	playlists, err := plh.playlists.List(req.Context(), playlists.ListArgs{
		Offset: (page - 1) * perPage,
		Count:  perPage,
	})
	if err != nil {
		webutils.JSONError(
			w,
			fmt.Sprintf("Getting playlists failed: %s", err),
			http.StatusInternalServerError,
		)
		return
	}

	for _, pl := range playlists {
		resp.Playlists = append(resp.Playlists, toAPIplaylist(pl))
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		webutils.JSONError(
			w,
			fmt.Sprintf("Encoding playlists response failed: %s", err),
			http.StatusInternalServerError,
		)
	}
}

func getPlaylistsPrevNextPageURI(page, perPage, count int64) (string, string) {
	prevPage := ""
	if page-1 > 0 {
		prevPage = fmt.Sprintf(
			"/v1/playlists?page=%d&per-page=%d",
			page-1,
			perPage,
		)
	}

	nextPage := ""
	if page*perPage < count {
		nextPage = fmt.Sprintf(
			"/v1/playlists?page=%d&per-page=%d",
			page+1,
			perPage,
		)
	}

	return prevPage, nextPage
}

type playlistsResponse struct {
	Playlists  []playlist `json:"playlists"`
	Next       string     `json:"next,omitempty"`
	Previous   string     `json:"previous,omitempty"`
	PagesCount int        `json:"pages_count"`
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

// toAPIplaylist converts a playlists.Playlist to a playlist object suitable for
// JSON encoding as an API response from the Euterpe APIs.
func toAPIplaylist(pl playlists.Playlist) playlist {
	return playlist{
		ID:          pl.ID,
		Name:        pl.Name,
		Desc:        pl.Desc,
		TracksCount: pl.TracksCount,
		Duration:    pl.Duration.Milliseconds(),
		CreatedAt:   pl.CreatedAt.Unix(),
		UpdatedAt:   pl.UpdatedAt.Unix(),
		Tracks:      pl.Tracks,
	}
}

type playlistRequest struct {
	Name          string              `json:"name"`
	Desc          string              `json:"description"`
	AddTracksByID []int64             `json:"add_tracks_by_id"`
	RemoveIndeces []int64             `json:"remove_indeces"`
	MoveTracks    []playlistTrackMove `json:"move_indeces"`
}

// playlistTrackMove encodes a request to move a track from a particular index to
// another.
type playlistTrackMove struct {
	From uint32 `json:"from"`
	To   uint32 `json:"to"`
}
