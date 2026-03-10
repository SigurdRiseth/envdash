package handlers

import (
	"net/http"
	"strings"

	"envdash/internal/services"
)

const apiPrefix = "/envdash/v1"

// NewRouter registers all routes and returns the root http.Handler.
func NewRouter(
	regSvc services.RegistrationService,
	dashSvc services.DashboardService,
	notifSvc services.NotificationService,
	statusSvc services.StatusService,
) http.Handler {
	reg := newRegistrationHandler(regSvc)
	dash := newDashboardHandler(dashSvc)
	notif := newNotificationHandler(notifSvc)
	status := newStatusHandler(statusSvc)

	mux := http.NewServeMux()

	// Registration routes
	mux.HandleFunc(apiPrefix+"/registrations/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if isCollectionPath(path, apiPrefix+"/registrations/") {
			reg.handleCollection(w, r)
		} else {
			reg.handleItem(w, r)
		}
	})

	// Dashboard routes — item only (no collection endpoint)
	mux.HandleFunc(apiPrefix+"/dashboards/", func(w http.ResponseWriter, r *http.Request) {
		dash.handleItem(w, r)
	})

	// Notification routes
	mux.HandleFunc(apiPrefix+"/notifications/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if isCollectionPath(path, apiPrefix+"/notifications/") {
			notif.handleCollection(w, r)
		} else {
			notif.handleItem(w, r)
		}
	})

	// Status route
	mux.HandleFunc(apiPrefix+"/status/", status.handle)

	return mux
}

// isCollectionPath returns true when the URL refers to the collection root
// (i.e. no ID segment after the prefix).
func isCollectionPath(path, prefix string) bool {
	suffix := strings.TrimPrefix(path, prefix)
	return suffix == "" || suffix == "/"
}
