# Change Log

## v1.0.4

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
