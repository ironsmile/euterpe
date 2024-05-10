package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
)

// SetTrackRating stores the rating for particular track into the database.
func (lib *LocalLibrary) SetTrackRating(
	ctx context.Context,
	mediaID int64,
	rating uint8,
) error {
	if rating > 5 {
		return errWrongRating
	}
	work := func(db *sql.DB) error {
		var dbRating any = rating
		if rating == 0 {
			dbRating = nil
		}

		_, err := db.ExecContext(
			ctx,
			`
				INSERT INTO user_stats (track_id, user_rating)
				VALUES (@trackID, @rating)
				ON CONFLICT(track_id) DO UPDATE SET
					user_rating = @rating
			`,
			sql.Named("trackID", mediaID),
			sql.Named("rating", dbRating),
		)

		return err
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing set track rating: %s", err)
		return fmt.Errorf("failed SQL query: %w", err)
	}

	return nil
}

// SetAlbumRating stores the rating for particular album into the database.
func (lib *LocalLibrary) SetAlbumRating(
	ctx context.Context,
	albumID int64,
	rating uint8,
) error {
	if rating > 5 {
		return errWrongRating
	}
	work := func(db *sql.DB) error {
		var dbRating any = rating
		if rating == 0 {
			dbRating = nil
		}

		_, err := db.ExecContext(
			ctx,
			`
				INSERT INTO albums_stats (album_id, user_rating)
				VALUES (@albumID, @rating)
				ON CONFLICT(album_id) DO UPDATE SET
					user_rating = @rating
			`,
			sql.Named("albumID", albumID),
			sql.Named("rating", dbRating),
		)

		return err
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing set album rating: %s", err)
		return fmt.Errorf("failed SQL query: %w", err)
	}

	return nil
}

// SetArtistRating stores the rating for particular artist into the database.
func (lib *LocalLibrary) SetArtistRating(
	ctx context.Context,
	artistID int64,
	rating uint8,
) error {
	if rating > 5 {
		return errWrongRating
	}
	work := func(db *sql.DB) error {
		var dbRating any = rating
		if rating == 0 {
			dbRating = nil
		}

		_, err := db.ExecContext(
			ctx,
			`
				INSERT INTO artists_stats (artist_id, user_rating)
				VALUES (@artistID, @rating)
				ON CONFLICT(artist_id) DO UPDATE SET
					user_rating = @rating
			`,
			sql.Named("artistID", artistID),
			sql.Named("rating", dbRating),
		)

		return err
	}

	if err := lib.executeDBJobAndWait(work); err != nil {
		log.Printf("Error executing set artist rating: %s", err)
		return fmt.Errorf("failed SQL query: %w", err)
	}

	return nil
}

var errWrongRating = errors.New("rating must be in [0-5] range")
