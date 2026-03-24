package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"envdash/internal/firebase"
	"envdash/internal/services"
)

// decodeJSON validates the request Content-Type and decodes the JSON body into v.
// Returns true on success. Returns false and writes an appropriate error response
// if the Content-Type is not "application/json" (415 Unsupported Media Type) or
// if the body cannot be decoded (400 Bad Request). Callers should return immediately
// on false — the error has already been written to w.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	ct := r.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

// extractID extracts the resource ID from a URL path by stripping the given prefix
// and any trailing slash. For example, extractID("/envdash/v1/registrations/abc123",
// "/envdash/v1/registrations/") returns "abc123". Returns an empty string if no
// ID segment follows the prefix.
func extractID(path, prefix string) string {
	id := strings.TrimPrefix(path, prefix)
	return strings.TrimSuffix(id, "/")
}

// handleServiceError translates service-layer errors into appropriate HTTP responses.
// firebase.ErrNotFound → 404 Not Found
// *services.ValidationError → 400 Bad Request (with the validation message)
// all other errors → 500 Internal Server Error
func handleServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, firebase.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var ve *services.ValidationError
	if errors.As(err, &ve) {
		writeError(w, http.StatusBadRequest, ve.Message)
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}
