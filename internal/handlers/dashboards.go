package handlers

import (
	"net/http"

	"envdash/internal/services"
)

type dashboardHandler struct {
	svc services.DashboardService
}

// newDashboardHandler creates a dashboardHandler backed by the given DashboardService.
func newDashboardHandler(svc services.DashboardService) *dashboardHandler {
	return &dashboardHandler{svc: svc}
}

// handleItem handles GET /dashboards/{id} — retrieve and return a rendered dashboard.
// Looks up the registration by ID, then fetches live data from all configured
// external APIs concurrently (weather, air quality, currency, etc.) and assembles
// the result. Responds 200 OK with the dashboard payload, or 404 Not Found if
// the registration does not exist. Only GET is accepted.
func (h *dashboardHandler) handleItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := extractID(r.URL.Path, apiPrefix+"/dashboards/")
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
