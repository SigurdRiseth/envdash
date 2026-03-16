package handlers

import (
	"net/http"

	"envdash/internal/services"
)

type authHandler struct {
	svc services.AuthService
}

// newAuthHandler creates an authHandler backed by the given AuthService.
func newAuthHandler(svc services.AuthService) *authHandler {
	return &authHandler{svc: svc}
}

// handleCollection handles POST /auth/ — generate and persist a new API key.
// Responds 201 Created with {"apiKey": "<key>"} on success.
// Only POST is accepted; all other methods return 405 Method Not Allowed.
func (h *authHandler) handleCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	key, err := h.svc.Register(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"apiKey": key})
}

// handleItem handles DELETE /auth/{key} — permanently revoke an existing API key.
// The key is extracted directly from the URL path. Responds 204 No Content on
// success, 404 Not Found if the key does not exist, or 400 Bad Request if no
// key segment is present in the path. Only DELETE is accepted.
func (h *authHandler) handleItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	key := extractID(r.URL.Path, apiPrefix+"/auth/")
	if key == "" {
		writeError(w, http.StatusBadRequest, "missing API key")
		return
	}

	if err := h.svc.Revoke(r.Context(), key); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
