package handlers

import (
	"errors"
	"net/http"
	"strings"

	"envdash/internal/firebase"
	"envdash/internal/services"
)

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
