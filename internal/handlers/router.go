package handlers

import (
	"net/http"
	"strings"

	"envdash/internal/services"
)

const apiPrefix = "/envdash/v1"

// NewRouter registers all routes and returns the root http.Handler.
// If authSvc is non-nil, an X-API-Key middleware is applied to all routes
// except /status/ and /auth/.
func NewRouter(
	regSvc services.RegistrationService,
	dashSvc services.DashboardService,
	notifSvc services.NotificationService,
	statusSvc services.StatusService,
	authSvc services.AuthService,
) http.Handler {
	reg := newRegistrationHandler(regSvc)
	dash := newDashboardHandler(dashSvc)
	notif := newNotificationHandler(notifSvc)
	status := newStatusHandler(statusSvc)
	auth := newAuthHandler(authSvc)

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

	// Auth routes
	mux.HandleFunc(apiPrefix+"/auth/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if isCollectionPath(path, apiPrefix+"/auth/") {
			auth.handleCollection(w, r)
		} else {
			auth.handleItem(w, r)
		}
	})

	if authSvc != nil {
		return apiKeyMiddleware(authSvc, mux)
	}
	return mux
}

// isCollectionPath returns true when the URL refers to the collection root
// (i.e. no ID segment after the prefix).
func isCollectionPath(path, prefix string) bool {
	suffix := strings.TrimPrefix(path, prefix)
	return suffix == "" || suffix == "/"
}
