package handlers

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"envdash/internal/services"
)

// responseWriter wraps http.ResponseWriter to capture the status code for logging.
// The default status is 200 OK, which is used when a handler writes a body
// without explicitly calling WriteHeader.
type responseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader captures the status code before delegating to the underlying ResponseWriter.
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// loggingMiddleware logs method, path, status code, and duration for every request
// as structured fields using the provided logger.
func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration", time.Since(start).String(),
		)
	})
}

// apiKeyMiddleware validates the X-API-Key header on all protected routes.
// Routes listed in apiKeyExemptPrefixes (currently /status/ and /auth/) are
// passed through without authentication. For all other routes, the middleware
// returns 401 Unauthorized if the header is absent, or 403 Forbidden if the
// key is invalid or has been revoked.
func apiKeyMiddleware(authSvc services.AuthService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Allow requests to exempt routes through without any key check. We
		// append "/" before the prefix check so that exact matches (e.g.
		// "/envdash/v1/status") pass the same way as "/envdash/v1/status/".
		for _, prefix := range apiKeyExemptPrefixes {
			if strings.HasPrefix(path+"/", prefix) {
				next.ServeHTTP(w, r)
				return
			}
		}

		key := r.Header.Get("X-API-Key")
		if key == "" {
			writeError(w, http.StatusUnauthorized, "missing API key")
			return
		}

		// Look up the key in Firestore; a missing key returns false without error.
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
