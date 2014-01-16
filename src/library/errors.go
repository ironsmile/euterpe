package library

import (
	"fmt"
)

// Returned when something has not been found
type ErrorNotFound struct {
	what string
}

// implements error interface
func (err ErrorNotFound) Error() string {
	return fmt.Sprintf("%s was not found", err.what)
}
