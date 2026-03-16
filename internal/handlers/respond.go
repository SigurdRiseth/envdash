package handlers

import (
	"encoding/json"
	"net/http"

	"envdash/internal/models"
)

// writeJSON sets Content-Type to application/json, writes the given HTTP status
// code, and encodes v as JSON into the response body. Encoding errors are
// intentionally ignored — by the time Encode is called the status line has
// already been sent and the error cannot be surfaced to the client.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error response with the given status code and message.
// The body is formatted as {"error": "<msg>"} using models.ErrorResponse.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, models.ErrorResponse{Error: msg})
}
