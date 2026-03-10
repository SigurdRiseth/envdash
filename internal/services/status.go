package services

import (
	"context"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"

	"envdash/internal/clients"
	"envdash/internal/config"
	"envdash/internal/firebase"
)

// StatusResponse is the response body for GET /status/.
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
	fs        *firestore.Client
	notifs    firebase.NotificationRepository
	http      clients.HTTPDoer
	startTime time.Time
}

// NewStatusService constructs a StatusService.
func NewStatusService(
	cfg *config.Config,
	fs *firestore.Client,
	notifs firebase.NotificationRepository,
	http clients.HTTPDoer,
	startTime time.Time,
) StatusService {
	return &statusService{
		cfg:       cfg,
		fs:        fs,
		notifs:    notifs,
		http:      http,
		startTime: startTime,
	}
}

func (s *statusService) Get(ctx context.Context) StatusResponse {
	type probeResult struct {
		name string
		code int
	}

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

	results := make(map[string]int, len(probes))
	ch := make(chan probeResult, len(probes))

	for _, p := range probes {
		go func(name, url string) {
			ch <- probeResult{name: name, code: s.probe(ctx, url)}
		}(p.name, p.url)
	}
	for range probes {
		r := <-ch
		results[r.name] = r.code
	}

	// Firestore ping
	dbStatus := 200
	if _, err := s.fs.Collection("notifications").Limit(1).Documents(ctx).GetAll(); err != nil {
		dbStatus = 503
	}

	webhookCount, _ := s.notifs.Count(ctx)

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

func (s *statusService) probe(ctx context.Context, url string) int {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 503
	}
	req.Header.Set("User-Agent", "envdash/1.0 (status probe)")

	resp, err := s.http.Do(req)
	if err != nil {
		return 503
	}
	defer resp.Body.Close()
	return resp.StatusCode
}
