package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"envdash/internal/firebase"
	"envdash/internal/models"
	"envdash/internal/services"
)

type registrationHandler struct {
	svc services.RegistrationService
}

func newRegistrationHandler(svc services.RegistrationService) *registrationHandler {
	return &registrationHandler{svc: svc}
}

// handleRegistrations routes POST and GET (collection) requests.
func (h *registrationHandler) handleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.create(w, r)
	case http.MethodGet:
		h.list(w, r)
	case http.MethodHead:
		h.head(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleItem routes GET, PUT, PATCH, DELETE requests for a specific registration.
func (h *registrationHandler) handleItem(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, apiPrefix+"/registrations/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing registration ID")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.get(w, r, id)
	case http.MethodPut:
		h.update(w, r, id)
	case http.MethodPatch:
		h.patch(w, r, id)
	case http.MethodDelete:
		h.delete(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *registrationHandler) create(w http.ResponseWriter, r *http.Request) {
	var req models.RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	reg, err := h.svc.Create(r.Context(), req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, models.RegistrationCreateResponse{
		ID:         reg.ID,
		LastChange: reg.LastChange,
	})
}

func (h *registrationHandler) list(w http.ResponseWriter, r *http.Request) {
	regs, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list registrations: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, regs)
}

// head returns the same headers as GET /registrations/ but with no body.
func (h *registrationHandler) head(w http.ResponseWriter, r *http.Request) {
	regs, err := h.svc.List(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Encode to determine Content-Length
	data, _ := json.Marshal(regs)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(http.StatusOK)
}

func (h *registrationHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	reg, err := h.svc.Get(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reg)
}

func (h *registrationHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	var req models.RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	updated, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *registrationHandler) patch(w http.ResponseWriter, r *http.Request, id string) {
	var patch map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	updated, err := h.svc.Patch(r.Context(), id, patch)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *registrationHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.svc.Delete(r.Context(), id); err != nil {
		handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

// extractID strips the prefix from the URL path and returns the remaining segment.
func extractID(path, prefix string) string {
	id := strings.TrimPrefix(path, prefix)
	return strings.TrimSuffix(id, "/")
}
