package library

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"

	migrate "github.com/ironsmile/sql-migrate"
)

// sqlMigrateDirectory is the directory whithin the `sqlFilesFS` which contains
// the .sql files for sql-migrate.
const sqlMigrateDirectory = "migrations"

// applyMigrations reads the database migrations dir and applies them to the currently
// open database if it is necessary.
func (lib *LocalLibrary) applyMigrations() error {
	migrationFiles, err := fs.Sub(lib.sqlFilesFS, sqlMigrateDirectory)
	if err != nil {
		return fmt.Errorf("locating migrate dir within sqlFiles fs.FS failed: %w", err)
	}

	migrations := &migrate.HttpFileSystemMigrationSource{
		FileSystem: http.FS(migrationFiles),
	}

	_, err = migrate.ExecMax(lib.db, "sqlite3", migrations, migrate.Up, 0)
	if err == nil {
		return nil
	}

	if _, ok := err.(*migrate.PlanError); ok {
		log.Printf("Error applying database migrations: %s\n", err)
		return nil
	}

	return fmt.Errorf("executing db migration failed: %w", err)
}
