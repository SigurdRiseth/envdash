package handlers

import (
	"net/http"
	"strings"

	"envdash/internal/services"
)

const apiPrefix = "/envdash/v1"

// apiKeyExemptPrefixes lists route prefixes that bypass X-API-Key authentication.
// Update this list when adding or removing unauthenticated routes.
var apiKeyExemptPrefixes = []string{
	apiPrefix + "/status/",
	apiPrefix + "/auth/",
}

// NewRouter wires together all service dependencies, registers every API route
// under the /envdash/v1 prefix, and returns the root http.Handler.
// All requests pass through loggingMiddleware. If authSvc is non-nil, an
// apiKeyMiddleware is also applied, protecting all routes except /status/ and /auth/.
// Pass authSvc as nil to disable API key enforcement (e.g. during local development).
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

	var h http.Handler = mux
	if authSvc != nil {
		h = apiKeyMiddleware(authSvc, h)
	}
	return loggingMiddleware(h)
}

// isCollectionPath reports whether path refers to a collection root rather than
// a specific item. It returns true when there is no ID segment after prefix
// (e.g. "/envdash/v1/registrations/" is a collection; "/envdash/v1/registrations/abc" is not).
func isCollectionPath(path, prefix string) bool {
	suffix := strings.TrimPrefix(path, prefix)
	return suffix == "" || suffix == "/"
}
