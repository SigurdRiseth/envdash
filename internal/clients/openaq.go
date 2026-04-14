package clients

import (
	"context"
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
	PM25  float64 // µg/m³ mean across nearby stations; -1 if no data
	PM10  float64 // µg/m³ mean across nearby stations; -1 if no data
	Level string  // EPA AQI category derived from PM2.5 (e.g. "Good", "Moderate")
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

// NewOpenAQClient constructs an OpenAQClient. apiKey is sent as the
// X-API-Key header on every request. If cacheTTL is 0 it defaults to 1 hour.
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

	if data, ok := cacheGet[OpenAQData](ctx, c.cache, key); ok {
		return &data, nil
	}

	// Search for monitoring stations within 25 km of the given coordinates.
	// Up to 100 locations are returned; more stations improve the average accuracy.
	params := url.Values{}
	params.Set("coordinates", fmt.Sprintf("%f,%f", lat, lon))
	params.Set("radius", "25000")
	params.Set("limit", "100")

	reqURL := fmt.Sprintf("%s/locations?%s", c.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("openaq: build request: %w", err)
	}
	req.Header.Set("X-API-Key", c.apiKey)

	var raw openaqLocationsResponse
	if err := fetchJSON(c.http, req, &raw, "openaq"); err != nil {
		return nil, err
	}

	// No stations found near the coordinates — report all values as unknown.
	if len(raw.Results) == 0 {
		return &OpenAQData{PM25: -1, PM10: -1, Level: "Unknown"}, nil
	}

	// Collect the latest PM2.5 and PM10 readings from every returned station.
	// Each station may contribute zero or more values of each type.
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

	// Default to -1 (unknown) for each pollutant; overwrite only when we have data.
	data := &OpenAQData{
		PM25:  -1,
		PM10:  -1,
		Level: "Unknown",
	}
	if len(pm25Vals) > 0 {
		data.PM25 = mean(pm25Vals)
		// Derive the EPA AQI category label from the averaged PM2.5 value.
		data.Level = models.AQILevel(data.PM25)
	}
	if len(pm10Vals) > 0 {
		data.PM10 = mean(pm10Vals)
	}

	cacheSet(ctx, c.cache, key, c.cacheTTL, data)

	return data, nil
}
