package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"envdash/internal/firebase"
)

const meteoCacheTTL = 3 * time.Hour

// MeteoData holds the aggregated forecast values from Open-Meteo.
type MeteoData struct {
	Temperature   float64 // mean °C across all hourly entries
	Precipitation float64 // mean mm across all hourly entries
}

type meteoAPIResponse struct {
	Hourly struct {
		Temperature2m []float64 `json:"temperature_2m"`
		Precipitation []float64 `json:"precipitation"`
	} `json:"hourly"`
}

// MeteoClient fetches weather forecast data from the Open-Meteo API.
type MeteoClient struct {
	baseURL  string
	http     HTTPDoer
	cache    firebase.CacheRepository
	cacheTTL time.Duration
}

// NewMeteoClient constructs a MeteoClient.
func NewMeteoClient(baseURL string, http HTTPDoer, cache firebase.CacheRepository, cacheTTL time.Duration) *MeteoClient {
	if cacheTTL == 0 {
		cacheTTL = meteoCacheTTL
	}
	return &MeteoClient{baseURL: baseURL, http: http, cache: cache, cacheTTL: cacheTTL}
}

// GetForecast returns mean temperature and precipitation for the given coordinates.
// Results are cached for 3 hours.
func (c *MeteoClient) GetForecast(ctx context.Context, lat, lon float64) (*MeteoData, error) {
	key := fmt.Sprintf("meteo:%.4f,%.4f", lat, lon)

	if cached, ok, err := c.cache.Get(ctx, key); err == nil && ok {
		var data MeteoData
		if json.Unmarshal(cached, &data) == nil {
			return &data, nil
		}
	}

	params := url.Values{}
	params.Set("latitude", fmt.Sprintf("%f", lat))
	params.Set("longitude", fmt.Sprintf("%f", lon))
	params.Set("hourly", "temperature_2m,precipitation")
	params.Set("forecast_days", "1")

	reqURL := fmt.Sprintf("%s/forecast?%s", c.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("meteo: build request: %w", err)
	}

	var raw meteoAPIResponse
	if err := fetchJSON(c.http, req, &raw, "meteo"); err != nil {
		return nil, err
	}

	data := &MeteoData{
		Temperature:   mean(raw.Hourly.Temperature2m),
		Precipitation: mean(raw.Hourly.Precipitation),
	}

	if b, err := json.Marshal(data); err == nil {
		_ = c.cache.Set(ctx, key, b, c.cacheTTL)
	}

	return data, nil
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}
