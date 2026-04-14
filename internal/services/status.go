package services

import (
	"context"
	"net/http"
	"strings"
	"time"

	"envdash/internal/clients"
	"envdash/internal/config"
	"envdash/internal/firebase"
)

// StatusResponse is the response body for GET /status/. Each *API field holds
// the HTTP status code returned by a lightweight probe of that upstream service
// (or 503 if the probe failed). NotificationDB reflects Firestore reachability.
// Webhooks is the total number of registered webhook notifications.
type StatusResponse struct {
	CountriesAPI   int    `json:"countries_api"`
	MeteoAPI       int    `json:"meteo_api"`
	OpenAQAPI      int    `json:"openaq_api"`
	NominatimAPI   int    `json:"nominatim_api"`
	CurrencyAPI    int    `json:"currency_api"`
	NotificationDB int    `json:"notification_db"`
	Webhooks       int    `json:"webhooks"`
	Version        string `json:"version"`
	Uptime         int64  `json:"uptime"` // seconds since service start
}

// StatusService probes upstream services and reports operational metadata.
type StatusService interface {
	Get(ctx context.Context) StatusResponse
}

type statusService struct {
	cfg       *config.Config
	notifs    firebase.NotificationRepository
	http      clients.HTTPDoer
	startTime time.Time
}

// NewStatusService constructs a StatusService. The notification repository is
// used both to count registered webhooks and to verify Firestore reachability.
func NewStatusService(
	cfg *config.Config,
	notifs firebase.NotificationRepository,
	http clients.HTTPDoer,
	startTime time.Time,
) StatusService {
	return &statusService{
		cfg:       cfg,
		notifs:    notifs,
		http:      http,
		startTime: startTime,
	}
}

// Get probes all upstream APIs concurrently and returns a snapshot of
// operational health for the entire service.
func (s *statusService) Get(ctx context.Context) StatusResponse {
	type probeResult struct {
		name string
		code int
	}

	// Each entry pairs a short label with a lightweight probe URL that
	// exercises the real network path without triggering expensive queries.
	probes := []struct {
		name string
		url  string
	}{
		{"countries", s.cfg.CountriesBaseURL + "/alpha/NO?fields=name"},
		{"meteo", s.cfg.MeteoBaseURL + "/forecast?latitude=0&longitude=0&hourly=temperature_2m&forecast_days=1"},
		{"openaq", s.cfg.OpenAQBaseURL + "/providers?limit=1"},
		{"nominatim", s.cfg.NominatimBaseURL + "/search?country=NO&format=json&limit=1"},
		{"currency", s.cfg.CurrencyBaseURL + "/NOK"},
	}

	// Fan out all probes in parallel. The buffered channel ensures goroutines
	// never block on send even if the receiver is slow.
	results := make(map[string]int, len(probes))
	ch := make(chan probeResult, len(probes))

	for _, p := range probes {
		go func(name, url string) {
			ch <- probeResult{name: name, code: s.probe(ctx, url)}
		}(p.name, p.url)
	}

	// Collect exactly len(probes) results before moving on.
	for range probes {
		r := <-ch
		results[r.name] = r.code
	}

	// Count also serves as the Firestore reachability probe — if it errors, the DB is down.
	webhookCount, err := s.notifs.Count(ctx)
	dbStatus := 200
	if err != nil {
		dbStatus = 503
	}

	return StatusResponse{
		CountriesAPI:   results["countries"],
		MeteoAPI:       results["meteo"],
		OpenAQAPI:      results["openaq"],
		NominatimAPI:   results["nominatim"],
		CurrencyAPI:    results["currency"],
		NotificationDB: dbStatus,
		Webhooks:       webhookCount,
		Version:        "v1",
		Uptime:         int64(time.Since(s.startTime).Seconds()),
	}
}

// probe sends a GET request to url and returns the HTTP status code.
// Returns 503 if the request cannot be constructed or the connection fails.
// The User-Agent header is set to identify the probe; the OpenAQ X-API-Key
// header is added automatically when the URL belongs to the OpenAQ base URL.
func (s *statusService) probe(ctx context.Context, url string) int {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 503
	}
	req.Header.Set("User-Agent", "envdash/1.0 (status probe)")

	// OpenAQ requires an API key on every request, even lightweight probes.
	if s.cfg.OpenAQKey != "" && strings.Contains(url, s.cfg.OpenAQBaseURL) {
		req.Header.Set("X-API-Key", s.cfg.OpenAQKey)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		// Network failure or timeout — treat the upstream as unavailable.
		return 503
	}
	defer resp.Body.Close()
	return resp.StatusCode
}
