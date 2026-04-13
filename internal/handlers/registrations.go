package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"envdash/internal/models"
	"envdash/internal/services"
)

// registrationHandler handles HTTP requests for dashboard registrations.
// It delegates all business logic to a RegistrationService and only concerns
// itself with request parsing, response formatting, and error translation.
type registrationHandler struct {
	svc services.RegistrationService
}

// newRegistrationHandler creates a registrationHandler backed by the given RegistrationService.
func newRegistrationHandler(svc services.RegistrationService) *registrationHandler {
	return &registrationHandler{svc: svc}
}

// handleCollection routes POST, GET, and HEAD requests for the registrations collection.
// POST creates a new registration, GET lists all registrations, and HEAD returns
// collection metadata headers without a body. All other methods return 405.
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

// handleItem routes GET, PUT, PATCH, and DELETE requests for a specific registration.
// The registration ID is extracted from the URL path. Returns 400 Bad Request
// if no ID segment is present, and 405 Method Not Allowed for unsupported methods.
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

// create handles POST /registrations/ — create a new dashboard registration.
// Decodes a RegistrationRequest from the JSON body and delegates to the service.
// Responds 201 Created with the new registration's ID and creation timestamp.
func (h *registrationHandler) create(w http.ResponseWriter, r *http.Request) {
	var req models.RegistrationRequest
	if !decodeJSON(w, r, &req) {
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

// list handles GET /registrations/ — return all registered dashboards.
// Always responds 200 OK; returns an empty array if no registrations exist.
func (h *registrationHandler) list(w http.ResponseWriter, r *http.Request) {
	regs, err := h.svc.List(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, regs)
}

// head handles HEAD /registrations/ — return collection metadata without a response body.
// Sets Content-Type and Content-Length to the same values a GET would produce,
// allowing clients to determine payload size before making a full request.
func (h *registrationHandler) head(w http.ResponseWriter, r *http.Request) {
	regs, err := h.svc.List(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}
	// Encode to determine Content-Length
	data, err := json.Marshal(regs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
}

// get handles GET /registrations/{id} — return a single registration by ID.
// Responds 200 OK with the full registration document, or 404 Not Found if
// no registration with the given ID exists.
func (h *registrationHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	reg, err := h.svc.Get(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reg)
}

// update handles PUT /registrations/{id} — fully replace an existing registration.
// All fields must be provided in the request body; omitted fields are overwritten
// with zero values. Responds 200 OK with the updated registration on success.
func (h *registrationHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	var req models.RegistrationRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	_, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// patch handles PATCH /registrations/{id} — partially update a registration.
// Only the fields present in the JSON body are updated; all other fields are
// left unchanged. Responds 200 OK with the full updated registration.
func (h *registrationHandler) patch(w http.ResponseWriter, r *http.Request, id string) {
	var patch map[string]any
	if !decodeJSON(w, r, &patch) {
		return
	}

	updated, err := h.svc.Patch(r.Context(), id, patch)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// delete handles DELETE /registrations/{id} — permanently remove a registration.
// Responds 204 No Content on success, or 404 Not Found if the ID does not exist.
func (h *registrationHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.svc.Delete(r.Context(), id); err != nil {
		handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
