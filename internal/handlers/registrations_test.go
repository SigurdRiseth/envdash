package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"envdash/internal/handlers"
	"envdash/internal/models"
	"envdash/internal/services"
)

func newTestRouter(
	regSvc services.RegistrationService,
	dashSvc services.DashboardService,
	notifSvc services.NotificationService,
	statusSvc services.StatusService,
) http.Handler {
	return handlers.NewRouter(regSvc, dashSvc, notifSvc, statusSvc)
}

func TestRegistrations_POST(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid request",
			body:       `{"country":"Norway","isoCode":"NO","features":{"temperature":true}}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing country and isoCode",
			body:       `{"features":{"temperature":true}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON",
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newMockRegService()
			router := newTestRouter(svc, newMockDashService(), newMockNotifService(), &mockStatusService{})

			req := httptest.NewRequest(http.MethodPost, "/envdash/v1/registrations/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", rr.Code, tt.wantStatus, rr.Body.String())
			}

			if tt.wantStatus == http.StatusCreated {
				var resp models.RegistrationCreateResponse
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if resp.ID == "" {
					t.Error("expected non-empty ID in response")
				}
			}
		})
	}
}

func TestRegistrations_GET_item(t *testing.T) {
	svc := newMockRegService()
	// Pre-populate a registration
	reg := &models.Registration{ID: "abc123", Country: "Norway", ISOCode: "NO"}
	svc.regs["abc123"] = reg

	router := newTestRouter(svc, newMockDashService(), newMockNotifService(), &mockStatusService{})

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{"found", "abc123", http.StatusOK},
		{"not found", "nonexistent", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/envdash/v1/registrations/"+tt.id, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestRegistrations_GET_list(t *testing.T) {
	svc := newMockRegService()
	svc.regs["a"] = &models.Registration{ID: "a", Country: "Norway"}
	svc.regs["b"] = &models.Registration{ID: "b", Country: "Sweden"}

	router := newTestRouter(svc, newMockDashService(), newMockNotifService(), &mockStatusService{})

	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/registrations/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}

	var list []models.Registration
	if err := json.NewDecoder(rr.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d registrations, want 2", len(list))
	}
}

func TestRegistrations_HEAD(t *testing.T) {
	svc := newMockRegService()
	router := newTestRouter(svc, newMockDashService(), newMockNotifService(), &mockStatusService{})

	req := httptest.NewRequest(http.MethodHead, "/envdash/v1/registrations/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Error("HEAD response must have no body")
	}
}

func TestRegistrations_PUT(t *testing.T) {
	svc := newMockRegService()
	svc.regs["abc"] = &models.Registration{ID: "abc", Country: "Norway"}

	router := newTestRouter(svc, newMockDashService(), newMockNotifService(), &mockStatusService{})

	tests := []struct {
		name       string
		id         string
		body       string
		wantStatus int
	}{
		{
			name:       "valid update",
			id:         "abc",
			body:       `{"country":"Sweden","isoCode":"SE","features":{}}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found",
			id:         "missing",
			body:       `{"country":"Sweden","isoCode":"SE","features":{}}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid JSON",
			id:         "abc",
			body:       `{bad}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/envdash/v1/registrations/"+tt.id, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestRegistrations_PATCH(t *testing.T) {
	svc := newMockRegService()
	svc.regs["abc"] = &models.Registration{ID: "abc", Country: "Norway"}

	router := newTestRouter(svc, newMockDashService(), newMockNotifService(), &mockStatusService{})

	req := httptest.NewRequest(http.MethodPatch, "/envdash/v1/registrations/abc",
		bytes.NewBufferString(`{"country":"Sweden"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestRegistrations_DELETE(t *testing.T) {
	svc := newMockRegService()
	svc.regs["abc"] = &models.Registration{ID: "abc"}

	router := newTestRouter(svc, newMockDashService(), newMockNotifService(), &mockStatusService{})

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{"existing", "abc", http.StatusNoContent},
		{"not found", "missing", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/envdash/v1/registrations/"+tt.id, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestRegistrations_MethodNotAllowed(t *testing.T) {
	router := newTestRouter(newMockRegService(), newMockDashService(), newMockNotifService(), &mockStatusService{})

	req := httptest.NewRequest(http.MethodOptions, "/envdash/v1/registrations/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}
