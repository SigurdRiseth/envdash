package handlers

import (
	"net/http"

	"envdash/internal/services"
)

type statusHandler struct {
	svc services.StatusService
}

func newStatusHandler(svc services.StatusService) *statusHandler {
	return &statusHandler{svc: svc}
}

func (h *statusHandler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	status := h.svc.Get(r.Context())
	writeJSON(w, http.StatusOK, status)
}
