package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"envdash/internal/firebase"
	"envdash/internal/services"
)

// decodeJSON validates Content-Type and decodes the JSON request body into v.
// Returns false and writes an error response if either check fails — callers
// should return immediately on false.
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

// extractID strips the prefix from the URL path and returns the remaining segment.
func extractID(path, prefix string) string {
	id := strings.TrimPrefix(path, prefix)
	return strings.TrimSuffix(id, "/")
}

// handleServiceError maps service-layer errors to appropriate HTTP responses.
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
