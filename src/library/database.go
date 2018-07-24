package library

import (
	"database/sql"
	"errors"
	"log"
	"runtime"
)

// DatabaseExecutable is the type used for passing "work unit" to the databaseWorker.
// Every function which wants to do something with the database creates one and sends
// it to the databaseWorker for execution.
type DatabaseExecutable func(db *sql.DB) error

// Reads from the media channel and saves into the database every file
// received.
func (lib *LocalLibrary) databaseWorker() {
	defer func() {
		close(lib.dbExecutes)
		lib.dbWorkerWG.Done()
	}()
	runtime.LockOSThread()

	for {
		select {
		case executable, ok := <-lib.dbExecutes:
			if !ok {
				return
			}
			if err := executable(lib.db); err != nil {
				log.Printf("Error from db executable: %s", err)
			}
		case <-lib.ctx.Done():
			return
		}
	}
}

// The only possible error from executeDBJob is one from the closed context.
func (lib *LocalLibrary) executeDBJob(executable DatabaseExecutable) error {
	select {
	case lib.dbExecutes <- executable:
		return nil
	case <-lib.ctx.Done():
		return lib.ctx.Err()
	}
}

// Returns the last ID insert in the database.
func lastInsertID(db *sql.DB) (int64, error) {
	var id int64

	if db == nil {
		return 0, errors.New("The db connection property was nil")
	}

	err := db.QueryRow("SELECT last_insert_rowid();").Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}
