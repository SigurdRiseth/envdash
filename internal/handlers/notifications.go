package handlers

import (
	"net/http"

	"envdash/internal/models"
	"envdash/internal/services"
)

type notificationHandler struct {
	svc services.NotificationService
}

func newNotificationHandler(svc services.NotificationService) *notificationHandler {
	return &notificationHandler{svc: svc}
}

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

func (h *notificationHandler) list(w http.ResponseWriter, r *http.Request) {
	ns, err := h.svc.List(r.Context())
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ns)
}

func (h *notificationHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	n, err := h.svc.Get(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (h *notificationHandler) patch(w http.ResponseWriter, r *http.Request, id string) {
	var patch map[string]interface{}
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

func (h *notificationHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.svc.Delete(r.Context(), id); err != nil {
		handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
