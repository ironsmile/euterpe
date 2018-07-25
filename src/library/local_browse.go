package library

import (
	"database/sql"
	"fmt"
	"log"
)

// BrowseArtists implements the Library interface for the local library by getting
// artists from the database ordered by their name. Returns an artists slice and the
// total count of all artists in the database.
func (lib *LocalLibrary) BrowseArtists(args BrowseArgs) ([]Artist, int) {
	page := args.Page
	perPage := args.PerPage

	order := "ASC"
	orderBy := "ar.name"

	if args.OrderBy == OrderByID {
		orderBy = "ar.id"
	}

	if args.Order == OrderDesc {
		order = "DESC"
	}

	artistsCount := lib.getTableSize("artists")
	var output []Artist

	var outErr error
	done := make(chan struct{})
	defer close(done)

	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()

		rows, err := db.Query(fmt.Sprintf(`
            SELECT
                ar.id,
                ar.name
            FROM
                artists ar
            ORDER BY
                %s %s
            LIMIT
                ?, ?
        `, orderBy, order), page*perPage, perPage)

		if err != nil {
			outErr = err
			return nil
		}

		defer rows.Close()
		for rows.Next() {
			var res Artist
			rows.Scan(&res.ID, &res.Name)
			output = append(output, res)
		}

		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		log.Printf("Error browse artist query: %s", err)
		return output, artistsCount
	}

	<-done
	if outErr != nil {
		log.Printf("Query for browsing artists not successful: %s\n", outErr)
	}

	return output, artistsCount
}

// BrowseAlbums implements the Library interface for the local library by getting
// albums from the database ordered by their name.
func (lib *LocalLibrary) BrowseAlbums(args BrowseArgs) ([]Album, int) {
	page := args.Page
	perPage := args.PerPage

	var (
		output      []Album
		albumsCount int
	)

	var outErr error
	done := make(chan struct{})
	defer close(done)

	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()

		smt, err := db.Prepare(`
            SELECT
                COUNT(DISTINCT tr.album_id) as cnt
            FROM
                tracks tr
        `)

		if err != nil {
			log.Printf("Query for getting albums count not prepared: %s\n", err)
		} else {
			err = smt.QueryRow().Scan(&albumsCount)

			if err != nil {
				log.Printf("Query for getting albums count not successful: %s\n", err)
			}
		}

		order := "ASC"
		orderBy := "al.name"

		if args.OrderBy == OrderByID {
			orderBy = "al.id"
		}

		if args.Order == OrderDesc {
			order = "DESC"
		}

		rows, err := db.Query(fmt.Sprintf(`
            SELECT
                al.id,
                al.name as album_name,
                CASE WHEN COUNT(DISTINCT tr.artist_id) = 1
                THEN ar.name
                ELSE "Various Artists"
                END AS arist_name
            FROM
                tracks tr
                LEFT JOIN
                    albums al ON al.id = tr.album_id
                LEFT JOIN
                    artists ar ON ar.id = tr.artist_id
            GROUP BY
                tr.album_id
            ORDER BY
                %s %s
            LIMIT
                ?, ?
        `, orderBy, order), page*perPage, perPage)

		if err != nil {
			outErr = err
			return nil
		}

		defer rows.Close()
		for rows.Next() {
			var res Album
			rows.Scan(&res.ID, &res.Name, &res.Artist)
			output = append(output, res)
		}

		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		log.Printf("Error browse artist query: %s", err)
		return output, albumsCount
	}

	<-done
	if outErr != nil {
		log.Printf("Query for browsing albums not successful: %s\n", outErr)
	}

	return output, albumsCount
}

func (lib *LocalLibrary) getTableSize(table string) int {

	var count int
	done := make(chan struct{})
	defer close(done)

	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()

		smt, err := db.Prepare(fmt.Sprintf(`
            SELECT
                COUNT(*) as cnt
            FROM
                %s
        `, table))

		if err != nil {
			log.Printf("Query for getting %s count not prepared: %s\n", table, err)
			return nil
		}

		err = smt.QueryRow().Scan(&count)

		if err != nil {
			log.Printf("Query for getting %s count not successful: %s\n", table, err)
			return nil
		}

		return nil
	}
	if err := lib.executeDBJob(work); err != nil {
		log.Printf("Error browse artist query: %s", err)
		return count
	}

	<-done
	return count
}
