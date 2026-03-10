package handlers

import (
	"net/http"

	"envdash/internal/services"
)

type dashboardHandler struct {
	svc services.DashboardService
}

func newDashboardHandler(svc services.DashboardService) *dashboardHandler {
	return &dashboardHandler{svc: svc}
}

func (h *dashboardHandler) handleItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := extractID(r.URL.Path, "/dashboards/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing dashboard ID")
		return
	}

	dashboard, err := h.svc.Get(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dashboard)
}
