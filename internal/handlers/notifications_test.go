package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"envdash/internal/models"
)

func TestNotifications_POST(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid INVOKE webhook",
			body:       `{"url":"http://example.com/hook","country":"NO","event":"INVOKE"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "valid THRESHOLD webhook",
			body:       `{"url":"http://example.com/hook","country":"NO","event":"THRESHOLD","threshold":[{"field":"pm25","operator":">","value":35}]}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "valid compound THRESHOLD webhook",
			body:       `{"url":"http://example.com/hook","country":"NO","event":"THRESHOLD","threshold":[{"field":"temperature","operator":">","value":0},{"field":"temperature","operator":"<=","value":5}]}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing url",
			body:       `{"country":"NO","event":"INVOKE"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid event",
			body:       `{"url":"http://example.com","country":"NO","event":"INVALID"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "THRESHOLD missing threshold body",
			body:       `{"url":"http://example.com","country":"NO","event":"THRESHOLD"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON",
			body:       `{bad`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newMockNotifService()
			router := newTestRouter(newMockRegService(), newMockDashService(), svc, &mockStatusService{})

			req := httptest.NewRequest(http.MethodPost, "/envdash/v1/notifications/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("%s: status = %d, want %d (body: %s)", tt.name, rr.Code, tt.wantStatus, rr.Body.String())
			}

			if tt.wantStatus == http.StatusCreated {
				var resp models.NotificationCreateResponse
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if resp.ID == "" {
					t.Error("expected non-empty ID in response")
				}
			}
		})
	}
}

func TestNotifications_GET_item(t *testing.T) {
	svc := newMockNotifService()
	svc.notifs["n1"] = &models.Notification{ID: "n1", URL: "http://example.com", Event: "INVOKE"}

	router := newTestRouter(newMockRegService(), newMockDashService(), svc, &mockStatusService{})

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{"found", "n1", http.StatusOK},
		{"not found", "missing", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/envdash/v1/notifications/"+tt.id, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestNotifications_GET_list(t *testing.T) {
	svc := newMockNotifService()
	svc.notifs["n1"] = &models.Notification{ID: "n1"}
	svc.notifs["n2"] = &models.Notification{ID: "n2"}

	router := newTestRouter(newMockRegService(), newMockDashService(), svc, &mockStatusService{})

	req := httptest.NewRequest(http.MethodGet, "/envdash/v1/notifications/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	var list []models.Notification
	json.NewDecoder(rr.Body).Decode(&list)
	if len(list) != 2 {
		t.Errorf("got %d notifications, want 2", len(list))
	}
}

func TestNotifications_DELETE(t *testing.T) {
	svc := newMockNotifService()
	svc.notifs["n1"] = &models.Notification{ID: "n1"}

	router := newTestRouter(newMockRegService(), newMockDashService(), svc, &mockStatusService{})

	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{"existing", "n1", http.StatusNoContent},
		{"not found", "missing", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/envdash/v1/notifications/"+tt.id, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestNotifications_PATCH(t *testing.T) {
	svc := newMockNotifService()
	svc.notifs["n1"] = &models.Notification{ID: "n1", URL: "http://old.example.com"}

	router := newTestRouter(newMockRegService(), newMockDashService(), svc, &mockStatusService{})

	req := httptest.NewRequest(http.MethodPatch, "/envdash/v1/notifications/n1",
		bytes.NewBufferString(`{"url":"http://new.example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestNotifications_MethodNotAllowed(t *testing.T) {
	router := newTestRouter(newMockRegService(), newMockDashService(), newMockNotifService(), &mockStatusService{})

	req := httptest.NewRequest(http.MethodPut, "/envdash/v1/notifications/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}
