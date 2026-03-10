package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"envdash/internal/models"
	"envdash/internal/services"
)

func TestDashboards_GET(t *testing.T) {
	temp := 5.2
	dashSvc := newMockDashService()
	dashSvc.dashboards["abc"] = &models.DashboardResponse{
		Country: "Norway",
		ISOCode: "NO",
		Features: models.DashboardFeatures{
			Temperature: &temp,
		},
		LastRetrieval: "20250301 12:00",
	}

	router := newTestRouter(newMockRegService(), dashSvc, newMockNotifService(), &mockStatusService{})

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{"found", "abc", http.StatusOK},
		{"not found", "nonexistent", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/envdash/v1/dashboards/"+tt.id, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var resp models.DashboardResponse
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if resp.Country != "Norway" {
					t.Errorf("country = %q, want %q", resp.Country, "Norway")
				}
				if resp.Features.Temperature == nil || *resp.Features.Temperature != 5.2 {
					t.Error("temperature mismatch")
				}
			}
		})
	}
}

func TestDashboards_MethodNotAllowed(t *testing.T) {
	router := newTestRouter(newMockRegService(), newMockDashService(), newMockNotifService(), &mockStatusService{})

	req := httptest.NewRequest(http.MethodPost, "/envdash/v1/dashboards/abc", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

func TestDashboards_MissingID(t *testing.T) {
	router := newTestRouter(newMockRegService(), newMockDashService(), newMockNotifService(), &mockStatusService{})

	// The router pattern /dashboards/ with no trailing ID falls through to the handler
	// which should return bad request (no ID extracted).
	// Note: Go's ServeMux routes /dashboards/ to the handler for any /dashboards/... path.
	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/dashboards/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Either 400 (missing ID) or 404 (empty ID treated as not found) is acceptable.
	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 400 or 404", rr.Code)
	}
}

func TestDashboards_LastRetrieval(t *testing.T) {
	dashSvc := newMockDashService()
	dashSvc.dashboards["x"] = &models.DashboardResponse{
		Country:       "Norway",
		ISOCode:       "NO",
		LastRetrieval: "20250301 12:00",
	}

	router := newTestRouter(newMockRegService(), dashSvc, newMockNotifService(), &mockStatusService{})
	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/dashboards/x", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var resp models.DashboardResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.LastRetrieval == "" {
		t.Error("lastRetrieval must not be empty")
	}
}

// Ensure response has correct Content-Type.
func TestDashboards_ContentType(t *testing.T) {
	dashSvc := newMockDashService()
	dashSvc.dashboards["ct"] = &models.DashboardResponse{Country: "Norway", ISOCode: "NO"}

	router := newTestRouter(newMockRegService(), dashSvc, newMockNotifService(), &mockStatusService{})
	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/dashboards/ct", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// TestDashboards_FieldsOmitted verifies that fields with false flags are omitted.
func TestDashboards_FieldsOmitted(t *testing.T) {
	dashSvc := newMockDashService()
	// No features populated — all nil/omitted
	dashSvc.dashboards["y"] = &models.DashboardResponse{
		Country: "Norway",
		ISOCode: "NO",
		Features: models.DashboardFeatures{
			// No fields set — all omitempty should cause them to be absent from JSON
		},
		LastRetrieval: "20250301 12:00",
	}

	router := newTestRouter(newMockRegService(), dashSvc, newMockNotifService(), &mockStatusService{})
	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/dashboards/y", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	var raw map[string]interface{}
	_ = json.NewDecoder(rr.Body).Decode(&raw)

	features, ok := raw["features"].(map[string]interface{})
	if !ok {
		t.Fatal("features field missing from response")
	}

	// temperature, precipitation, airQuality etc. should not be present
	for _, field := range []string{"temperature", "precipitation", "airQuality", "capital"} {
		if _, found := features[field]; found {
			t.Errorf("field %q should be omitted when not populated", field)
		}
	}
}

var _ services.DashboardService = (*mockDashboardService)(nil)
