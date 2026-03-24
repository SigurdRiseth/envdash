package services_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"envdash/internal/config"
	"envdash/internal/models"
	"envdash/internal/services"
)

// ---- stub notification repo for status tests ----

type stubCountingNotifRepo struct {
	count    int
	countErr error
}

func (r *stubCountingNotifRepo) Create(_ context.Context, _ *models.Notification) error { return nil }
func (r *stubCountingNotifRepo) Get(_ context.Context, _ string) (*models.Notification, error) {
	return nil, nil
}
func (r *stubCountingNotifRepo) List(_ context.Context) ([]models.Notification, error) {
	return nil, nil
}
func (r *stubCountingNotifRepo) Delete(_ context.Context, _ string) error { return nil }
func (r *stubCountingNotifRepo) ListMatching(_ context.Context, _, _ string) ([]models.Notification, error) {
	return nil, nil
}
func (r *stubCountingNotifRepo) Count(_ context.Context) (int, error) {
	return r.count, r.countErr
}

// newStatusProbeServer starts a server that returns the given status code for every request.
func newStatusProbeServer(code int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
	}))
}

// newTestStatusService wires a status service where all 5 probe URLs point to probeServer.
func newTestStatusService(probeServer *httptest.Server, notifRepo *stubCountingNotifRepo, startTime time.Time) services.StatusService {
	cfg := &config.Config{
		CountriesBaseURL: probeServer.URL,
		MeteoBaseURL:     probeServer.URL,
		OpenAQBaseURL:    probeServer.URL,
		NominatimBaseURL: probeServer.URL,
		CurrencyBaseURL:  probeServer.URL,
		OpenAQKey:        "",
	}
	return services.NewStatusService(cfg, notifRepo, http.DefaultClient, startTime)
}

// ---- tests ----

func TestStatusService_Get_AllUp(t *testing.T) {
	srv := newStatusProbeServer(http.StatusOK)
	defer srv.Close()

	notifRepo := &stubCountingNotifRepo{count: 3}
	svc := newTestStatusService(srv, notifRepo, time.Now())

	resp := svc.Get(context.Background())

	if resp.CountriesAPI != 200 {
		t.Errorf("countries_api = %d, want 200", resp.CountriesAPI)
	}
	if resp.MeteoAPI != 200 {
		t.Errorf("meteo_api = %d, want 200", resp.MeteoAPI)
	}
	if resp.OpenAQAPI != 200 {
		t.Errorf("openaq_api = %d, want 200", resp.OpenAQAPI)
	}
	if resp.NominatimAPI != 200 {
		t.Errorf("nominatim_api = %d, want 200", resp.NominatimAPI)
	}
	if resp.CurrencyAPI != 200 {
		t.Errorf("currency_api = %d, want 200", resp.CurrencyAPI)
	}
	if resp.NotificationDB != 200 {
		t.Errorf("notification_db = %d, want 200", resp.NotificationDB)
	}
	if resp.Webhooks != 3 {
		t.Errorf("webhooks = %d, want 3", resp.Webhooks)
	}
	if resp.Version != "v1" {
		t.Errorf("version = %q, want v1", resp.Version)
	}
	if resp.Uptime < 0 {
		t.Error("uptime must be non-negative")
	}
}

func TestStatusService_Get_ProbeReturns503(t *testing.T) {
	srv := newStatusProbeServer(http.StatusServiceUnavailable)
	defer srv.Close()

	svc := newTestStatusService(srv, &stubCountingNotifRepo{count: 0}, time.Now())
	resp := svc.Get(context.Background())

	if resp.CountriesAPI != 503 {
		t.Errorf("countries_api = %d, want 503", resp.CountriesAPI)
	}
}

func TestStatusService_Get_DBDown(t *testing.T) {
	srv := newStatusProbeServer(http.StatusOK)
	defer srv.Close()

	notifRepo := &stubCountingNotifRepo{countErr: errors.New("firestore down")}
	svc := newTestStatusService(srv, notifRepo, time.Now())

	resp := svc.Get(context.Background())

	if resp.NotificationDB != 503 {
		t.Errorf("notification_db = %d, want 503 when repo errors", resp.NotificationDB)
	}
}

func TestStatusService_Get_Uptime(t *testing.T) {
	srv := newStatusProbeServer(http.StatusOK)
	defer srv.Close()

	startTime := time.Now().Add(-5 * time.Second)
	svc := newTestStatusService(srv, &stubCountingNotifRepo{}, startTime)

	resp := svc.Get(context.Background())

	if resp.Uptime < 5 {
		t.Errorf("uptime = %d, want >= 5", resp.Uptime)
	}
}

func TestStatusService_Get_WebhookCount(t *testing.T) {
	srv := newStatusProbeServer(http.StatusOK)
	defer srv.Close()

	svc := newTestStatusService(srv, &stubCountingNotifRepo{count: 7}, time.Now())
	resp := svc.Get(context.Background())

	if resp.Webhooks != 7 {
		t.Errorf("webhooks = %d, want 7", resp.Webhooks)
	}
}

func TestStatusService_Get_Version(t *testing.T) {
	srv := newStatusProbeServer(http.StatusOK)
	defer srv.Close()

	svc := newTestStatusService(srv, &stubCountingNotifRepo{}, time.Now())
	resp := svc.Get(context.Background())

	if resp.Version != "v1" {
		t.Errorf("version = %q, want v1", resp.Version)
	}
}
