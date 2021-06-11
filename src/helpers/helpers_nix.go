// +build !windows

/*
   Helpers for all non-windows machines
*/

package helpers

const (
	// euterpeDir is the name of the Euterpe directory in the user's home directory.
	euterpeDir = ".euterpe"

	// httpmsDir was a directory where the Euterpe files were stored before its
	// rename from HTTPMS. Now it is kept for backward compatibility. If this
	// directory is present then it will be used instead of the one in euterpeDir.
	// With the presumption that migration to the new directory hasn't happened yet.
	httpmsDir = ".httpms"
)
