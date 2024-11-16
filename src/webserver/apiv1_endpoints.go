package webserver

import "net/http"

// The following are URL Path endpoints for certain API calls.
const (
	APIv1EndpointAbout          = "/v1/about"
	APIv1EndpointFile           = "/v1/file/{fileID}"
	APIv1EndpointAlbumArtwork   = "/v1/album/{albumID}/artwork"
	APIv1EndpointDownloadAlbum  = "/v1/album/{albumID}"
	APIv1EndpointArtistImage    = "/v1/artist/{artistID}/image"
	APIv1EndpointBrowse         = "/v1/browse"
	APIv1EndpointSearchWithPath = "/v1/search/{searchQuery}"
	APIv1EndpointSearch         = "/v1/search/"
	APIv1EndpointLoginToken     = "/v1/login/token/"
	APIv1EndpointRegisterToken  = "/v1/register/token/"

	APIv1EndpointPlaylists = "/v1/playlists"
	APIv1EndpointPlaylist  = "/v1/playlist/{playlistID}"
)

// APIv1Methods defines on which HTTP methods APIv1 endpoints will respond to.
// It is an uri_path => list of HTTP methods map.
var APIv1Methods map[string][]string = map[string][]string{
	APIv1EndpointAbout:          {http.MethodGet},
	APIv1EndpointFile:           {http.MethodGet},
	APIv1EndpointDownloadAlbum:  {http.MethodGet},
	APIv1EndpointBrowse:         {http.MethodGet},
	APIv1EndpointSearchWithPath: {http.MethodGet},
	APIv1EndpointSearch:         {http.MethodGet},
	APIv1EndpointLoginToken:     {http.MethodPost},
	APIv1EndpointRegisterToken:  {http.MethodPost},
	APIv1EndpointArtistImage: {
		http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete,
	},
	APIv1EndpointAlbumArtwork: {
		http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete,
	},

	APIv1EndpointPlaylists: {http.MethodGet, http.MethodPost},
	APIv1EndpointPlaylist: {
		http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete,
	},
}
