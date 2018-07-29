package library

import (
	"log"

	migrate "github.com/ironsmile/sql-migrate"
)

// sqlMigrationsPath is path to the migrations directory which contains SQL
// migrations go sql-migrate. It must be relative to `sqlFilesPath`.
const sqlMigrationsPath = "migrations"

// applyMigrations reads the database migrations dir and applies them to the currently
// open database if it is necessary.
func (lib *LocalLibrary) applyMigrations() error {

	migrations := &migrate.PackrMigrationSource{
		Box: lib.sqlFiles,
		Dir: sqlMigrationsPath,
	}

	_, err := migrate.ExecMax(lib.db, "sqlite3", migrations, migrate.Up, 0)
	if err == nil {
		return nil
	}

	if _, ok := err.(*migrate.PlanError); ok {
		log.Printf("Error applying database migrations: %s\n", err)
		return nil
	}

	return err
}
