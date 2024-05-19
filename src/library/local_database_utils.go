package library

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// queryTracks executes a database query for tracks and returns the result. The query
// is written with the appropriate JOIN and following aliases are available:
//
// * `t` - the tracks table
// * `us` - the user_stats table
// * `at` - the artists table
// * `al` - the albums table
//
// The function arguments are:
//
//   - where - list of clauses which will be joined with "AND" statement. Example:
//     []string{"baba = 1", "dqdo = 1"} will become "WHERE baba=1 AND dqdo = 2". Can
//     be left blank.
//
//   - orderBy - the order by statement as a whole. Can be left blank.
//
//   - queryArgs - arguments to be used in the db.QueryContext call. If the two
//     named arguments "offset" and "count", created with sql.Named(...) are set
//     then the they will be used with LIMIT for the query.
func queryTracks(
	ctx context.Context,
	db *sql.DB,
	where []string,
	orderBy string,
	queryArgs []any,
) (*sql.Rows, error) {
	whereStr := ""
	if len(where) > 0 {
		whereStr = "WHERE " + strings.Join(where, " AND ")
	}

	orderByStr := ""
	if orderBy != "" {
		orderByStr = "ORDER BY " + orderBy
	}

	limitStr := ""
	for _, queryArg := range queryArgs {
		named, ok := queryArg.(sql.NamedArg)
		if !ok {
			continue
		}
		if named.Name == "offset" || named.Name == "count" {
			limitStr = "LIMIT @offset, @count"
			break
		}
	}

	return db.QueryContext(
		ctx,
		fmt.Sprintf(`
			%s
			%s
			%s
			%s
		`, dbTracksQuery, whereStr, orderByStr, limitStr,
		),
		queryArgs...,
	)
}

// scanTrack scans a database row returned by `queryTracks` into a TrackInfo.
func scanTrack(rows scanner) (TrackInfo, error) {
	var (
		res TrackInfo

		// nullable values from the result
		dur        sql.NullInt64
		fav        sql.NullInt64
		rating     sql.NullInt16
		lastPlayed sql.NullInt64
		playCount  sql.NullInt64
		year       sql.NullInt32
		bitrate    sql.NullInt64
		size       sql.NullInt64
		createdAt  sql.NullInt64
	)

	err := rows.Scan(&res.ID, &res.Title, &res.Album, &res.Artist,
		&res.ArtistID, &res.TrackNumber, &res.AlbumID, &res.Format,
		&dur, &year, &bitrate, &size, &createdAt, &fav, &rating, &lastPlayed, &playCount,
	)
	if err != nil {
		return res, err
	}

	res.Format = mediaFormatFromFileName(res.Format)
	if fav.Valid && fav.Int64 > 0 {
		res.Favourite = fav.Int64
	}
	if rating.Valid && rating.Int16 >= 1 && rating.Int16 <= 5 {
		res.Rating = uint8(rating.Int16)
	}
	if lastPlayed.Valid {
		res.LastPlayed = lastPlayed.Int64
	}
	if playCount.Valid {
		res.Plays = playCount.Int64
	}
	if dur.Valid {
		res.Duration = dur.Int64
	}
	if year.Valid {
		res.Year = year.Int32
	}
	if bitrate.Valid && bitrate.Int64 > 0 {
		res.Bitrate = uint64(bitrate.Int64)
	}
	if size.Valid && size.Int64 > 0 {
		res.Size = size.Int64
	}
	if createdAt.Valid {
		res.CreatedAt = createdAt.Int64
	}

	return res, nil
}

type scanner interface {
	Scan(dest ...any) error
}

var (
	// dbTracksQuery is the query used in `queryTracks` and other places
	// for selecting the information for tracks.
	dbTracksQuery = `
	SELECT
		t.id as track_id,
		t.name as track,
		al.name as album,
		at.name as artist,
		at.id as artist_id,
		t.number as track_number,
		t.album_id as album_id,
		t.fs_path as fs_path,
		t.duration as duration,
		t.year as year,
		t.bitrate as bitrate,
		t.size as file_size,
		t.created_at as file_created_at,
		us.favourite as fav,
		us.user_rating as rating,
		us.last_played as last_played,
		us.play_count as play_count
	FROM
		tracks as t
			LEFT JOIN albums as al ON al.id = t.album_id
			LEFT JOIN artists as at ON at.id = t.artist_id
			LEFT JOIN user_stats as us ON us.track_id = t.id
	`
)
