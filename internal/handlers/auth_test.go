package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"envdash/internal/handlers"
)

// newTestRouterWithAuth builds a router with auth middleware enabled.
func newTestRouterWithAuth(authSvc *mockAuthService) http.Handler {
	return handlers.NewRouter(
		newMockRegService(),
		newMockDashService(),
		newMockNotifService(),
		&mockStatusService{},
		authSvc,
	)
}

// ---- POST /auth/ ----

func TestAuth_POST_CreatesKey(t *testing.T) {
	authSvc := newMockAuthService()
	router := newTestRouterWithAuth(authSvc)

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/auth/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rr.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if key := body["key"]; key == "" {
		t.Error("expected non-empty key in response")
	}
	if body["createdAt"] == "" {
		t.Error("expected non-empty createdAt in response")
	}
}

func TestAuth_POST_MethodNotAllowed(t *testing.T) {
	router := newTestRouterWithAuth(newMockAuthService())

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/envdash/v1/auth/", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /auth/ status = %d, want 405", method, rr.Code)
		}
	}
}

func TestAuth_POST_InternalError(t *testing.T) {
	authSvc := newMockAuthService()
	authSvc.createErr = errInternal
	router := newTestRouterWithAuth(authSvc)

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/auth/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rr.Code)
	}
}

// ---- DELETE /auth/{key} ----

func TestAuth_DELETE_RevokesKey(t *testing.T) {
	authSvc := newMockAuthService()
	authSvc.keys["sk-envdash-testkey1234567890ab"] = true
	router := newTestRouterWithAuth(authSvc)

	req := httptest.NewRequest(http.MethodDelete, "/envdash/v1/auth/sk-envdash-testkey1234567890ab", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rr.Code)
	}
}

func TestAuth_DELETE_NotFound(t *testing.T) {
	router := newTestRouterWithAuth(newMockAuthService())

	req := httptest.NewRequest(http.MethodDelete, "/envdash/v1/auth/sk-envdash-doesnotexist0000", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestAuth_DELETE_MethodNotAllowed(t *testing.T) {
	router := newTestRouterWithAuth(newMockAuthService())

	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/auth/somekey", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

// ---- Middleware ----

func TestMiddleware_MissingKey_Returns401(t *testing.T) {
	authSvc := newMockAuthService()
	router := newTestRouterWithAuth(authSvc)

	// Request to a protected route without API key
	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/registrations/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

func TestMiddleware_InvalidKey_Returns403(t *testing.T) {
	authSvc := newMockAuthService()
	router := newTestRouterWithAuth(authSvc)

	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/registrations/", nil)
	req.Header.Set("X-API-Key", "sk-envdash-notvalid00000000000000")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rr.Code)
	}
}

func TestMiddleware_ValidKey_Passes(t *testing.T) {
	authSvc := newMockAuthService()
	authSvc.keys["sk-envdash-validkey123456789012"] = true
	router := newTestRouterWithAuth(authSvc)

	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/registrations/", nil)
	req.Header.Set("X-API-Key", "sk-envdash-validkey123456789012")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Should reach the handler (200 OK for empty list, not 401/403)
	if rr.Code == http.StatusUnauthorized || rr.Code == http.StatusForbidden {
		t.Errorf("status = %d, want non-auth error", rr.Code)
	}
}

func TestMiddleware_StatusExempt(t *testing.T) {
	authSvc := newMockAuthService()
	router := newTestRouterWithAuth(authSvc)

	// /status/ is exempt — no API key required
	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/status/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code == http.StatusUnauthorized || rr.Code == http.StatusForbidden {
		t.Errorf("/status/ should be exempt from auth, got %d", rr.Code)
	}
}

func TestMiddleware_AuthRouteExempt(t *testing.T) {
	authSvc := newMockAuthService()
	router := newTestRouterWithAuth(authSvc)

	// POST /auth/ is exempt — no API key required
	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/auth/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code == http.StatusUnauthorized || rr.Code == http.StatusForbidden {
		t.Errorf("POST /auth/ should be exempt from auth, got %d", rr.Code)
	}
}

func TestMiddleware_ValidateError_Returns500(t *testing.T) {
	authSvc := newMockAuthService()
	authSvc.existsErr = errInternal
	router := newTestRouterWithAuth(authSvc)

	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/registrations/", nil)
	req.Header.Set("X-API-Key", "sk-envdash-anykey000000000000000")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rr.Code)
	}
}
