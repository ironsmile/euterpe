package library

import (
	"log"
	"path/filepath"

	"github.com/ironsmile/httpms/src/helpers"
	migrate "github.com/ironsmile/sql-migrate"
)

// applyMigrations reads the database migrations dir and applies them to the currently
// open database if it is necessary.
func (lib *LocalLibrary) applyMigrations() error {
	projRoot, err := helpers.ProjectRoot()
	if err != nil {
		return err
	}

	migrationsDir := migrate.FileMigrationSource{
		Dir: filepath.Join(projRoot, "sqls", "migrations"),
	}

	_, err = migrate.ExecMax(lib.db, "sqlite3", migrationsDir, migrate.Up, 0)
	if err == nil {
		return nil
	}

	if _, ok := err.(*migrate.PlanError); ok {
		log.Printf("Error applying database: migrations: %s\n", err)
		return nil
	}

	return err
}
