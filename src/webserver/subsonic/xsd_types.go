package subsonic

import (
	"fmt"
	"mime"
	"net/url"
	"path/filepath"
	"time"

	"github.com/ironsmile/euterpe/src/library"
	"github.com/ironsmile/euterpe/src/playlists"
	"github.com/ironsmile/euterpe/src/radio"
)

type xsdIndexes struct {
	LastModified    int64      `xml:"lastModified,attr" json:"lastModified"`
	IgnoredArticles string     `xml:"ignoredArticles,attr" json:"ignoredArticles"`
	Children        []xsdIndex `xml:"index" json:"index"`
}

type xsdIndex struct {
	Name     string      `xml:"name,attr" json:"name"`
	Children []xsdArtist `xml:"artist" json:"artist"`
}

type xsdArtist struct {
	ID             int64      `xml:"id,attr" json:"id,string"`
	Name           string     `xml:"name,attr" json:"name"`
	ArtistImageURL string     `xml:"artistImageUrl,attr,omitempty" json:"artistImageUrl,omitempty"`
	UserRating     uint8      `xml:"userRating,attr,omitempty" json:"userRating,omitempty"`
	Starred        *time.Time `xml:"starred,attr,omitempty" json:"starred,omitempty"`
}

func toXSDArtist(artist library.Artist, artURL url.URL) xsdArtist {
	return xsdArtist{
		ID:             artistFSID(artist.ID),
		Name:           artist.Name,
		ArtistImageURL: artURL.String(),
		Starred:        toUnixTimeWithNull(artist.Favourite),
		UserRating:     artist.Rating,
	}
}

type xsdSearchResult2 struct {
	Artists []xsdArtist `xml:"artist" json:"artist"`
	Albums  []xsdChild  `xml:"album" json:"album"`
	Songs   []xsdChild  `xml:"song" json:"song"`
}

type xsdChild struct {
	ID            int64      `xml:"id,attr" json:"id,string"`
	ParentID      int64      `xml:"parent,attr,omitempty" json:"parent,omitempty,string"`
	DirectoryType string     `xml:"type,attr,omitempty" json:"type,omitempty"`
	Title         string     `xml:"title,attr,omitempty" json:"title"`
	Artist        string     `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	ArtistID      int64      `xml:"artistId,attr,omitempty" json:"artistId,omitempty,string"`
	Album         string     `xml:"album,attr,omitempty" json:"album"`
	AlbumID       int64      `xml:"albumId,attr,omitempty" json:"albumId,omitempty,string"`
	IsDir         bool       `xml:"isDir,attr" json:"isDir"`
	IsVideo       bool       `xml:"isVideo,attr,omitempty" json:"isVideo"`
	CoverArtID    string     `xml:"coverArt,attr,omitempty" json:"coverArt"`
	Track         int64      `xml:"track,attr,omitempty" json:"track,omitempty"`       // position in album, I suppose
	Duration      int64      `xml:"duration,attr,omitempty" json:"duration,omitempty"` // in seconds
	Year          int16      `xml:"year,attr" json:"year"`
	Genre         string     `xml:"genre,attr,omitempty" json:"gener,omitempty"`
	Size          int64      `xml:"size,attr,omitempty" json:"size,omitempty"` // in bytes
	ContentType   string     `xml:"contentType,attr,omitempty" json:"contentType,omitempty"`
	PlayCount     int64      `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	UserRating    uint8      `xml:"userRating,attr,omitempty" json:"userRating,omitempty"`
	Suffix        string     `xml:"suffix,attr,omitempty" json:"suffix,omitempty"`
	BitRate       int        `xml:"bitRate,attr,omitempty" json:"bitRate,omitempty"`
	Path          string     `xml:"path,attr,omitempty" json:"path,omitempty"` // on the file system I suppose
	Created       time.Time  `xml:"created,attr,omitempty" json:"created,omitempty"`
	Starred       *time.Time `xml:"starred,attr,omitempty" json:"starred,omitempty"`

	// Open Subsonic additions
	Name      string `xml:"-" json:"-"`
	SongCount int64  `xml:"-" json:"songCount,omitempty"`
	MediaType string `xml:"-" json:"mediaType"`
}

func trackToChild(track library.TrackInfo, defaultCreated time.Time) xsdChild {
	created := defaultCreated
	if track.CreatedAt != 0 {
		created = time.Unix(track.CreatedAt, 0)
	}

	return xsdChild{
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
		Starred:    toUnixTimeWithNull(track.Favourite),
		Year:       int16(track.Year),
		Size:       track.Size,
		BitRate:    int(track.Bitrate),

		// Here we take advantage of the knowledge that the track.Format is just
		// the file name extension.
		ContentType: mime.TypeByExtension(filepath.Ext("." + track.Format)),
	}
}

// albumToChild converts a library Album to a directory child entry.
// artistID is a in-db library ID.
//
// If artistID is empty then ParentID and ArtistID properties of the child
// will not be set.
func albumToChild(
	album library.Album,
	artistID int64,
	created time.Time,
) xsdChild {
	entry := xsdChild{
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
		Starred:       toUnixTimeWithNull(album.Favourite),
		UserRating:    album.Rating,
		PlayCount:     album.Plays,
		Year:          int16(album.Year),
	}

	if artistID != 0 {
		artistSubsonicID := artistFSID(artistID)
		entry.ParentID = artistSubsonicID
		entry.ArtistID = artistSubsonicID
	}

	return entry
}

func artistToChild(
	artist library.Artist,
	created time.Time,
) xsdChild {
	return xsdChild{
		ID:            albumFSID(artist.ID),
		MediaType:     "artist",
		DirectoryType: "music",
		Name:          artist.Name,
		Artist:        artist.Name,
		ArtistID:      artistFSID(artist.ID),
		IsDir:         true,
		CoverArtID:    artistCoverArtID(artist.ID),
		Created:       created,
		Starred:       toUnixTimeWithNull(artist.Favourite),
		UserRating:    artist.Rating,
	}
}

type xsdAlbumID3 struct {
	ID         int64      `xml:"id,attr" json:"id,string"`
	Name       string     `xml:"name,attr" json:"name"`
	Artist     string     `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	ArtistID   int64      `xml:"artistId,attr,omitempty" json:"artistId,omitempty,string"`
	CoverArtID string     `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	SongCount  int64      `xml:"songCount,attr" json:"songCount"`
	Duration   int64      `xml:"duration,attr" json:"duration"` // in seconds
	PlayCount  int64      `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	Created    time.Time  `xml:"created,attr" json:"created"`
	Starred    *time.Time `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	Year       int16      `xml:"year,attr" json:"year"`
	Genre      string     `xml:"genre,attr,omitempty" json:"gener,omitempty"`
}

func toAlbumID3Entry(child xsdChild) xsdAlbumID3 {
	return xsdAlbumID3{
		ID:         child.ID,
		Name:       child.Name,
		Artist:     child.Artist,
		ArtistID:   child.ArtistID,
		CoverArtID: child.CoverArtID,
		Duration:   child.Duration,
		Year:       child.Year,
		Genre:      child.Genre,
		SongCount:  child.SongCount,
		Created:    child.Created,
		Starred:    child.Starred,
		PlayCount:  child.PlayCount,
	}
}

func dbAlbumToAlbumID3Entry(album library.Album) xsdAlbumID3 {
	return xsdAlbumID3{
		ID:         albumFSID(album.ID),
		Name:       album.Name,
		Artist:     album.Artist,
		SongCount:  album.SongCount,
		CoverArtID: albumConverArtID(album.ID),
		Duration:   album.Duration / 1000,
		Starred:    toUnixTimeWithNull(album.Favourite),
		PlayCount:  album.Plays,
		Year:       int16(album.Year),
	}
}

type xsdAlbumList struct {
	Children []xsdChild `xml:"album" json:"album"`
}

type xsdAlbumList2 struct {
	Children []xsdAlbumID3 `xml:"album" json:"album"`
}

type xsdArtistID3 struct {
	ID             int64      `xml:"id,attr" json:"id,string"`
	Name           string     `xml:"name,attr" json:"name"`
	AlbumCount     int64      `xml:"albumCount,attr,omitempty" json:"albumCount,omitempty"`
	ArtistImageURL string     `xml:"artistImageUrl,attr,omitempty" json:"artistImageUrl,omitempty"`
	CoverArtID     string     `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Starred        *time.Time `xml:"starred,attr,omitempty" json:"starred,omitempty"`

	// Open Subsonic additions
	ParentID  int64 `xml:"-" json:"parent,string,omitempty"`
	SongCount int64 `xml:"songCount,attr,omitempty" json:"songCount,omitempty"`
}

func directoryToArtistID3(entry xsdDirectory) xsdArtistID3 {
	return xsdArtistID3{
		ID:             entry.ID,
		ParentID:       entry.ParentID,
		Name:           entry.Name,
		AlbumCount:     entry.AlbumCount,
		SongCount:      entry.SongCount,
		CoverArtID:     entry.CoverArtID,
		Starred:        entry.Starred,
		ArtistImageURL: entry.ArtistImageURL,
	}
}

type xsdArtistWithAlbumsID3 struct {
	xsdArtistID3

	Children []xsdAlbumID3 `xml:"album,omitempty" json:"album,omitempty"`
}

func dbArtistToArtistID3(artist library.Artist, artURL url.URL) xsdArtistID3 {
	return xsdArtistID3{
		ID:             artistFSID(artist.ID),
		Name:           artist.Name,
		AlbumCount:     artist.AlbumCount,
		CoverArtID:     artistCoverArtID(artist.ID),
		ArtistImageURL: artURL.String(),
		Starred:        toUnixTimeWithNull(artist.Favourite),
	}
}

type xsdArtistsID3 struct {
	IgnoredArticles string        `xml:"ignoredArticles,attr" json:"ignoredArticles"`
	Children        []xsdIndexID3 `xml:"index" json:"index"`
}

type xsdIndexID3 struct {
	Name     string         `xml:"name,attr" json:"name"`
	Children []xsdArtistID3 `xml:"artist" json:"artist"`
}

type xsdDirectory struct {
	ID         int64      `xml:"id,attr" json:"id,string"`
	ParentID   int64      `xml:"parent,attr,omitempty" json:"parent,omitempty,string"`
	Name       string     `xml:"name,attr" json:"name"`
	PlayCount  int64      `xml:"playCount,attr,omitempty" json:"playCount,omitempty"`
	Starred    *time.Time `xml:"starred,attr,omitempty" json:"starred,omitempty"`
	UserRating uint8      `xml:"userRating,attr,omitempty" json:"userRating,omitempty"`

	// Added in order to store data for other endpoint which reuse the
	// get_music_directory methods.
	ArtistImageURL string `xml:"-" json:"artistImageUrl,omitempty"`
	Duration       int64  `xml:"-" json:"duration,omitempty"` // in seconds
	AlbumCount     int64  `xml:"-" json:"albumCount,omitempty"`
	SongCount      int64  `xml:"-" json:"songCount,omitempty"`
	CoverArtID     string `xml:"-" json:"coverArt,omitempty"`
	Artist         string `xml:"-" json:"-"`

	Children []xsdChild `xml:"child" json:"child"`
}

type xsdAlbumWithSongsID3 struct {
	xsdAlbumID3

	Children []xsdChild `xml:"song" json:"song"`
}

type xsdArtistInfoBase struct {
	Notes          string `xml:"notes,omitempty" json:"notes,omitempty"`
	LastfmURL      string `xml:"lastFmUrl,omitempty" json:"lastFmUrl,omitempty"`
	SmallImageURL  string `xml:"smallImageUrl" json:"smallImageUrl"`
	MediumImageURL string `xml:"mediumImageUrl" json:"mediumImageUrl"`
	LargeImageURL  string `xml:"largeImageUrl" json:"largeImageUrl"`
}

type xsdAlbumInfo struct {
	Notes          string `xml:"notes,omitempty" json:"notes,omitempty"`
	LastfmURL      string `xml:"lastFmUrl,omitempty" json:"lastFmUrl,omitempty"`
	SmallImageURL  string `xml:"smallImageUrl" json:"smallImageUrl"`
	MediumImageURL string `xml:"mediumImageUrl" json:"mediumImageUrl"`
	LargeImageURL  string `xml:"largeImageUrl" json:"largeImageUrl"`
}

type xsdSearchResult3 struct {
	Artists []xsdArtistID3 `xml:"artist" json:"artist"`
	Albums  []xsdAlbumID3  `xml:"album" json:"album"`
	Songs   []xsdChild     `xml:"song" json:"song"`
}
type xsdSearchResult struct {
	Offset    int64      `xml:"offset,attr" json:"offset"`
	TotalHits int64      `xml:"totalHits,attr" json:"totalHits"`
	Matches   []xsdChild `xml:"match" json:"match"`
}

// toUnixTimeWithNull returns nil when `timestamp` is zero. Otherwise it returns
// returns a pointer to the time.Time which is the `timestamp` number of seconds size
// 01 Jan 1970 00:00:00 UTC.
func toUnixTimeWithNull(timestamp int64) *time.Time {
	if timestamp == 0 {
		return nil
	}

	ts := time.Unix(timestamp, 0)
	return &ts
}

type xsdStarred struct {
	Artists []xsdArtist `xml:"artist,omitempty" json:"artist,omitempty"`
	Albums  []xsdChild  `xml:"album,omitempty" json:"album,omitempty"`
	Songs   []xsdChild  `xml:"song,omitempty" json:"song,omitempty"`
}

type xsdStarred2 struct {
	Artists []xsdArtistID3 `xml:"artist,omitempty" json:"artist,omitempty"`
	Albums  []xsdAlbumID3  `xml:"album,omitempty" json:"album,omitempty"`
	Songs   []xsdChild     `xml:"song,omitempty" json:"song,omitempty"`
}

type xsdTopSongs struct {
	Songs []xsdChild `xml:"song,omitempty" json:"song,omitempty"`
}

type xsdInternetRadioStations struct {
	Stations []xsdInternetRadioStation `xml:"internetRadioStation,omitempty" json:"internetRadioStation,omitempty"`
}

type xsdInternetRadioStation struct {
	ID          int64  `xml:"id,attr" json:"id,string"`
	Name        string `xml:"name,attr" json:"name"`
	StreamURL   string `xml:"streamUrl,attr" json:"streamUrl"`
	HomePageURL string `xml:"homePageUrl,attr,omitempty" json:"homePageUrl,omitempty"`
}

func fromRadioStation(station radio.Station) xsdInternetRadioStation {
	s := xsdInternetRadioStation{
		ID:        station.ID,
		Name:      station.Name,
		StreamURL: station.StreamURL.String(),
	}
	if station.HomePage != nil {
		s.HomePageURL = station.HomePage.String()
	}
	return s
}

type xsdUser struct {
	Folders             []int64    `xml:"folder,omitempty" json:"folder,omitempty"`
	Username            string     `xml:"username,attr" json:"username"`
	Email               string     `xml:"email,attr,omitempty" json:"email,omitempty"`
	Scrobbling          bool       `xml:"scrobblingEnabled,attr" json:"scrobblingEnabled"`
	MaxBitrate          int        `xml:"maxBitRate,attr,omitempty" json:"maxBitRate,omitempty"`
	AdminRole           bool       `xml:"adminRole,attr" json:"adminRole"`
	SettingsRole        bool       `xml:"settingsRole,attr" json:"settingsRole"`
	DownloadRole        bool       `xml:"downloadRole,attr" json:"downloadRole"`
	UploadRole          bool       `xml:"uploadRole,attr" json:"uploadRole"`
	PlaylistRole        bool       `xml:"playlistRole,attr" json:"playlistRole"`
	CoverArtRole        bool       `xml:"coverArtRole,attr" json:"coverArtRole"`
	CommentRole         bool       `xml:"commentRole,attr" json:"commentRole"`
	PodcastRole         bool       `xml:"podcastRole,attr" json:"podcastRole"`
	StreamRole          bool       `xml:"streamRole,attr" json:"streamRole"`
	JukeboxRole         bool       `xml:"jukeboxRole,attr" json:"jukeboxRole"`
	ShareRole           bool       `xml:"shareRole,attr" json:"shareRole"`
	VideoConversionRole bool       `xml:"videoConversionRole,attr" json:"videoConversionRole"`
	AvatarLastChanged   *time.Time `xml:"avatarLastChanged,attr,omitempty" json:"avatarLastChanged,omitempty"`
}

type xsdPlaylist struct {
	ID         int64     `xml:"id,attr" json:"id,string"`
	Name       string    `xml:"name,attr" json:"name"`
	Comment    string    `xml:"comment,attr" json:"comment"`
	Owner      string    `xml:"owner,attr" json:"owner"`
	SongsCount int64     `xml:"songCount,attr" json:"songCount"`
	Duration   int64     `xml:"duration,attr" json:"duration"`
	Public     bool      `xml:"public,attr" json:"public"`
	Created    time.Time `xml:"created,attr" json:"created"`
	Changed    time.Time `xml:"changed,attr" json:"changed"`
	CoverArt   string    `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`

	AllowedUsers []string `xml:"allowedUser" json:"allowedUser"`
}

func toXsdPlaylist(playlist playlists.Playlist, owner string) xsdPlaylist {
	return xsdPlaylist{
		ID:           playlist.ID,
		Name:         playlist.Name,
		Comment:      playlist.Desc,
		Owner:        owner,
		SongsCount:   playlist.TracksCount,
		Public:       playlist.Public,
		Created:      playlist.CreatedAt,
		Changed:      playlist.UpdatedAt,
		Duration:     int64(playlist.Duration.Seconds()),
		AllowedUsers: []string{owner},
		CoverArt:     fmt.Sprintf("pl-%d", playlist.ID),
	}
}

type xsdPlaylistWithSongs struct {
	xsdPlaylist

	Entries []xsdChild `xml:"entry" json:"entry,omitempty"`
}

func toXsdPlaylistWithSongs(
	playlist playlists.Playlist,
	owner string,
	defaultLastModified time.Time,
) xsdPlaylistWithSongs {
	xsdPlst := xsdPlaylistWithSongs{
		xsdPlaylist: toXsdPlaylist(playlist, owner),
	}

	for _, track := range playlist.Tracks {
		xsdPlst.Entries = append(
			xsdPlst.Entries,
			trackToChild(track, defaultLastModified),
		)
	}

	return xsdPlst
}

type xsdPlaylists struct {
	Children []xsdPlaylist `xml:"playlist" json:"playlist"`
}

type xsdSongs struct {
	Songs []xsdChild `xml:"song" json:"song"`
}
