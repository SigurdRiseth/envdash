package handlers

import (
	"net/http"

	"envdash/internal/services"
)

type statusHandler struct {
	svc services.StatusService
}

// newStatusHandler creates a statusHandler backed by the given StatusService.
func newStatusHandler(svc services.StatusService) *statusHandler {
	return &statusHandler{svc: svc}
}

// handle handles GET /status/ — return the current service status.
// Reports uptime, version, and the reachability of all upstream APIs.
// This endpoint is exempt from API key authentication. Only GET is accepted.
func (h *statusHandler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	status := h.svc.Get(r.Context())
	httpStatus := http.StatusOK
	if status.NotificationDB != http.StatusOK {
		httpStatus = http.StatusInternalServerError
	}
	writeJSON(w, httpStatus, status)
}
