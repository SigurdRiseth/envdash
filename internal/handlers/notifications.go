package handlers

import (
	"net/http"

	"envdash/internal/models"
	"envdash/internal/services"
)

type notificationHandler struct {
	svc services.NotificationService
}

// newNotificationHandler creates a notificationHandler backed by the given NotificationService.
func newNotificationHandler(svc services.NotificationService) *notificationHandler {
	return &notificationHandler{svc: svc}
}

// handleCollection routes POST and GET requests for the notifications collection.
// POST registers a new webhook, GET lists all registered webhooks.
// All other methods return 405 Method Not Allowed.
func (h *notificationHandler) handleCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.create(w, r)
	case http.MethodGet:
		h.list(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleItem routes GET, PATCH, and DELETE requests for a specific notification.
// The notification ID is extracted from the URL path. Returns 400 Bad Request
// if no ID segment is present, and 405 Method Not Allowed for unsupported methods.
func (h *notificationHandler) handleItem(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, apiPrefix+"/notifications/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing notification ID")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.get(w, r, id)
	case http.MethodPatch:
		h.patch(w, r, id)
	case http.MethodDelete:
		h.delete(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// create handles POST /notifications/ — register a new webhook notification.
// Decodes a NotificationRequest from the JSON body and persists it via the service.
// Responds 201 Created with the assigned notification ID.
func (h *notificationHandler) create(w http.ResponseWriter, r *http.Request) {
	var req models.NotificationRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	n, err := h.svc.Create(r.Context(), req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, models.NotificationCreateResponse{ID: n.ID})
}

// list handles GET /notifications/ — return all registered webhook notifications.
// Always responds 200 OK; returns an empty array if no notifications are registered.
func (h *notificationHandler) list(w http.ResponseWriter, r *http.Request) {
	ns, err := h.svc.List(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ns)
}

// get handles GET /notifications/{id} — return a single notification by ID.
// Responds 200 OK with the full notification document, or 404 Not Found if
// no notification with the given ID exists.
func (h *notificationHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	n, err := h.svc.Get(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

// patch handles PATCH /notifications/{id} — partially update a notification.
// Only the fields present in the JSON body are updated; all other fields are
// left unchanged. Responds 200 OK with the full updated notification.
func (h *notificationHandler) patch(w http.ResponseWriter, r *http.Request, id string) {
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

// delete handles DELETE /notifications/{id} — permanently remove a notification.
// Responds 204 No Content on success, or 404 Not Found if the ID does not exist.
func (h *notificationHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.svc.Delete(r.Context(), id); err != nil {
		handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
