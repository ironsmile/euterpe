package radio

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"slices"
	"strings"

	"github.com/ironsmile/euterpe/src/library"
)

// manager implements the Stations interface by just requiring a function for
// sending database work.
type manager struct {
	executeDBJobAndWait func(library.DatabaseExecutable) error
}

// NewManager returns a Stations interface which will use the `sendDBWork` to
// execute its database queries.
func NewManager(sendDBWork func(library.DatabaseExecutable) error) Stations {
	return &manager{
		executeDBJobAndWait: sendDBWork,
	}
}

// GetAll implements the Stations interface.
func (m *manager) GetAll(ctx context.Context) ([]Station, error) {
	var stations []Station
	query := `
		SELECT id, name, stream_url, home_page
		FROM radio_stations
	`

	work := func(db *sql.DB) error {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return fmt.Errorf("could not query the database: %w", err)
		}

		for rows.Next() {
			var (
				station   Station
				homePage  sql.NullString
				streamURL sql.NullString
			)

			err := rows.Scan(&station.ID, &station.Name, &streamURL, &homePage)
			if err != nil {
				return fmt.Errorf("error scanning station: %w", err)
			}

			streamParseURL, err := url.Parse(streamURL.String)
			if err != nil {
				log.Printf(
					"Could no parse radio_station.stream_url %d from DB: %s",
					station.ID,
					err,
				)
				continue
			}
			station.StreamURL = *streamParseURL

			if homePage.Valid {
				homePageParseURL, err := url.Parse(homePage.String)
				if err != nil {
					log.Printf(
						"Could no parse radio_station.home_page %d from DB: %s",
						station.ID,
						err,
					)
					continue
				}
				station.HomePage = homePageParseURL
			}

			stations = append(stations, station)
		}

		return nil
	}
	if err := m.executeDBJobAndWait(work); err != nil {
		return nil, err
	}

	return stations, nil
}

// Create implements the Stations interface.
func (m *manager) Create(ctx context.Context, new Station) (int64, error) {
	if new.Name == "" {
		return 0, fmt.Errorf("name cannot be empty")
	}

	allowedSchemes := []string{"http", "https"}

	if !slices.Contains(allowedSchemes, new.StreamURL.Scheme) {
		return 0, fmt.Errorf(
			"stream URL scheme can only be one of %s",
			strings.Join(allowedSchemes, ", "),
		)
	}
	if new.HomePage != nil && !slices.Contains(allowedSchemes, new.HomePage.Scheme) {
		return 0, fmt.Errorf(
			"home page URL scheme can only be one of %s",
			strings.Join(allowedSchemes, ", "),
		)
	}

	var lastInsertID int64

	query := `
		INSERT INTO
			radio_stations (name, stream_url, home_page)
		VALUES
			(@name, @streamURL, @homePage)
	`

	work := func(db *sql.DB) (retErr error) {
		homePage := sql.Named("homePage", nil)
		if new.HomePage != nil {
			homePage = sql.Named("homePage", new.HomePage.String())
		}
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

		res, err := tx.ExecContext(ctx, query,
			sql.Named("name", new.Name),
			sql.Named("streamURL", new.StreamURL.String()),
			homePage,
		)
		if err != nil {
			return fmt.Errorf("failed to insert internet radio: %w", err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("cannot get last insert ID for radio station: %w", err)
		}

		lastInsertID = id
		return nil
	}

	if err := m.executeDBJobAndWait(work); err != nil {
		return 0, err
	}

	return lastInsertID, nil
}

// Replace implements the Stations interface.
func (m *manager) Replace(ctx context.Context, updated Station) error {
	if updated.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	allowedSchemes := []string{"http", "https"}

	if !slices.Contains(allowedSchemes, updated.StreamURL.Scheme) {
		return fmt.Errorf(
			"stream URL scheme can only be one of %s",
			strings.Join(allowedSchemes, ", "),
		)
	}
	if updated.HomePage != nil && !slices.Contains(allowedSchemes, updated.HomePage.Scheme) {
		return fmt.Errorf(
			"home page URL scheme can only be one of %s",
			strings.Join(allowedSchemes, ", "),
		)
	}

	query := `
		UPDATE
			radio_stations
		SET
			name = @name,
			stream_url = @streamURL,
			home_page = @homePage
		WHERE
			id = @id
	`

	work := func(db *sql.DB) (retErr error) {
		homePage := sql.Named("homePage", nil)
		if updated.HomePage != nil {
			homePage = sql.Named("homePage", updated.HomePage.String())
		}
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

		res, err := tx.ExecContext(ctx, query,
			sql.Named("id", updated.ID),
			sql.Named("name", updated.Name),
			sql.Named("streamURL", updated.StreamURL.String()),
			homePage,
		)
		if err != nil {
			return fmt.Errorf("failed to update internet radio: %w", err)
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("cannot get the number of affected stations: %w", err)
		}

		if affected != 1 {
			return ErrNotFound
		}

		return nil
	}

	if err := m.executeDBJobAndWait(work); err != nil {
		return err
	}

	return nil
}

// Delete implements the Stations interface.
func (m *manager) Delete(ctx context.Context, stationID int64) error {
	query := `
		DELETE FROM
			radio_stations
		WHERE
			id = @id
	`

	work := func(db *sql.DB) (retErr error) {
		res, err := db.ExecContext(ctx, query, sql.Named("id", stationID))
		if err != nil {
			return fmt.Errorf("failed to delete internet radio: %w", err)
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("cannot get the number of affected stations: %w", err)
		}

		if affected < 1 {
			return ErrNotFound
		}

		if affected != 1 {
			return fmt.Errorf("deleted more than one radio station")
		}

		return nil
	}

	if err := m.executeDBJobAndWait(work); err != nil {
		return err
	}

	return nil
}
