package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"envdash/internal/services"
)

func TestStatus_GET(t *testing.T) {
	statusSvc := &mockStatusService{
		response: services.StatusResponse{
			CountriesAPI:   200,
			MeteoAPI:       200,
			OpenAQAPI:      200,
			NominatimAPI:   200,
			CurrencyAPI:    200,
			NotificationDB: 200,
			Webhooks:       3,
			Version:        "v1",
			Uptime:         120,
		},
	}

	router := newTestRouter(newMockRegService(), newMockDashService(), newMockNotifService(), statusSvc)

	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/status/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}

	var resp services.StatusResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Version != "v1" {
		t.Errorf("version = %q, want v1", resp.Version)
	}
	if resp.CountriesAPI != 200 {
		t.Errorf("countries_api = %d, want 200", resp.CountriesAPI)
	}
	if resp.Webhooks != 3 {
		t.Errorf("webhooks = %d, want 3", resp.Webhooks)
	}
	if resp.Uptime < 0 {
		t.Error("uptime must be non-negative")
	}
}

func TestStatus_MethodNotAllowed(t *testing.T) {
	router := newTestRouter(newMockRegService(), newMockDashService(), newMockNotifService(), &mockStatusService{})

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/status/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

func TestStatus_ContentType(t *testing.T) {
	router := newTestRouter(newMockRegService(), newMockDashService(), newMockNotifService(), &mockStatusService{})

	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/status/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}
