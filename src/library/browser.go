package library

// BrowseOrder represents different strategies which can be made with respect to the
// comparison function.
type BrowseOrder int

const (
	// OrderUndefined means "order any way you wish"
	OrderUndefined BrowseOrder = iota

	// OrderAsc will order values in an ascending manner defined by their
	// comparison function.
	OrderAsc

	// OrderDesc will order values in a descending manager defined by their
	// comparison function.
	OrderDesc
)

// BrowseOrderBy represents the different properties by which values could be ordered.
// For every browse method the semantics for "name" and "id" could be different.
type BrowseOrderBy int

const (
	// OrderByUndefined means "order by any property you wish".
	OrderByUndefined BrowseOrderBy = iota

	// OrderByID will order values by their respective ID field.
	OrderByID

	// OrderByName will order values by their name.
	OrderByName

	// OrderByRandom will cause the returned list to be in random order.
	OrderByRandom

	// OrderByRecentlyPlayed will order the list by how recent the particular
	// item has been played.
	OrderByRecentlyPlayed

	// OrderByFrequentlyPlayed will order the list by how many times the media
	// has been played.
	OrderByFrequentlyPlayed

	// OrderByFavourites will order the list by whether they have been added to
	// the favourites or not.
	OrderByFavourites

	// OrderByArtistName orders lists by the artist name of its items.
	OrderByArtistName

	// OrderByYear orders the lists by the year of recording.
	OrderByYear
)

// BrowseArgs defines all arguments one can pass to the browse methods to later
// their behaviour.
type BrowseArgs struct {
	// Page is used for skipping to a particular multiple of "PerPage" items in the
	// list to be returned.
	//
	// Page is ignored if Offset is used.
	//
	// Deprecated: Use Offset instead.
	Page uint

	// PerPage defines how many items to be returned from browse methods.
	PerPage uint

	// Order defines whether items will be in ascending or descending order based
	// on the OrderBy ordering type.
	Order BrowseOrder

	// OrderBy sets how the items to be returned will be ordered.
	OrderBy BrowseOrderBy

	// Offset is the number of items to skip before listing. If Offset is greater than
	// zero then the value of Page is ignored. Offset allows more precise targeting of
	// the next item.
	Offset uint64

	// ArtistID may be used for filtering the results so that only results which
	// belong this ArtistID are returned.
	ArtistID int64

	// FromYear is the inclusive lower limit for the year of recording of the returned
	// results.
	FromYear *int64

	// To year is the inclusive upper limit for the year of recording for the returned
	// results.
	ToYear *int64
}

//counterfeiter:generate . Browser

// Browser defines the methods for browsing a library.
type Browser interface {
	// BrowseArtists makes it possible to browse through the library artists page by page.
	// Returns a list of artists for particular page and the number of all artists who
	// match the browsing criteria.
	BrowseArtists(BrowseArgs) ([]Artist, int)

	// BrowseAlbums makes it possible to browse through the library albums page by page.
	// Returns a list of albums for particular page and the number of all albums which
	// match the browsing criteria.
	BrowseAlbums(BrowseArgs) ([]Album, int)

	// BrowseTracks makes possible browsing through the library songs. Returns a list
	// of songs (optionally sorted) and the number of songs which match the browsing
	// criteria.
	BrowseTracks(BrowseArgs) ([]TrackInfo, int)
}
