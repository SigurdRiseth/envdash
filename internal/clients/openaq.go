package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"envdash/internal/firebase"
	"envdash/internal/models"
)

const openaqCacheTTL = 1 * time.Hour

// OpenAQData holds aggregated air quality readings.
type OpenAQData struct {
	PM25  float64
	PM10  float64
	Level string
}

// openaqLocationsResponse is the response from GET /v3/locations.
type openaqLocationsResponse struct {
	Results []struct {
		ID      int     `json:"id"`
		Sensors []struct {
			Parameter struct {
				Name string `json:"name"`
			} `json:"parameter"`
		} `json:"sensors"`
		LatestMeasurements []struct {
			Parameter struct {
				Name string `json:"name"`
			} `json:"parameter"`
			Value float64 `json:"value"`
		} `json:"latestMeasurements,omitempty"`
	} `json:"results"`
}

// OpenAQClient fetches air quality data from the OpenAQ v3 API.
type OpenAQClient struct {
	baseURL  string
	apiKey   string
	http     HTTPDoer
	cache    firebase.CacheRepository
	cacheTTL time.Duration
}

// NewOpenAQClient constructs an OpenAQClient.
func NewOpenAQClient(baseURL, apiKey string, http HTTPDoer, cache firebase.CacheRepository, cacheTTL time.Duration) *OpenAQClient {
	if cacheTTL == 0 {
		cacheTTL = openaqCacheTTL
	}
	return &OpenAQClient{baseURL: baseURL, apiKey: apiKey, http: http, cache: cache, cacheTTL: cacheTTL}
}

// GetAirQuality returns aggregated PM2.5 and PM10 readings for monitoring stations
// within 50 km of the given coordinates. Returns -1 values if no stations are found.
func (c *OpenAQClient) GetAirQuality(ctx context.Context, lat, lon float64) (*OpenAQData, error) {
	key := fmt.Sprintf("openaq:%.4f,%.4f", lat, lon)

	if cached, ok, err := c.cache.Get(ctx, key); err == nil && ok {
		var data OpenAQData
		if json.Unmarshal(cached, &data) == nil {
			return &data, nil
		}
	}

	// Query locations within 50 km and fetch latest measurements
	params := url.Values{}
	params.Set("coordinates", fmt.Sprintf("%f,%f", lat, lon))
	params.Set("radius", "50000")
	params.Set("limit", "100")

	reqURL := fmt.Sprintf("%s/locations?%s", c.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("openaq: build request: %w", err)
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openaq: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openaq: unexpected status %d", resp.StatusCode)
	}

	var raw openaqLocationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("openaq: decode locations response: %w", err)
	}

	if len(raw.Results) == 0 {
		return &OpenAQData{PM25: -1, PM10: -1, Level: "Unknown"}, nil
	}

	// Aggregate latest measurements from all returned locations
	var pm25Vals, pm10Vals []float64
	for _, loc := range raw.Results {
		for _, m := range loc.LatestMeasurements {
			switch m.Parameter.Name {
			case "pm25":
				pm25Vals = append(pm25Vals, m.Value)
			case "pm10":
				pm10Vals = append(pm10Vals, m.Value)
			}
		}
	}

	data := &OpenAQData{
		PM25:  -1,
		PM10:  -1,
		Level: "Unknown",
	}
	if len(pm25Vals) > 0 {
		data.PM25 = mean(pm25Vals)
		data.Level = models.AQILevel(data.PM25)
	}
	if len(pm10Vals) > 0 {
		data.PM10 = mean(pm10Vals)
	}

	if b, err := json.Marshal(data); err == nil {
		_ = c.cache.Set(ctx, key, b, c.cacheTTL)
	}

	return data, nil
}
