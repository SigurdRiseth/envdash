package services_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"envdash/internal/clients"
	"envdash/internal/firebase"
	"envdash/internal/models"
	"envdash/internal/services"
	"envdash/internal/webhook"
)

// ---- noop cache (for client constructors) ----

type noopCacheRepo struct{}

func (n *noopCacheRepo) Get(_ context.Context, _ string) ([]byte, bool, error) {
	return nil, false, nil
}
func (n *noopCacheRepo) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}
func (n *noopCacheRepo) Purge(_ context.Context) (int, error) { return 0, nil }

// ---- stub notification repo that can return matched notifications ----

type stubMatchingNotifRepo struct {
	matching []models.Notification
	countVal int
}

func (r *stubMatchingNotifRepo) Create(_ context.Context, _ *models.Notification) error { return nil }
func (r *stubMatchingNotifRepo) Get(_ context.Context, _ string) (*models.Notification, error) {
	return nil, firebase.ErrNotFound
}
func (r *stubMatchingNotifRepo) List(_ context.Context) ([]models.Notification, error) { return nil, nil }
func (r *stubMatchingNotifRepo) Delete(_ context.Context, _ string) error              { return nil }
func (r *stubMatchingNotifRepo) ListMatching(_ context.Context, _, _ string) ([]models.Notification, error) {
	return r.matching, nil
}
func (r *stubMatchingNotifRepo) Count(_ context.Context) (int, error) { return r.countVal, nil }

// ---- stub API servers ----

func newCountriesStubServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"name":{"common":"Norway"},"capital":["Oslo"],"latlng":[62.0,10.0],"population":5379475,"area":323802.0,"currencies":{"NOK":{"name":"Norwegian krone","symbol":"kr"}}}`)
	}))
}

func newMeteoStubServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"hourly":{"temperature_2m":[4.0,6.0],"precipitation":[0.1,0.3]}}`)
	}))
}

func newOpenAQStubServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"results":[{"id":1,"sensors":[],"latestMeasurements":[{"parameter":{"name":"pm25"},"value":8.5},{"parameter":{"name":"pm10"},"value":14.2}]}]}`)
	}))
}

func newCurrencyStubServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"EUR":0.087,"USD":0.095}`)
	}))
}

func newErrorServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
}

// ---- dashboard service builder ----

type dashboardTestServers struct {
	countries *httptest.Server
	meteo     *httptest.Server
	openaq    *httptest.Server
	currency  *httptest.Server
}

func newDefaultDashServers() *dashboardTestServers {
	return &dashboardTestServers{
		countries: newCountriesStubServer(),
		meteo:     newMeteoStubServer(),
		openaq:    newOpenAQStubServer(),
		currency:  newCurrencyStubServer(),
	}
}

func (s *dashboardTestServers) close() {
	s.countries.Close()
	s.meteo.Close()
	s.openaq.Close()
	s.currency.Close()
}

func buildDashService(
	regRepo firebase.RegistrationRepository,
	notifRepo firebase.NotificationRepository,
	srvs *dashboardTestServers,
) services.DashboardService {
	cache := &noopCacheRepo{}
	hc := http.DefaultClient
	return services.NewDashboardService(
		regRepo,
		notifRepo,
		clients.NewCountriesClient(srvs.countries.URL, hc, cache, 0),
		clients.NewMeteoClient(srvs.meteo.URL, hc, cache, 0),
		clients.NewOpenAQClient(srvs.openaq.URL, "", hc, cache, 0),
		clients.NewCurrencyClient(srvs.currency.URL, hc, cache, 0),
		webhook.NewDispatcher(&noopHTTPDoer{}),
	)
}

// ---- helper: registration with all features ----

func fullFeaturesReg(id string) *models.Registration {
	return &models.Registration{
		ID:      id,
		Country: "Norway",
		ISOCode: "NO",
		Features: models.Features{
			Temperature:      true,
			Precipitation:    true,
			AirQuality:       true,
			Capital:          true,
			Coordinates:      true,
			Population:       true,
			Area:             true,
			TargetCurrencies: []string{"EUR", "USD"},
		},
	}
}

// ---- tests ----

func TestDashboardService_Get_NotFound(t *testing.T) {
	srvs := newDefaultDashServers()
	defer srvs.close()

	regRepo := newStubRegRepo()
	svc := buildDashService(regRepo, &stubMatchingNotifRepo{}, srvs)

	_, err := svc.Get(context.Background(), "no-such-id")
	if !errors.Is(err, firebase.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDashboardService_Get_AllFeatures(t *testing.T) {
	srvs := newDefaultDashServers()
	defer srvs.close()

	regRepo := newStubRegRepo()
	regRepo.regs["reg1"] = fullFeaturesReg("reg1")
	svc := buildDashService(regRepo, &stubMatchingNotifRepo{}, srvs)

	resp, err := svc.Get(context.Background(), "reg1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if resp.Country != "Norway" {
		t.Errorf("Country = %q, want Norway", resp.Country)
	}
	if resp.ISOCode != "NO" {
		t.Errorf("ISOCode = %q, want NO", resp.ISOCode)
	}
	if resp.LastRetrieval == "" {
		t.Error("LastRetrieval must not be empty")
	}

	// Country-sourced features
	if resp.Features.Capital == nil || *resp.Features.Capital != "Oslo" {
		t.Errorf("Capital = %v, want Oslo", resp.Features.Capital)
	}
	if resp.Features.Coordinates == nil {
		t.Fatal("Coordinates must not be nil")
	}
	if resp.Features.Coordinates.Latitude != 62.0 || resp.Features.Coordinates.Longitude != 10.0 {
		t.Errorf("Coordinates = %+v, want {62, 10}", resp.Features.Coordinates)
	}
	if resp.Features.Population == nil || *resp.Features.Population != 5379475 {
		t.Errorf("Population = %v, want 5379475", resp.Features.Population)
	}
	if resp.Features.Area == nil || *resp.Features.Area != 323802.0 {
		t.Errorf("Area = %v, want 323802", resp.Features.Area)
	}

	// Meteo features — mean of [4.0, 6.0] = 5.0, mean of [0.1, 0.3] = 0.2
	if resp.Features.Temperature == nil {
		t.Fatal("Temperature must not be nil")
	}
	if *resp.Features.Temperature != 5.0 {
		t.Errorf("Temperature = %f, want 5.0", *resp.Features.Temperature)
	}
	if resp.Features.Precipitation == nil {
		t.Fatal("Precipitation must not be nil")
	}
	if *resp.Features.Precipitation != 0.2 {
		t.Errorf("Precipitation = %f, want 0.2", *resp.Features.Precipitation)
	}

	// Air quality
	if resp.Features.AirQuality == nil {
		t.Fatal("AirQuality must not be nil")
	}
	if resp.Features.AirQuality.PM25 != 8.5 {
		t.Errorf("PM25 = %f, want 8.5", resp.Features.AirQuality.PM25)
	}
	if resp.Features.AirQuality.Level != "Good" {
		t.Errorf("Level = %q, want Good (pm25=8.5)", resp.Features.AirQuality.Level)
	}

	// Currency
	if resp.Features.TargetCurrencies == nil {
		t.Fatal("TargetCurrencies must not be nil")
	}
	if resp.Features.TargetCurrencies["EUR"] != 0.087 {
		t.Errorf("EUR = %f, want 0.087", resp.Features.TargetCurrencies["EUR"])
	}
}

func TestDashboardService_Get_FeaturesRespected(t *testing.T) {
	srvs := newDefaultDashServers()
	defer srvs.close()

	regRepo := newStubRegRepo()
	// Only temperature enabled
	regRepo.regs["reg2"] = &models.Registration{
		ID: "reg2", Country: "Norway", ISOCode: "NO",
		Features: models.Features{Temperature: true},
	}
	svc := buildDashService(regRepo, &stubMatchingNotifRepo{}, srvs)

	resp, err := svc.Get(context.Background(), "reg2")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if resp.Features.Temperature == nil {
		t.Error("Temperature should be populated (flag=true)")
	}
	if resp.Features.Precipitation != nil {
		t.Error("Precipitation should be nil (flag=false)")
	}
	if resp.Features.AirQuality != nil {
		t.Error("AirQuality should be nil (flag=false)")
	}
	if resp.Features.Capital != nil {
		t.Error("Capital should be nil (flag=false)")
	}
	if resp.Features.Population != nil {
		t.Error("Population should be nil (flag=false)")
	}
	if resp.Features.TargetCurrencies != nil {
		t.Error("TargetCurrencies should be nil (empty list)")
	}
}

func TestDashboardService_Get_CountriesAPIFails(t *testing.T) {
	srvs := newDefaultDashServers()
	srvs.countries.Close()
	srvs.countries = newErrorServer()
	defer srvs.close()

	regRepo := newStubRegRepo()
	regRepo.regs["reg3"] = fullFeaturesReg("reg3")
	svc := buildDashService(regRepo, &stubMatchingNotifRepo{}, srvs)

	// Dashboard should still return (not error) even when Countries API fails
	resp, err := svc.Get(context.Background(), "reg3")
	if err != nil {
		t.Fatalf("expected partial response, got error: %v", err)
	}

	// Country-sourced features should be absent
	if resp.Features.Capital != nil {
		t.Error("Capital should be nil when Countries API fails")
	}
	if resp.Features.Population != nil {
		t.Error("Population should be nil when Countries API fails")
	}
	// LastRetrieval should still be set
	if resp.LastRetrieval == "" {
		t.Error("LastRetrieval must not be empty even on partial failure")
	}
}

func TestDashboardService_Get_LastRetrievalNotEmpty(t *testing.T) {
	srvs := newDefaultDashServers()
	defer srvs.close()

	regRepo := newStubRegRepo()
	regRepo.regs["reg4"] = &models.Registration{
		ID: "reg4", Country: "Norway", ISOCode: "NO",
	}
	svc := buildDashService(regRepo, &stubMatchingNotifRepo{}, srvs)

	resp, err := svc.Get(context.Background(), "reg4")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.LastRetrieval == "" {
		t.Error("LastRetrieval must be a non-empty timestamp")
	}
}

func TestDashboardService_Get_NoCurrencyWhenNoTargets(t *testing.T) {
	calls := 0
	currencySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"EUR":0.087}`)
	}))
	defer currencySrv.Close()

	srvs := newDefaultDashServers()
	defer srvs.close()
	srvs.currency.Close()
	srvs.currency = currencySrv

	regRepo := newStubRegRepo()
	// No targetCurrencies
	regRepo.regs["reg5"] = &models.Registration{
		ID: "reg5", Country: "Norway", ISOCode: "NO",
		Features: models.Features{TargetCurrencies: nil},
	}
	svc := buildDashService(regRepo, &stubMatchingNotifRepo{}, srvs)

	_, err := svc.Get(context.Background(), "reg5")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if calls != 0 {
		t.Errorf("currency API called %d times, want 0 (no target currencies)", calls)
	}
}
