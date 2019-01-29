package library

import (
	"database/sql"
	"errors"
	"log"
	"runtime"
	"sync"
)

// DatabaseExecutable is the type used for passing "work unit" to the databaseWorker.
// Every function which wants to do something with the database creates one and sends
// it to the databaseWorker for execution.
type DatabaseExecutable func(db *sql.DB) error

// Reads from the media channel and saves into the database every file
// received.
//
// !TODO: Separate the database worker from the library. It should be stand alone
// type. Ideally the library will not have direct access to the sql.DB object.
func (lib *LocalLibrary) databaseWorker(wg *sync.WaitGroup) {
	lib.dbExecutes = make(chan DatabaseExecutable)
	runtime.LockOSThread()

	wg.Done()
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

// executeDBJobAndWait executes the `executable`, waits for it to finish. Then returns
// its error.
func (lib *LocalLibrary) executeDBJobAndWait(executable DatabaseExecutable) error {
	var executableErr error
	done := make(chan struct{})
	defer close(done)

	work := func(db *sql.DB) error {
		defer func() {
			done <- struct{}{}
		}()
		executableErr = executable(db)
		return nil
	}

	if err := lib.executeDBJob(work); err != nil {
		return err
	}

	<-done
	return executableErr
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
