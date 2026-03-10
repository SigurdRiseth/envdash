package handlers

import (
	"errors"
	"net/http"

	"envdash/internal/firebase"
	"envdash/internal/services"
)

type authHandler struct {
	svc services.AuthService
}

func newAuthHandler(svc services.AuthService) *authHandler {
	return &authHandler{svc: svc}
}

// handleCollection handles POST /auth/ — register a new API key.
func (h *authHandler) handleCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	key, err := h.svc.Register(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to register API key")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"apiKey": key})
}

// handleItem handles DELETE /auth/{key} — revoke an API key.
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
		if errors.Is(err, firebase.ErrNotFound) {
			writeError(w, http.StatusNotFound, "API key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to revoke API key")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
