package library

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// RecordFavourite stores as favourites the tracks, albums and artists in `fav`.
func (lib *LocalLibrary) RecordFavourite(ctx context.Context, favs Favourites) error {
	work := func(db *sql.DB) (workErr error) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("cannot begin transaction: %w", err)
		}
		defer func() {
			if workErr != nil {
				_ = tx.Rollback()
				return
			}

			if err := tx.Commit(); err != nil {
				workErr = fmt.Errorf("failed to commit transaction: %w", err)
			}
		}()

		// First, insert favourites for user tracks.

		var queryArgs []any

		query := `
			INSERT INTO user_stats (track_id, favourite)
			VALUES
		`

		for _, trackID := range favs.TrackIDs {
			queryArgs = append(queryArgs, trackID)
		}

		query += strings.Repeat(`(?, strftime('%s')),`, len(queryArgs))
		query = strings.TrimSuffix(query, ",")

		query += `
			ON CONFLICT(track_id) DO UPDATE SET
				favourite = strftime('%s');
		`

		if len(queryArgs) > 0 {
			_, err := tx.ExecContext(ctx, query, queryArgs...)
			if err != nil {
				return fmt.Errorf("track SQL query error: %w", err)
			}
		}

		// Second, insert favourites for albums.

		queryArgs = []any{}

		query = `
			INSERT INTO albums_stats (album_id, favourite)
			VALUES
		`

		for _, albumID := range favs.AlbumIDs {
			queryArgs = append(queryArgs, albumID)
		}

		query += strings.Repeat(`(?, strftime('%s')),`, len(queryArgs))
		query = strings.TrimSuffix(query, ",")

		query += `
			ON CONFLICT(album_id) DO UPDATE SET
				favourite = strftime('%s');
		`

		if len(queryArgs) > 0 {
			_, err := tx.ExecContext(ctx, query, queryArgs...)
			if err != nil {
				return fmt.Errorf("albums SQL query error: %w", err)
			}
		}

		// And lastly, insert favourites for artists.

		queryArgs = []any{}

		query = `
			INSERT INTO artists_stats (artist_id, favourite)
			VALUES
		`

		for _, artistID := range favs.ArtistIDs {
			queryArgs = append(queryArgs, artistID)
		}

		query += strings.Repeat(`(?, strftime('%s')),`, len(queryArgs))
		query = strings.TrimSuffix(query, ",")

		query += `
			ON CONFLICT(artist_id) DO UPDATE SET
				favourite = strftime('%s');
		`

		if len(queryArgs) > 0 {
			_, err := tx.ExecContext(ctx, query, queryArgs...)
			if err != nil {
				return fmt.Errorf("artists SQL query error: %w", err)
			}
		}

		return nil
	}

	if err := lib.ExecuteDBJobAndWait(work); err != nil {
		log.Printf("Error executing record favourites: %s", err)
		return fmt.Errorf("storing into the database failed: %w", err)
	}

	return nil
}

// RemoveFavourite removes tracks, albums and artists in `fav` from the favourites.
func (lib *LocalLibrary) RemoveFavourite(ctx context.Context, favs Favourites) error {
	work := func(db *sql.DB) (workErr error) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("cannot begin transaction: %w", err)
		}
		defer func() {
			if workErr != nil {
				_ = tx.Rollback()
				return
			}

			if err := tx.Commit(); err != nil {
				workErr = fmt.Errorf("failed to commit transaction: %w", err)
			}
		}()

		// Update the tracks.

		var queryArgs []any
		query := `
			UPDATE user_stats
			SET
				favourite = NULL
			WHERE
				track_id IN (%s)
		`
		for _, trackID := range favs.TrackIDs {
			queryArgs = append(queryArgs, trackID)
		}

		if len(queryArgs) > 0 {
			placeHolders := strings.TrimSuffix(strings.Repeat("?,", len(queryArgs)), ",")
			query = fmt.Sprintf(query, placeHolders)
			_, err := tx.ExecContext(ctx, query, queryArgs...)
			if err != nil {
				return fmt.Errorf("query for updating tracks failed: %w", err)
			}
		}

		// Then update the albums.

		queryArgs = []any{}
		query = `
			UPDATE albums_stats
		SET
			favourite = NULL
		WHERE
			album_id IN (%s)
		`
		for _, albumID := range favs.AlbumIDs {
			queryArgs = append(queryArgs, albumID)
		}

		if len(queryArgs) > 0 {
			placeHolders := strings.TrimSuffix(strings.Repeat("?,", len(queryArgs)), ",")
			query = fmt.Sprintf(query, placeHolders)
			_, err := tx.ExecContext(ctx, query, queryArgs...)
			if err != nil {
				return fmt.Errorf("query for updating albums failed: %w", err)
			}
		}

		// And at the end update artists.

		queryArgs = []any{}
		query = `
			UPDATE artists_stats
		SET
			favourite = NULL
		WHERE
			artist_id IN (%s)
		`
		for _, artistID := range favs.ArtistIDs {
			queryArgs = append(queryArgs, artistID)
		}

		if len(queryArgs) > 0 {
			placeHolders := strings.TrimSuffix(strings.Repeat("?,", len(queryArgs)), ",")
			query = fmt.Sprintf(query, placeHolders)
			_, err := tx.ExecContext(ctx, query, queryArgs...)
			if err != nil {
				return fmt.Errorf("query for updating artists failed: %w", err)
			}
		}

		return nil
	}

	if err := lib.ExecuteDBJobAndWait(work); err != nil {
		log.Printf("Error executing remove favourites: %s", err)
		return fmt.Errorf("removing from the database failed: %w", err)
	}

	return nil
}
