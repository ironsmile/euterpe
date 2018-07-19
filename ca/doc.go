/*
Package ca (Cover Art) is responsible for getting cover art for albums over the
internet.

It does that by first querying the MusicBrainz web service for a releaseID using the
artist name and album name. Then using this ID it quries the Cover Art Archive
for the corresponding album front art.

The two APIs in questin are as follows:

 * MusicBrainz API: https://musicbrainz.org/doc/Development/XML_Web_Service/Version_2
 * Cover Art Archive: https://musicbrainz.org/doc/Cover_Art_Archive/
*/
package ca
