package caa

import (
	u "github.com/pborman/uuid"
)

// StringToUUID is a reexported helper function of the UUID module to parse a
// string into a UUID.
func StringToUUID(str string) (uuid u.UUID) {
	return u.Parse(str)
}
