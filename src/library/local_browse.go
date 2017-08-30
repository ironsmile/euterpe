package library

// BrowseArtists implements the Library interface for the local library by getting artists from
// the database ordered by their name.
func (lib *LocalLibrary) BrowseArtists(page, perPage int) ([]Artist, int) {
	return []Artist{}, 0
}

// BrowseAlbums implements the Library interface for the local library by getting albums from
// the database ordered by their name.
func (lib *LocalLibrary) BrowseAlbums(page, perPage int) ([]Album, int) {
	return []Album{}, 0
}
