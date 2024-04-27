package subsonic

import (
	"net/http"
	"strconv"
	"strings"
)

// getProtoFromRequest returns the original request scheme used for accessing
// the server. It takes into account the X-Forwarded-Proto and the Forwarded
// HTTP headers.
func getProtoFromRequest(req *http.Request) string {
	proto := "http"
	if forwadedProto := req.Header.Get("X-Forwarded-Proto"); forwadedProto == "https" {
		proto = "https"
	}

	if forwarded := req.Header.Get("Forwarded"); forwarded != "" {
		vals := splitForwarded(forwarded)
		if forwardedProto, ok := vals["proto"]; ok && forwardedProto == "https" {
			proto = "https"
		}
	}

	return proto
}

// getHostFromRequest returns the original request Host used for accessing the
// server. It takes into account the X-Forwarded-Host and Forwarded HTTP headers.
func getHostFromRequest(req *http.Request) string {
	host := req.Host
	if forwadedHost := req.Header.Get("X-Forwarded-Host"); forwadedHost != "" {
		host = forwadedHost
	}

	if forwarded := req.Header.Get("Forwarded"); forwarded != "" {
		vals := splitForwarded(forwarded)
		if forwardedHost, ok := vals["host"]; ok {
			host = forwardedHost
		}
	}

	return host
}

// splitForwarded splits the value of the HTTP header Forwarded into a map of
// keys and values.
func splitForwarded(val string) map[string]string {
	vals := make(map[string]string)

	// Example:
	// Forwarded: by=<identifier>;for=<identifier>;host=<host>;proto=<http|https>
	pairs := strings.Split(val, ";")
	for _, pair := range pairs {
		k, v, ok := strings.Cut(pair, "=")
		if !ok || v == "" {
			continue
		}

		vals[k] = strings.Trim(v, `"`)
	}

	return vals
}

// parseIntOrDefault parses `s` as a base 10 int and on error returns def.
func parseIntOrDefault(s string, def uint32) uint32 {
	val, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return def
	}
	return uint32(val)
}
