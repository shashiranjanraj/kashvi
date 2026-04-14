package response

import "net/http"

// DebugError sends a JSON error response with optional debug fields.
// Use this only in development; in production prefer Error().
func DebugError(w http.ResponseWriter, status int, message string, debug any) {
	write(w, status, envelope{Status: status, Message: message, Errors: debug})
}
