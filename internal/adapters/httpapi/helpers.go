// Package httpapi is the inbound HTTP transport adapter. The package name is
// "httpapi" rather than "http" to avoid shadowing the standard library
// net/http package in callers.
package httpapi

import (
	"encoding/json"
	"net/http"
)

// writeJSON encodes body as JSON with the given status. Errors during encode
// are silently dropped — they almost always mean the client disconnected.
func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// userIDFromRequest extracts a user identifier from the request. Authentication
// is not yet implemented (see TODO.md); every request is attributed to the
// hardcoded default user.
func userIDFromRequest(_ *http.Request) string {
	return "DefaultUser"
}
