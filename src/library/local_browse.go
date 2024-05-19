package library

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// BrowseArtists implements the Library interface for the local library by getting
// artists from the database. Returns an artists slice and the total count of all
// artists in the database.
func (lib *LocalLibrary) BrowseArtists(args BrowseArgs) ([]Artist, int) {
	offset := uint64(args.Page * args.PerPage)
	perPage := args.PerPage

	if args.Offset > 0 {
		offset = args.Offset
	}

	var (
		queryArgs []any
		where     []string
		whereStr  string
	)

	if args.ArtistID > 0 {
		where = append(where, "ar.id = @artistID")
		queryArgs = append(queryArgs, sql.Named("artistID", args.ArtistID))
	}

	order := "ASC"
	orderBy := "ar.name"

	if args.Order == OrderDesc {
		order = "DESC"
	}

	if args.OrderBy == OrderByID {
		orderBy = "ar.id"
	} else if args.OrderBy == OrderByRandom {
		orderBy = "RANDOM()"
		order = ""

		// When ordering by random the offset does not matter. Only the limit
		// does.
		offset = 0
	} else if args.OrderBy == OrderByFavourites {
		orderBy = "ars.favourite"
		where = append(where, "ars.favourite IS NOT NULL AND ars.favourite != 0")
	}

	artistsCount := lib.getTableSize("artists")
	var output []Artist

	if len(where) > 0 {
		whereStr = "WHERE " + strings.Join(where, " AND ")
	}

	queryArgs = append(
		queryArgs,
		sql.Named("offset", offset),
		sql.Named("perPage", perPage),
	)

	work := func(db *sql.DB) error {
		rows, err := db.Query(fmt.Sprintf(`
			SELECT
				ar.id,
				ar.name,
				(SELECT COUNT(DISTINCT(tr.album_id))
					FROM tracks tr
					WHERE tr.artist_id = ar.id) as albumsCount,
				ars.favourite,
				ars.user_rating
			FROM
				artists ar
				LEFT JOIN artists_stats as ars ON ars.artist_id = ar.id
			%s
			ORDER BY
				%s %s
			LIMIT
				@offset, @perPage
		`, whereStr, orderBy, order), queryArgs...)

		if err != nil {
			return err
		}

		defer rows.Close()
		for rows.Next() {
			var (
				res    Artist
				fav    sql.NullInt64
				rating sql.NullInt16
			)
			if err := rows.Scan(
				&res.ID, &res.Name, &res.AlbumCount, &fav, &rating,
			); err != nil {
				return fmt.Errorf("scanning db failed: %w", err)
			}
			if fav.Valid {
				res.Favourite = fav.Int64
			}
			if rating.Valid {
				res.Rating = uint8(rating.Int16)
			}

			output = append(output, res)
		}

		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error browse artist query: %s", err)
		return output, artistsCount
	}

	return output, artistsCount
}

// BrowseAlbums implements the Library interface for the local library by getting
// albums from the database.
func (lib *LocalLibrary) BrowseAlbums(args BrowseArgs) ([]Album, int) {
	offset := uint64(args.Page * args.PerPage)
	perPage := args.PerPage

	if args.Offset > 0 {
		offset = args.Offset
	}

	var (
		output      []Album
		albumsCount int

		queryArgs []any
		where     []string
		whereStr  string
	)

	if args.ArtistID > 0 {
		where = append(where, "tr.artist_id = @artistID")
		queryArgs = append(queryArgs, sql.Named("artistID", args.ArtistID))
	}

	order := "ASC"
	orderBy := "al.name"

	if args.Order == OrderDesc {
		order = "DESC"
	}

	switch args.OrderBy {
	case OrderByID:
		orderBy = "al.id"
	case OrderByRandom:
		orderBy = "RANDOM()"
		order = ""

		// When ordering by random the offset does not matter. Only the limit
		// does.
		offset = 0
	case OrderByFrequentlyPlayed:
		orderBy = "SUM(us.play_count)"
	case OrderByRecentlyPlayed:
		orderBy = "MAX(us.last_played)"
	case OrderByArtistName:
		orderBy = "artist_name"
	case OrderByFavourites:
		orderBy = "als.favourite"
		where = append(where, "als.favourite IS NOT NULL AND als.favourite != 0")
	}

	if len(where) > 0 {
		whereStr = "WHERE " + strings.Join(where, " AND ")
	}

	queryArgs = append(
		queryArgs,
		sql.Named("offset", offset),
		sql.Named("perPage", perPage),
	)

	work := func(db *sql.DB) error {
		smt, err := db.Prepare(`
			SELECT
				COUNT(DISTINCT tr.album_id) as cnt
			FROM
				tracks tr
				LEFT JOIN
					albums_stats als ON als.album_id = tr.album_id
			` + whereStr + `
		`)

		if err != nil {
			log.Printf("Query for getting albums count not prepared: %s\n", err)
		} else {
			err = smt.QueryRow().Scan(&albumsCount)

			if err != nil {
				log.Printf("Query for getting albums count not successful: %s\n", err)
			}
		}

		rows, err := db.Query(fmt.Sprintf(`
			SELECT
				al.id,
				al.name as album_name,
				CASE WHEN COUNT(DISTINCT tr.artist_id) = 1
					THEN ar.name
					ELSE "Various Artists"
					END AS artist_name,
				COUNT(tr.id) as songCount,
				SUM(tr.duration) as duration,
				als.favourite,
				als.user_rating
			FROM
				tracks tr
				LEFT JOIN
					albums al ON al.id = tr.album_id
				LEFT JOIN
					artists ar ON ar.id = tr.artist_id
				LEFT JOIN
					user_stats us ON us.track_id = tr.id
				LEFT JOIN
					albums_stats als ON als.album_id = tr.album_id
			%s
			GROUP BY
				tr.album_id
			ORDER BY
				%s %s
			LIMIT
				@offset, @perPage
		`, whereStr, orderBy, order), queryArgs...)

		if err != nil {
			return err
		}

		defer rows.Close()
		for rows.Next() {
			var (
				res    Album
				fav    sql.NullInt64
				rating sql.NullInt16
			)
			if err := rows.Scan(
				&res.ID, &res.Name, &res.Artist, &res.SongCount,
				&res.Duration, &fav, &rating,
			); err != nil {
				return fmt.Errorf("scanning db failed: %w", err)
			}
			if fav.Valid {
				res.Favourite = fav.Int64
			}
			if rating.Valid {
				res.Rating = uint8(rating.Int16)
			}

			output = append(output, res)
		}

		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error browse albums query: %s", err)
		return output, albumsCount
	}

	return output, albumsCount
}

// BrowseTracks implements the Library interface for the local library by getting
// tracks from the database.
func (lib *LocalLibrary) BrowseTracks(args BrowseArgs) ([]TrackInfo, int) {
	ctx, cancel := context.WithCancel(lib.ctx) //!TODO: use request context instead
	defer cancel()

	offset := uint64(args.Page * args.PerPage)
	perPage := args.PerPage

	if args.Offset > 0 {
		offset = args.Offset
	}

	var (
		output      []TrackInfo
		tracksCount int

		queryArgs []any
		where     []string
	)

	if args.ArtistID > 0 {
		where = append(where, "t.artist_id = @artistID")
		queryArgs = append(queryArgs, sql.Named("artistID", args.ArtistID))
	}

	order := "ASC"

	if args.Order == OrderDesc {
		order = "DESC"
	}

	orderBy := "t.id " + order

	switch args.OrderBy {
	case OrderByID:
		orderBy = "t.id " + order
	case OrderByName:
		orderBy = "t.name " + order
	case OrderByRandom:
		orderBy = "RANDOM()"

		// When ordering by random the offset does not matter. Only the limit
		// does.
		offset = 0
	case OrderByFrequentlyPlayed:
		orderBy = "us.play_count " + order
	case OrderByRecentlyPlayed:
		orderBy = "us.last_played " + order
	case OrderByArtistName:
		orderBy = "at.name " + order + ", t.album_id, t.number ASC"
	case OrderByFavourites:
		orderBy = "us.favourite " + order
		where = append(where, "us.favourite IS NOT NULL AND us.favourite != 0")
	}

	queryArgs = append(
		queryArgs,
		sql.Named("offset", offset),
		sql.Named("count", perPage),
	)

	whereSrt := ""
	if len(where) > 0 {
		whereSrt = "WHERE " + strings.Join(where, " AND ")
	}

	work := func(db *sql.DB) error {
		row := db.QueryRowContext(ctx, fmt.Sprintf(`
			SELECT
				COUNT(*) as cnt
			FROM
				tracks t
			%s
		`, whereSrt), queryArgs...)
		if err := row.Scan(&tracksCount); err != nil {
			log.Printf("Query for getting tracks count not successful: %s\n", err)
		}

		rows, err := queryTracks(ctx, db, where, orderBy, queryArgs)
		if err != nil {
			log.Printf("Query for browsing songs not successful: %s\n", err.Error())
			return nil
		}

		defer rows.Close()
		for rows.Next() {
			res, err := scanTrack(rows)
			if err != nil {
				log.Printf("Error scanning search result: %s\n", err)
				continue
			}
			output = append(output, res)
		}

		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error browse songs query: %s", err)
		return output, tracksCount
	}

	return output, tracksCount
}

func (lib *LocalLibrary) getTableSize(table string) int {
	var count int

	work := func(db *sql.DB) error {
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
		defer smt.Close()

		err = smt.QueryRow().Scan(&count)

		if err != nil {
			log.Printf("Query for getting %s count not successful: %s\n", table, err)
			return nil
		}

		return nil
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error getting table size query: %s", err)
		return count
	}

	return count
}
