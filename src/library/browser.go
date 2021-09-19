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

	// OrderDesc will order values in a descending mannger defined by their
	// comparison function.
	OrderDesc
)

// BrowseOrderBy represents the different properties by which values could be oredered. For every
// browse method the semantics for "name" and "id" could be different.
type BrowseOrderBy int

const (
	// OrderByUndefined means "order by any preoperty you wish"
	OrderByUndefined BrowseOrderBy = iota

	// OrderByID will order values by their respective ID field
	OrderByID

	// OrderByName will order vlues by their name
	OrderByName
)

// BrowseArgs defines all arguments one can pass to the browse methods to later its behaviour.
type BrowseArgs struct {
	Page    uint
	PerPage uint
	Order   BrowseOrder
	OrderBy BrowseOrderBy
}

//counterfeiter:generate . Browser

// Browser defines the methods for browsing a library.
type Browser interface {
	// BrowseArtists makes it possible to browse through the library artists page by page.
	// Returns a list of artists for particular page and the number of all artists in the
	// library.
	BrowseArtists(BrowseArgs) ([]Artist, int)

	// BrowseAlbums makes it possible to browse through the library albums page by page.
	// Returns a list of albums for particular page and the number of all albums in the
	// library.
	BrowseAlbums(BrowseArgs) ([]Album, int)
}
