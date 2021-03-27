/*
Package art is responsible for getting cover art for albums or artist images over the
internet.

It finds album artwork by first querying the MusicBrainz web service for a releaseID using
the artist name and album name. Then using this ID it queries the Cover Art Archive
for the corresponding album front art.

Artist images are found using the MusicBrainz database and Discogs.

The following APIs are used to achieve this packages' objective:

 * MusicBrainz API: https://musicbrainz.org/doc/Development/XML_Web_Service/Version_2
 * Cover Art Archive: https://musicbrainz.org/doc/Cover_Art_Archive/
 * Discogs API: https://www.discogs.com/developers/
*/
package art
