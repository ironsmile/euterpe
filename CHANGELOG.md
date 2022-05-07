# Change Log

## v1.5.2 - 2022-05-07

Another release focused on bug fixes and optimizations.

**What is New**

* The Euterpe Docker images are now based on Alpine which makes them quite smaller.

* The shuffle button in the web UI will no longer play around with the DOM playlist. Instead it will be a toggle which causes the next played track to be selected at random in the playlist. This _greatly_ improves the functionality with big playlists.

**Bug Fixes**

* Removed an artificial restriction which caused Euterpe to not be buildable on many operating systems (c6b143b).

* Start recognizing `mp4` files as supported.

* Added the `-dont-watch` option. Without watching the library paths for changes Euterpe will take up a lot less resources such as file descriptors and memory.

## v1.5.1 - 2021-10-03

This release is focused on stability and bugfixes. Most notably, it is the most tested release so far by a wide margain. The code test coverage between 1.5.0 has increased from ~40% to ~80%. Even with such a sharp increase in tests no interesting bugs were found in the code base.

**What is New**

* Support for Opus and WebM was added. Actually there was nothing from preventing previous versions from supporting them but a file extension check. For this version this check has been extended to include `.opus` and `.webm` files as well.

**Bug Fixes**

* Fixed a bug in the web UI filter when selecting an artist was not making the album filter include only albums for this artist.

* The HTTP Basic Authenticate challenge now properly names the software as "Euterpe" instead of "HTTPMS".

* The HTTP Basic Auth credentials checking is now not vulnerable to timing attacks.

* Fixed a bug where on some errors the login API endpoint was not returning JSON but plain text.

* No database entries will be created for album artwork for album IDs which are not already in the database.

* Downloading albums as ZIP will now return HTTP status code 400 instead of 404 when the request was malformed.

## v1.5.0 - 2021-08-03

**What is New**

* The project is finally renamed to "Euterpe"! If you have the old version, uninstall it manually as the new installer will not recognize it.

* The web UI playlist has been improved. Everything is neatly ordered now.

* The `-local-fs` flag was added which allows using assets from the file system instead of the bundled into the binary static files.

**Bug fixes**

* The artwork view in the web UI is finally a square so most common album artworks are now fully visible.

## v1.4.1 - 2021-05-25

**What is New**

* Added dark mode to web UI, your eyes will be thankful!
* The local file system is used as HTTP root when running in debug (-D) mode

## v1.4.0 - 2021-04-26

**What is New**

* Track duration is returned from the API and is shwon in the web UI
* Added support for artist images: `/v1/artist/{id}/image`
* The artist ID is included in the search results for every track
* There is support for album artwork or artist image thumbnails by appending `?size=small`
* Added the 'rescan' command. Running `httpms -rescan` will cause all of the tracks in the database to be scanned for changes in their metadata. Useful for when the id3 metadata scanning is improved in further versions.

**Bug fixes**

* Fixed: non-ASCII searches were case sensitive
* Fixed: some tracks were associated with the wrong album
* Fixed: media files with uppercase extensions were not included in the library
* Fixed: there might be duplicate tracks if the library in config.json was ending at "/"

## v1.3.1 - 2020-08-24

* Show album artwork in the web UI.
* Cleanup database on startup. Artists and albums without tracks are removed.

## v1.3.0 - 2019-01-06

* Connecting with devices via QR code is now possible.
* Explicitly show which format a media file is in the web UI.

## v1.2.2 - 2018-07-31

Two small bug fixes:

* A regression in config parsing where duration values (such as "20ms") could not be parsed.

* Fix a bug where the address scheme in generated QR barcode is always http.

## v1.2.1 - 2018-07-31

Added a page for generating token in a QR barcode suitable for scaning in the HTTPMS mobile app.

## v1.2.0 - 2018-07-29

Album artwork support is added into the server. Now artwork will be searched on disk or if configured - using the [Cover Art Archive](https://musicbrainz.org/doc/Cover_Art_Archive/). For this new API endpoints are created for working with the artwork.

All static files are bundled into the binary. This makes installation and uninstallation easier. On top of that because we now use `upx` the binary is much smaller than before. Apparently "more is less" contrary to what commander Pike says.

When configured with authentication HTTPMs will now have 3 new options for authentication with JWT tokens: HTTP cookie, `Authorization: Bearer` and via query parameter `token`. At the moment there is no public API for generating tokens but the interactive web login.

On the development front - dependencies are now vendored using `dep` instead of `govendor`. Also, from this version forward releases will be made from the `master` branch instead of `release/X` branches. The latter will be abandoned.

## v1.1.0 - 2017-09-01

A new API endpoint is included: `/browse/`. Using it one can browse through all artists or albums in paginated manner. See [here](README.md#browse) for its full documentation.

## v1.0.4 - 2017-03-08

### Changes

* Changed the library for scanning mp3 files meta information. This would hopefully improve the accuracy.

* The web UI now supports multiple albums with the same name. They would be individually listed in the album filter.

### New Stuff

* Now one can share search, artist and album selections, along with the currently playing track by just copying the URL.

* The UI is greatly improved on devices with small screens. This comes on the cost of exclusion of some features. On such devices one wouldn't be able to use suffle and repeat. Also, no direct download of albums or tracks. The main reason for these ommisions is that the original jPlayer theme was completely unaware of small devices. Future patches may bring the features back.

## v1.0.3 - 2017-02-11

### Bug fixes

Fixed a bug where an album by multiple artists would not be downloadable in bulk. This was because all albums were assumed to by one artist only. This means that there were actually a different album (with the same name) for every atist. Which in turn means that by downloading such an album, you would get only the songs for the particular artist.

## v1.0.2 - 2016-10-12

### Bug fixes

* Fixed a bug where track attributes were jumbled while scanning a library. The effect of this bug were tracks with wrong data for album, track number, title or artist.

## v1.0.1 - 2015-12-05

### What's new?

* This version have its dependencies vendored. All the code required for building it can be found in the release.

## v1.0.0 - 2014-10-19

The first tagged version.
