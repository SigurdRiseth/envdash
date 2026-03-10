package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"envdash/internal/services"
)

// responseWriter wraps http.ResponseWriter to capture the status code for logging.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// loggingMiddleware logs method, path, status code, and duration for every request.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

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
