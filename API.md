You can use Euterpe as a REST API and write your own player. Or maybe a plug-in for your favourite player which would use your Euterpe installation as a back-end.

### v1 Compatibility Promise

The API presented in this README is stable and will continue to be supported as long as version one of the service is around. And this should be very _long time_. I don't plan to make backward incompatible changes. Ever. It has survived in this form since 2013. So it should be good for at least double than this amount of time in the future.

This means that **clients written for Euterpe will continue to work**. I will never break them on purpose and if this happened it will be considered a bug to be fixed as soon as possible.

### Authentication

When your server is open you don't have to authenticate requests to the API. Installations protected by user name and password require you to authenticate requests when using the API. For this the following methods are supported:

* Bearer token in the `Authorization` HTTP header (as described in [RFC 6750](https://tools.ietf.org/html/rfc6750)):

```
Authorization: Bearer token
```

* Basic authentication ([RFC 2617](https://tools.ietf.org/html/rfc2617)) with your username and password:

```
Authorization: Basic base64(username:password)
```

Authentication tokens can be acquired using the `/v1/login/token/` endpoint described below. Using tokens is the preferred method since it does not expose your username and password in every request. Once acquired users must _register_ the tokens using the `/v1/register/token/` endpoint in order to "activate" them. Tokens which are not registered may or may not work. Tokens may have expiration date or they may not. Integration applications must provide a mechanism for token renewal.

### Endpoints

<!-- MarkdownTOC -->

* [About](#about)
* [Search](#search)
* [Browse](#browse)
* [Play a Song](#play-a-song)
* [Download an Album](#download-an-album)
* [Album Artwork](#album-artwork)
    - [Get Artwork](#get-artwork)
    - [Upload Artwork](#upload-artwork)
    - [Remove Artwork](#remove-artwork)
* [Artist Image](#artist-image)
    - [Get Artist Image](#get-artist-image)
    - [Upload Artist Image](#upload-artist-image)
    - [Remove Artist Image](#remove-artist-image)
* [Playlists](#playlists)
    - [List Playlists](#list-playlists)
    - [Create Playlist](#create-playlist)
    - [Get Playlist](#get-playlist)
    - [Replace Playlist](#replace-playlist)
    - [Update Playlist](#update-playlist)
    - [Delete Playlist](#delete-playlist)
* [Token Request](#token-request)
* [Register Token](#register-token)

<!-- /MarkdownTOC -->

### About

Query information about the server.

```sh
GET /v1/about
```

The returned response includes the server version. Example response:

```js
{
    "server_version":"v1.5.4"
}
```

This information could be used by clients to know what APIs are supported by the
server.

### Search

One can do a search query at the following endpoint

```sh
GET /v1/search/?q={query}
```

which would return an JSON array with tracks. Every object in the JSON represents a single track which matches the `query`. Example:

```js
[
   {
      "id" : 18, // Unique identifier of the track. Used for playing it.
      "album" : "Battlefield Vietnam", // Name of the album in which this track is found.
      "title" : "Somebody to Love", // Name of the song.
      "track" : 10, // Position of this track in the album.
      "artist" : "Jefferson Airplane", // Name of the artist or band who have performed the song.
      "artist_id": 33, // The ID of the artist who have performed the track.
      "album_id" : 2, // ID of the album in which this track belongs.
      "format": "mp3", // File format of this track. mp3, flac, wav, etc...
      "duration": 180000, // Track duration in milliseconds.
      "plays": 3, // Number of times this track has been played.
      "last_played": 1714834066, // Unix timestamp (seconds) when the track was last played.
      "rating": 5, // User rating in the [1-5] range.
      "favourite": 1714834066, // Unix timestamp (seconds) when the track was added to favourites.
      "bitrate": 1536000, // Bits per second of this song.
      "size": 3303014, // Size of the track file in bytes.
      "year": 2004 // Year when this track has been included in the album.
   },
   {
      "album" : "Battlefield Vietnam",
      "artist" : "Jefferson Airplane",
      "track" : 14,
      "format": "flac",
      "title" : "White Rabbit",
      "album_id" : 2,
      "id" : 22,
      "artist_id": 33,
      "duration": 308000
   }
]
```

The most important thing here is the track ID at the `id` key. It can be used for playing this track. The other interesting thing is `album_id`. Tracks can be grouped in albums using this value. And the last field of particular interest is `track`. It is the position of this track in the album.

Note that the track duration is in milliseconds.

_Optional properties_: Some properties of tracks are optional and may be omitted in the response when they are not set. They may not be set because no user has performed an action which sets them or the value may not be set in the track file's metadata. E.g. playing a song for the fist time will set its `plays` property to 1. The list of optional properties is: `plays`, `favourite`, `last_played`, `rating`, `bitrate`, `size`, `year`.

### Browse

A way to browse through the whole collection is via the browse API call. It allows you to get its albums or artists in an ordered and paginated manner.

```sh
GET /v1/browse/[?by=artist|album|song][&per-page={number}][&page={number}][&order-by=id|name|random|frequency|recency][&order=desc|asc]
```

The returned JSON contains the data for the current page, the number of all pages for the current browse method and URLs of the next or previous pages.

```js
{
  "pages_count": 12,
  "next": "/v1/browse/?page=4&per-page=10",
  "previous": "/v1/browse/?page=2&per-page=10",
  "data": [ /* different data types are returned, determined by the `by` parameter */ ]
}
```

For the moment there are three possible values for the `by` parameter. Consequently there are two types of `data` that can be returned: "artist", "song" and "album" (which is the **default**).

**by=artist**

would result in value such as

```js
{
  "artist": "Jefferson Airplane",
  "artist_id": 73,
  "album_count": 3 // Number of albums from this artist in the library.
  "favourite": 1614834066, // Unix timestamp in seconds. When it was added to favourites.
  "rating": 5 // User rating in [1-5] range.
}
```

The following fields are optional and may not be set:

* `favourite`
* `rating`

Missing fields mean that the artist hasn't been given rating or added to favourites.

**by=album**

would result in value such as

```js
{
  "album": "Battlefield Vietnam"
  "artist": "Various Artists",
  "album_id": 2,
  "duration": 1953000, // In milliseconds.
  "track_count": 12, // Number of tracks (songs) which this album has.
  "plays": 2312, // Number of times a song from the album has been played.
  "favourite": 1614834066, // Unix timestamp in seconds. When it was added to favourites.
  "last_played": 1714834066, // Unix timestamp in seconds.
  "rating": 5, // User rating in [1-5] range.
  "year": 2004 // Four digit year of when this album has been released.
}
```

The following fields are optional and may not be set:

* `favourite`
* `last_played`
* `rating`

Missing fields mean that the album hasn't been given rating, added to favourites or
no tracks from it have ever been played.

**by=song**

would in a list of objects which are the same as the result from the `/v1/search` endpoint.

**Additional parameters**

_per-page_: controls how many items would be present in the `data` field for every particular page. The **default is 10**.

_page_: the generated data would be for this page. The **default is 1**.

_order-by_: controls how the results would be ordered. **Defaults to `name` for albums and artists and `id` for tracks**. The meaning for its possible values is as follows:

* `id` means the ordering would be done by the song, album or artist ID, depending on the `by` argument.
* `name` orders values by their name.
* `random` means that the list will be randomly ordered.
* `frequency` will order by the number of times tracks have been played. For album this is the number of times tracks in this album has been played. Only applicable when `by` is `album` or `song`.
* `recency` will order tracks or albums by when was the last time the song or the album was played. Only applicable when `by` is `album` or `song`.
* `year` will order tracks or albums by the year of their release. Only applicable when `by` is `album` or `song`.

_order_: controls if the order would ascending (with value `asc`) or descending (with value `desc`). **Defaults to `asc`**.


### Play a Song

```
GET /v1/file/{trackID}
```

This endpoint would return you the media file as is. A song's `trackID` can be found with the search API call.

### Download an Album

```
GET /v1/album/{albumID}
```

This endpoint would return you an archive which contains the songs of the whole album.


### Album Artwork

Euterpe supports album artwork. Here are all the methods for managing it through the API.

#### Get Artwork

```
GET /v1/album/{albumID}/artwork
```

Returns a bitmap image with artwork for this album if one is available. Searching for artwork works like this: the album's directory would be scanned for any images (png/jpeg/gif/tiff files) and if anyone of them looks like an artwork, it would be shown. If this fails, you can configure Euterpe to search in the [MusicBrainz Cover Art Archive](https://musicbrainz.org/doc/Cover_Art_Archive/). By default no external calls are made, see the 'download_artwork' configuration property.

By default the full size image will be served. One could request a thumbnail by appending the `?size=small` query.

#### Upload Artwork

```
PUT /v1/album/{albumID}/artwork
```

Can be used to upload artwork directly on the Euterpe server. This artwork will be stored in the server database and will not create any files in the library paths. The image should be sent in the body of the request in binary format without any transformations. Only images up to 5MB are accepted. Example:

```sh
curl -i -X PUT \
  --data-binary @/path/to/file.jpg \
  http://127.0.0.1:9996/v1/album/18/artwork
```

#### Remove Artwork

```
DELETE /v1/album/{albumID}/artwork
```

Will remove the artwork from the server database. Note, this will not touch any files on the file system. Thus it is futile to call it for artwork which was found on disk.

### Artist Image

Euterpe could build a database with artists' images. Which it could then be used throughout the interfaces. Here are all the methods for managing it through the API.

#### Get Artist Image

```
GET /v1/artist/{artistID}/image
```

Returns a bitmap image representing an artist if one is available. Searching for artwork works like this: if artist image is found in the database then it will be used. In case there is not and Euterpe is configured to download images from internet and has a Discogs access token then it will use the MusicBrainz and Discogs APIs in order to retrieve an image. By default no internet requests are made.

By default the full size image will be served. One could request a thumbnail by appending the `?size=small` query.

#### Upload Artist Image

```
PUT /v1/artist/{artistID}/image
```

Can be used to upload artist image directly on the Euterpe server. It will be stored in the server database and will not create any files in the library paths. The image should be sent in the body of the request in binary format without any transformations. Only images up to 5MB are accepted. Example:

```sh
curl -i -X PUT \
  --data-binary @/path/to/file.jpg \
  http://127.0.0.1:9996/v1/artist/23/image
```

#### Remove Artist Image

```
DELETE /v1/artist/{artistID}/image
```

Will remove the artist image the server database. Note, this will not touch any files on the file system.

### Playlists

Euterpe supports creating and using playlists. Below you will find all supported operations with playlists.

#### List Playlists

```
GET /v1/playlists[?per-page={number}][&page={number}]
```

Returns paginated list of playlists. This list omits the track information and returns only the basic information about each playlist. Example response:

```js
{
  "playlists": [ // List with playlists.
    {
      "id": 1, // ID of the playlist which have to be used for operations with it.
      "name": "Quiet Evening", // Display name of the playlist.
      "description": "For when tired of heavy metal!", // Optional longer description.
      "tracks_count": 3, // Number of track in this playlist.
      "duration": 488000, // Duration of the playlist in milliseconds.
      "created_at": 1728838802, // Unix timestamp for when the playlist was created.
      "updated_at": 1728838923 // Unix timestamp for when the playlist was last updated.
    },
    {
      "id": 2,
      "name": "Summer Hits",
      "tracks_count": 4,
      "duration": 435000,
      "created_at": 1731773035,
      "updated_at": 1731773035
    }
  ],
  "previous": "/v1/playlists?page=1&per-page=2", // Next page with playlists if available
  "next": "/v1/playlists?page=3&per-page=2", // Previous page with playlists if available
  "pages_count": 2 // How many pages in total there are with playlists
}
```

**Optional parameters**

_per-page_: controls how many items would be present in the `playlists` field for every particular page. The **default is 40**.

_page_: the generated data would be for this page. The **default is 1**.

#### Create Playlist

```
POST /v1/playlists
{
    "name": "Quiet Evening",
    "description": "Something to say about the playlist.",
    "add_tracks_by_id": [14, 18, 255, 99]
}
```

Creating a playlist is done with a `POST` request with a JSON body. The body is an object
with the following properties:

* `name` (_string_) - A short name of the playlist. Used for displaying it in lists.
* `description` (_string_) - Longer description of the playlist visible when showing this particular playlist.
* `add_tracks_by_id` (_list_ with integers) - An ordered list with track IDs which will be added in the playlist. IDs may repeat.

This API method returns the ID of the newly created playlist:

```js
{
    "created_playlsit_id": 2
}
```

#### Get Playlist

```
GET /v1/playlist/{playlistID}
```

It will return information about a particular playlist with ID `playlistID`. It includes the tracks which are part of this playlist but is otherwise the same as an item in the List API endpoint.

```js
{
  "id": 1,
  "name": "New Playlist",
  "description": "Some description text!",
  "tracks_count": 2,
  "duration": 311000,
  "created_at": 1728838802,
  "updated_at": 1728838923,
  "tracks": [ // A list of tracks which are included in the playlist.
    {
      "id": 93,
      "artist_id": 25,
      "artist": "Ketsa",
      "album_id": 10,
      "album": "Summer With Sound",
      "title": "Essence",
      "track": 7,
      "format": "mp3",
      "duration": 200000,
      "bitrate": 131072,
      "size": 3245946
    },
    {
      "id": 136,
      "artist_id": 27,
      "artist": "Daft Punk",
      "album_id": 13,
      "album": "Discovery",
      "title": "Nightvision",
      "track": 6,
      "format": "mp3",
      "duration": 111000,
      "plays": 1,
      "last_played": 1715795866,
      "year": 2001,
      "bitrate": 131072,
      "size": 1783108
    }
  ]
}
```

#### Replace Playlist

```
PUT /v1/playlist/{playlistID}
{
    "name": "Noisy Evening",
    "description": "It is not that interesting, to be honest",
    "add_tracks_by_id": [101, 25, 33]
}
```

Completely replace a particular playlist with a new one. The request body is the same as the [Create Playlist](#create-playlist) endpoint.

Note that all tracks of the old playlists will be removed before the tracks mentioned in the `add_tracks_by_id` are added to the playlist.

#### Update Playlist

```
PATCH /v1/playlist/{playlistID}
{
    "name": "New Name",
    "description": "A completely new description",
    "add_tracks_by_id": [101, 25, 33],
    "remove_indeces": [0, 5],
    "move_indeces": [
      {"from": 0, "to": 1}, {"from": 25, "to": 12}
    ]
}
```

All properties of the request for changing a playlist are **optional**. Not including them will preserve their original values. The properties are:

* `name` (_string_) - A short name of the playlist. Used for displaying it in lists.
* `description` (_string_) - Longer description of the playlist visible when showing this particular playlist.
* `add_tracks_by_id` (_list_ with integers) - An ordered list with track IDs which will be added in the playlist. IDs may repeat.
* `remove_indeces` (_list_ with integers) - A list with integers where each one is an index in the playlist. Tracks on these indexes will be removed from the playlist.
* `move_indeces` (_list_ with "move" objects) - A list of "move operations". Every move operation is a JSON object which contains "from" and "to" properties which values are indexes in the playlist.

Operations with tracks in the change request are performed in a strict order which is:

1. Removing tracks in `remove_indeces`
2. Adding tracks in `add_tracks_by_id`
3. Moving tracks around mentioned in the `move_indeces`

Note that moving tracks is done in the order given in `move_indeces` and each next move works on the new playlist state which came as a result of a previous moves.

#### Delete Playlist

```
DELETE /v1/playlist/{playlistID}
```

This will remove the playlist with ID `playlistID`.

### Token Request

```
POST /v1/login/token/
{
  "username": "your-username",
  "password": "your-password"
}
```

You have to send your username and password as a JSON in the body of the request as described above. Provided they are correct you will receive the following response:

```js
{
  "token": "new-authentication-token"
}
```

Before you can use this token for accessing the API you will have to register it with on "Register Token" endpoint.

### Register Token

```
POST /v1/register/token/
```

This endpoint registers the newly generated tokens with Euterpe. Only registered tokens will work. Requests at this endpoint must authenticate themselves using a previously generated token.