package handlers

import (
	"net/http"
	"strings"

	"envdash/internal/services"
)

// apiKeyMiddleware validates the X-API-Key header on all routes except
// /status/ and /auth/.
func apiKeyMiddleware(authSvc services.AuthService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, apiPrefix+"/status/") ||
			strings.HasPrefix(path, apiPrefix+"/auth/") {
			next.ServeHTTP(w, r)
			return
		}

		key := r.Header.Get("X-API-Key")
		if key == "" {
			writeError(w, http.StatusUnauthorized, "missing API key")
			return
		}

		valid, err := authSvc.Validate(r.Context(), key)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to validate API key")
			return
		}
		if !valid {
			writeError(w, http.StatusForbidden, "invalid or revoked API key")
			return
		}

		next.ServeHTTP(w, r)
	})
}
