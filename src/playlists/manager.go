package playlists

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ironsmile/euterpe/src/library"
)

// manager implements the Playlister interface by just requiring a function for
// sending database work.
type manager struct {
	executeDBJobAndWait func(library.DatabaseExecutable) error
}

// NewManager returns a Playlister which will send SQL queries to `sendDBWork`.
func NewManager(sendDBWork func(library.DatabaseExecutable) error) Playlister {
	return &manager{
		executeDBJobAndWait: sendDBWork,
	}
}

// Get implements Playlister.
func (m *manager) Get(ctx context.Context, id int64) (Playlist, error) {
	const getPlaylistQuery = selectPlaylistQuery + `
		WHERE pl.id = @playlist_id
		GROUP BY pl.id
	`

	const getTrackIDsQuery = `
		SELECT track_id, "index" FROM playlists_tracks
		WHERE playlist_id = @playlist_id
	`

	var playlist Playlist

	work := func(db *sql.DB) error {
		row := db.QueryRowContext(ctx, getPlaylistQuery, sql.Named("playlist_id", id))
		scanned, err := scanPlaylist(row)
		if err != nil && errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return err
		}

		playlist = scanned

		var (
			playlistTracks []any
			trackOrder     = map[int64]int64{}
		)
		res, err := db.QueryContext(ctx, getTrackIDsQuery, sql.Named("playlist_id", id))
		if err != nil {
			return fmt.Errorf("failed to get track IDs: %w", err)
		}
		for res.Next() {
			var (
				trackID int64
				index   int64
			)

			if err := res.Scan(&trackID, &index); err != nil {
				return fmt.Errorf("failed to scan track: %w", err)
			}

			playlistTracks = append(playlistTracks, trackID)
			trackOrder[trackID] = index
		}

		if len(playlistTracks) == 0 {
			return nil
		}

		queryTracksWhere := []string{
			"t.id IN (" + strings.TrimSuffix(
				strings.Repeat("?,", len(playlistTracks)),
				",",
			) + ")",
		}

		rows, err := library.QueryTracks(ctx, db, queryTracksWhere, "", playlistTracks)
		if err != nil {
			return fmt.Errorf("error selecting tracks for playlist: %w", err)
		}

		var tracks []library.TrackInfo
		for rows.Next() {
			track, err := library.ScanTrack(rows)
			if err != nil {
				return fmt.Errorf("error while scanning a track: %w", err)
			}

			tracks = append(tracks, track)
		}

		slices.SortFunc(tracks, func(a library.TrackInfo, b library.TrackInfo) int {
			if trackOrder[a.ID] < trackOrder[b.ID] {
				return -1
			} else if trackOrder[a.ID] > trackOrder[b.ID] {
				return 1
			}

			return 0
		})

		playlist.Tracks = tracks

		return nil
	}
	if err := m.executeDBJobAndWait(work); err != nil {
		return Playlist{}, err
	}

	return playlist, nil
}

// Count implements Playlister.
func (m *manager) Count(ctx context.Context) (int64, error) {
	var playlistsCount int64

	work := func(db *sql.DB) error {
		var count sql.NullInt64

		row := db.QueryRowContext(ctx, countPlaylistsQuery)
		if err := row.Scan(&count); err != nil {
			return fmt.Errorf("error in SQL query for getting playlists count: %w", err)
		}

		if count.Valid {
			playlistsCount = count.Int64
		} else {
			return fmt.Errorf("SQL query did not return rows for playlists count")
		}

		return nil
	}

	if err := m.executeDBJobAndWait(work); err != nil {
		return 0, err
	}

	return playlistsCount, nil
}

// List implements Playlister.
func (m *manager) List(ctx context.Context, args ListArgs) ([]Playlist, error) {
	var (
		playlists []Playlist
		queryArgs []any

		querySuffix = `
		GROUP BY
			pl.id
		`
	)

	if args.Count > 0 || args.Offset > 0 {
		querySuffix += `
		LIMIT ?, ?
		`
		queryArgs = append(queryArgs, args.Offset, args.Count)
	}

	getPlaylistsQuery := selectPlaylistQuery + querySuffix

	work := func(db *sql.DB) error {
		rows, err := db.QueryContext(ctx, getPlaylistsQuery, queryArgs...)
		if err != nil {
			return fmt.Errorf("could not query the database: %w", err)
		}

		for rows.Next() {
			playlist, err := scanPlaylist(rows)
			if err != nil {
				return fmt.Errorf("error scanning playlists: %w", err)
			}

			playlists = append(playlists, playlist)
		}

		return nil
	}
	if err := m.executeDBJobAndWait(work); err != nil {
		return nil, err
	}

	return playlists, nil
}

// Create implements Playlister.
func (m *manager) Create(
	ctx context.Context,
	name string,
	tracks []int64,
) (int64, error) {
	if name == "" {
		return 0, fmt.Errorf("name cannot be empty")
	}

	var lastInsertID int64

	insertPlaylistQuery := `
		INSERT INTO
			playlists (name, public, created_at, updated_at)
		VALUES
			(@name, 1, @current_time, @current_time)
	`

	insertSongsQuery := `
		INSERT INTO
			playlists_tracks (playlist_id, track_id, "index")
		VALUES
	`

	work := func(db *sql.DB) (retErr error) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("cannot begin DB transaction: %w", err)
		}
		defer func() {
			if retErr == nil {
				if commitErr := tx.Commit(); err != nil {
					retErr = commitErr
				}
			} else {
				_ = tx.Rollback()
			}
		}()

		res, err := tx.ExecContext(ctx, insertPlaylistQuery,
			sql.Named("name", name),
			sql.Named("current_time", time.Now().Unix()),
		)
		if err != nil {
			return fmt.Errorf("failed to insert playlist: %w", err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("cannot get last insert ID for playlist: %w", err)
		}

		lastInsertID = id
		if len(tracks) == 0 {
			return nil
		}

		insertSongsQuery += strings.TrimSuffix(strings.Repeat(
			"(@playlist_id, ?, ?),", len(tracks),
		), ",")

		queryVals := []any{
			sql.Named("playlist_id", lastInsertID),
		}
		for index, trackID := range tracks {
			queryVals = append(queryVals, trackID, index)
		}

		_, err = tx.ExecContext(ctx, insertSongsQuery, queryVals...)
		if err != nil {
			return fmt.Errorf("failed to insert playlist: %w", err)
		}

		return nil
	}

	if err := m.executeDBJobAndWait(work); err != nil {
		return 0, err
	}

	return lastInsertID, nil
}

// Update implements Playlister.
func (m *manager) Update(ctx context.Context, id int64, args UpdateArgs) error {
	var (
		updateFields []string
		updateValues []any
	)

	if args.Name != "" {
		updateFields = append(updateFields, "name = @name")
		updateValues = append(updateValues, sql.Named("name", args.Name))
	}

	if args.Desc != "" {
		updateFields = append(updateFields, "description = @description")
		updateValues = append(updateValues, sql.Named("description", args.Desc))
	}

	if args.Public != nil {
		var publicInt = 1
		if !*args.Public {
			publicInt = 0
		}
		updateFields = append(updateFields, "public = @public")
		updateValues = append(updateValues, sql.Named("public", publicInt))
	}

	if len(updateFields) == 0 && !args.RemoveAllTracks &&
		len(args.AddTracks) == 0 && len(args.RemoveTracks) == 0 &&
		len(args.MoveTracks) == 0 {
		// nothing to do here!
		return nil
	}

	updateFields = append(updateFields, "updated_at = @updated_time")
	updateValues = append(updateValues,
		sql.Named("updated_time", time.Now().Unix()),
		sql.Named("playlist_id", id),
	)

	updatePlaylistQuery := `
		UPDATE playlists
		SET
			` + strings.Join(updateFields, ",") + `
		WHERE
			id = @playlist_id
	`

	const removeAllQuery = `
		DELETE FROM playlists_tracks
		WHERE
			playlist_id = @playlist_id
	`

	insertSongsQuery := `
		INSERT INTO
			playlists_tracks (playlist_id, track_id, "index")
		VALUES
	`

	const removeTracksQuery = `
		DELETE FROM playlists_tracks
		WHERE
			playlist_id = @playlist_id AND
			"index" = @track_index
	`

	const updateTrackIndexesQuery = `
		UPDATE playlists_tracks
		SET
			"index" = "index" - 1
		WHERE
			playlist_id = @playlist_id AND
			"index" > @track_index
	`

	const getTrackByIndexQuery = `
		SELECT
			track_id
		FROM
			playlists_tracks
		WHERE
			playlist_id = @playlist_id AND
			"index" = @track_index
	`

	const createIndexGapQuery = `
		UPDATE playlists_tracks
		SET
			"index" = "index" + 1
		WHERE
			playlist_id = @playlist_id AND
			"index" >= @track_index
	`

	const maxIndexQuery = `
		SELECT
			MAX("index") as max_index
		FROM
			playlists_tracks
		WHERE
			playlist_id = @playlist_id
	`

	work := func(db *sql.DB) (retErr error) {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("cannot begin DB transaction: %w", err)
		}
		defer func() {
			if retErr == nil {
				if commitErr := tx.Commit(); err != nil {
					retErr = commitErr
				}
			} else {
				_ = tx.Rollback()
			}
		}()

		res, err := tx.ExecContext(ctx, updatePlaylistQuery, updateValues...)
		if err != nil && errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("playlist for updating not found: %w", ErrNotFound)
		} else if err != nil {
			return fmt.Errorf("update playlist error: %w", err)
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("could not get the number of affected rows: %w", err)
		}

		if affected < 1 {
			return fmt.Errorf("playlist for updating not found: %w", ErrNotFound)
		}

		if args.RemoveAllTracks {
			_, err := tx.ExecContext(ctx, removeAllQuery, sql.Named("playlist_id", id))
			if err != nil {
				return fmt.Errorf("failed to remove all tracks from playlist: %w", err)
			}
		}

		slices.Sort(args.RemoveTracks)
		slices.Reverse(args.RemoveTracks)

		for _, trackIndex := range args.RemoveTracks {
			var removeArgs = []any{
				sql.Named("playlist_id", id),
				sql.Named("track_index", trackIndex),
			}

			_, err = tx.ExecContext(ctx, removeTracksQuery, removeArgs...)
			if err != nil {
				return fmt.Errorf("failed to remove track index %d: %w", trackIndex, err)
			}

			_, err = tx.ExecContext(ctx, updateTrackIndexesQuery, removeArgs...)
			if err != nil {
				return fmt.Errorf("failed to update track index %d: %w", trackIndex, err)
			}
		}

		if len(args.AddTracks) > 0 {
			var (
				maxIndex  sql.NullInt64
				nextIndex int64
			)

			row := tx.QueryRowContext(ctx, maxIndexQuery, sql.Named("playlist_id", id))
			if err := row.Scan(&maxIndex); err != nil {
				return fmt.Errorf("failed to scan max index: %w", err)
			}

			if maxIndex.Valid {
				nextIndex = maxIndex.Int64 + 1
			}

			insertSongsQuery += strings.TrimSuffix(strings.Repeat(
				"(@playlist_id, ?, ?),", len(args.AddTracks),
			), ",")

			queryVals := []any{
				sql.Named("playlist_id", id),
			}
			for i, trackID := range args.AddTracks {
				queryVals = append(queryVals, trackID, nextIndex+int64(i))
			}

			_, err = tx.ExecContext(ctx, insertSongsQuery, queryVals...)
			if err != nil {
				return fmt.Errorf("failed to insert songs to playlist: %w", err)
			}
		}

		for ind, move := range args.MoveTracks {
			if move.FromIndex == move.ToIndex {
				continue
			}

			var trackID int64
			row := tx.QueryRowContext(ctx, getTrackByIndexQuery,
				sql.Named("playlist_id", id),
				sql.Named("track_index", move.FromIndex),
			)
			if err := row.Scan(&trackID); err != nil {
				return fmt.Errorf("failed to scan for track for move %d (%d->%d): %w",
					ind, move.FromIndex, move.ToIndex, err)
			}

			var removeArgs = []any{
				sql.Named("playlist_id", id),
				sql.Named("track_index", move.FromIndex),
			}

			_, err = tx.ExecContext(ctx, removeTracksQuery, removeArgs...)
			if err != nil {
				return fmt.Errorf("failed to remove track index (moving) %d: %w",
					move.FromIndex, err)
			}

			_, err = tx.ExecContext(ctx, updateTrackIndexesQuery, removeArgs...)
			if err != nil {
				return fmt.Errorf("failed to update track index (moving) %d: %w",
					move.FromIndex, err)
			}

			var gapArgs = []any{
				sql.Named("playlist_id", id),
				sql.Named("track_index", move.ToIndex),
			}

			_, err = tx.ExecContext(ctx, createIndexGapQuery, gapArgs...)
			if err != nil {
				return fmt.Errorf("failed to create gap during move (%d): %w", ind, err)
			}

			insertMovedQuery := `
				INSERT INTO
					playlists_tracks (playlist_id, track_id, "index")
				VALUES
					(@playlist_id, @track_id, @track_index)
			`

			var insertArgs = []any{
				sql.Named("playlist_id", id),
				sql.Named("track_id", trackID),
				sql.Named("track_index", move.ToIndex),
			}

			_, err = tx.ExecContext(ctx, insertMovedQuery, insertArgs...)
			if err != nil {
				return fmt.Errorf("failed insert track index during moving (move %d): %w",
					ind, err)
			}
		}

		return nil
	}

	if err := m.executeDBJobAndWait(work); err != nil {
		return err
	}

	return nil
}

// Delete implements Playlister.
func (m *manager) Delete(ctx context.Context, id int64) error {
	const deletePlaylistQuery = `
		DELETE FROM playlists
		WHERE id = @playlist_id
	`

	work := func(db *sql.DB) (retErr error) {
		res, err := db.ExecContext(ctx, deletePlaylistQuery, sql.Named("playlist_id", id))
		if err != nil {
			return fmt.Errorf("sql query error: %w", err)
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("cannot get number of affected rows: %w", err)
		}

		if affected < 1 {
			return ErrNotFound
		}

		return nil
	}

	if err := m.executeDBJobAndWait(work); err != nil {
		return err
	}

	return nil
}

const selectPlaylistQuery = `
	SELECT 
		pl.id,
		pl.name,
		pl.description,
		pl.public,
		pl.created_at,
		pl.updated_at,
		COUNT(pt.track_id) as track_count,
		SUM(t.duration) as duration
	FROM
		playlists pl
		LEFT JOIN playlists_tracks pt ON pl.id = pt.playlist_id
		LEFT JOIN tracks t ON pt.track_id = t.id
`

const countPlaylistsQuery = `
	SELECT
		COUNT(*) as cnt
	FROM
		playlists pl
`

func scanPlaylist(row rowScanner) (Playlist, error) {
	var (
		playlist    Playlist
		description sql.NullString
		public      int64
		created     int64
		updated     int64
		trackCount  sql.NullInt64
		duration    sql.NullInt64
	)

	err := row.Scan(
		&playlist.ID, &playlist.Name, &description,
		&public, &created, &updated, &trackCount, &duration,
	)
	if err != nil {
		return Playlist{}, fmt.Errorf("error scanning playlist: %w", err)
	}

	if description.Valid {
		playlist.Desc = description.String
	}

	if public != 0 {
		playlist.Public = true
	}

	if duration.Valid {
		playlist.Duration = time.Duration(duration.Int64) * time.Millisecond
	}

	if trackCount.Valid {
		playlist.TracksCount = trackCount.Int64
	}

	playlist.CreatedAt = time.Unix(created, 0)
	playlist.UpdatedAt = time.Unix(updated, 0)

	return playlist, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}
