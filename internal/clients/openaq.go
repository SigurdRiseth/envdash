package clients

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"envdash/internal/firebase"
	"envdash/internal/models"
)

const (
	openaqCacheTTL   = 1 * time.Hour
	openaqMaxLocations = 5 // cap concurrent /latest calls to limit outbound requests
)

// OpenAQData holds aggregated air quality readings.
type OpenAQData struct {
	PM25  float64 // µg/m³ mean across nearby stations; -1 if no data
	PM10  float64 // µg/m³ mean across nearby stations; -1 if no data
	Level string  // EPA AQI category derived from PM2.5 (e.g. "Good", "Moderate")
}

// openaqLocationsResponse is the response from GET /v3/locations.
// Results are sorted by distance from the query coordinates.
// The sensors array lists which parameters each station measures, including
// the sensor ID needed to look up actual readings via /v3/locations/{id}/latest.
type openaqLocationsResponse struct {
	Results []struct {
		ID      int `json:"id"`
		Sensors []struct {
			ID        int `json:"id"`
			Parameter struct {
				Name string `json:"name"`
			} `json:"parameter"`
		} `json:"sensors"`
	} `json:"results"`
}

// openaqLatestResponse is the response from GET /v3/locations/{id}/latest.
// Each result contains the most recent reading for one sensor at that location.
// SensorsID is cross-referenced against the sensor→parameter map built from
// the locations call to determine which pollutant the value belongs to.
type openaqLatestResponse struct {
	Results []struct {
		SensorsID int     `json:"sensorsId"`
		Value     float64 `json:"value"`
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

// GetAirQuality returns aggregated PM2.5 and PM10 readings for monitoring
// stations within 25 km of the given coordinates. Returns -1 for any pollutant
// with no available data.
//
// The OpenAQ v3 /locations endpoint returns station metadata and sensor IDs
// but not measurement values. A second call to /locations/{id}/latest per
// station retrieves the actual readings. Up to openaqMaxLocations stations
// with PM sensors are queried concurrently to limit outbound request count.
func (c *OpenAQClient) GetAirQuality(ctx context.Context, lat, lon float64) (*OpenAQData, error) {
	key := fmt.Sprintf("openaq:%.4f,%.4f", lat, lon)

	if data, ok := cacheGet[OpenAQData](ctx, c.cache, key); ok {
		return &data, nil
	}

	// Step 1: find monitoring stations within 25 km, sorted by distance.
	// The API returns up to 100 stations; we only need those with PM sensors.
	params := url.Values{}
	params.Set("coordinates", fmt.Sprintf("%f,%f", lat, lon))
	params.Set("radius", "25000")
	params.Set("limit", "100")

	reqURL := fmt.Sprintf("%s/locations?%s", c.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("openaq: build locations request: %w", err)
	}
	req.Header.Set("X-API-Key", c.apiKey)

	var locs openaqLocationsResponse
	if err := fetchJSON(c.http, req, &locs, "openaq"); err != nil {
		return nil, err
	}

	if len(locs.Results) == 0 {
		// No stations found near the coordinates — all values unknown.
		return &OpenAQData{PM25: -1, PM10: -1, Level: "Unknown"}, nil
	}

	// Step 2: build a sensorId → parameterName map and collect location IDs
	// that have at least one PM2.5 or PM10 sensor. Results are already sorted
	// nearest-first, so the first N candidates are the closest stations.
	sensorParam := make(map[int]string)
	var candidateIDs []int
	for _, loc := range locs.Results {
		hasPM := false
		for _, s := range loc.Sensors {
			name := s.Parameter.Name
			if name == "pm25" || name == "pm10" {
				sensorParam[s.ID] = name
				hasPM = true
			}
		}
		if hasPM {
			candidateIDs = append(candidateIDs, loc.ID)
		}
	}

	if len(candidateIDs) == 0 {
		// Stations exist nearby but none measure PM2.5 or PM10.
		return &OpenAQData{PM25: -1, PM10: -1, Level: "Unknown"}, nil
	}

	// Cap to the closest N stations to limit concurrent outbound calls.
	if len(candidateIDs) > openaqMaxLocations {
		candidateIDs = candidateIDs[:openaqMaxLocations]
	}

	// Step 3: concurrently fetch the latest readings for each candidate station.
	type latestResult struct {
		resp openaqLatestResponse
		err  error
	}
	latestResults := make([]latestResult, len(candidateIDs))
	var wg sync.WaitGroup
	for i, locID := range candidateIDs {
		i, locID := i, locID
		wg.Add(1)
		go func() {
			defer wg.Done()
			latestURL := fmt.Sprintf("%s/locations/%d/latest", c.baseURL, locID)
			r, err := http.NewRequestWithContext(ctx, http.MethodGet, latestURL, nil)
			if err != nil {
				latestResults[i].err = fmt.Errorf("openaq: build latest request: %w", err)
				return
			}
			r.Header.Set("X-API-Key", c.apiKey)
			latestResults[i].err = fetchJSON(c.http, r, &latestResults[i].resp, "openaq")
		}()
	}
	wg.Wait()

	// Step 4: aggregate PM2.5 and PM10 values across all stations.
	// Sensor IDs are resolved to parameter names via the map from step 2.
	var pm25Vals, pm10Vals []float64
	for _, lr := range latestResults {
		if lr.err != nil {
			continue // partial failure: skip this station, use the rest
		}
		for _, m := range lr.resp.Results {
			switch sensorParam[m.SensorsID] {
			case "pm25":
				pm25Vals = append(pm25Vals, m.Value)
			case "pm10":
				pm10Vals = append(pm10Vals, m.Value)
			}
		}
	}

	// Default to -1 (unknown) for each pollutant; overwrite only when data exists.
	data := &OpenAQData{PM25: -1, PM10: -1, Level: "Unknown"}
	if len(pm25Vals) > 0 {
		data.PM25 = mean(pm25Vals)
		data.Level = models.AQILevel(data.PM25)
	}
	if len(pm10Vals) > 0 {
		data.PM10 = mean(pm10Vals)
	}

	cacheSet(ctx, c.cache, key, c.cacheTTL, data)

	return data, nil
}
