package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"envdash/internal/models"
	"envdash/internal/services"
)

// osloLocation is used to stamp createdAt consistently with the rest of the app.
var osloLocation = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		return time.UTC
	}
	return loc
}()

type authHandler struct {
	svc services.AuthService
}

// newAuthHandler creates an authHandler backed by the given AuthService.
func newAuthHandler(svc services.AuthService) *authHandler {
	return &authHandler{svc: svc}
}

// handleCollection handles POST /auth/ — generate and persist a new API key.
// Decodes an optional AuthRequest body (name, email) from JSON; missing or
// malformed bodies are accepted without error since those fields are not
// validated. Responds 201 Created with {"key": "...", "createdAt": "..."}.
// Only POST is accepted; all other methods return 405 Method Not Allowed.
func (h *authHandler) handleCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Decode the request body; name/email are accepted but not currently stored.
	// Errors are ignored — the body is optional for this endpoint.
	var req models.AuthRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	key, err := h.svc.Register(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, models.AuthCreateResponse{
		Key:       key,
		CreatedAt: time.Now().In(osloLocation).Format("20060102 15:04"),
	})
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
