package webserver

import "net/http"

// NewRigisterDeviceHandler returns a handler resposible for checking and eventually
// registering registering in the database token generated to a device.
// !TODO: actually store the device token in the database once it has a unique ID
func NewRigisterDeviceHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNoContent)
	})
}
