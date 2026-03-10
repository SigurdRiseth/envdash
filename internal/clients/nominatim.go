package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"envdash/internal/firebase"
)

const (
	nominatimCacheTTL = 24 * time.Hour
	nominatimUserAgent = "envdash/1.0 (PROG2005 course project)"
)

// NominatimData holds coordinates returned by Nominatim geocoding.
type NominatimData struct {
	Latitude  float64
	Longitude float64
}

type nominatimResult struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

// NominatimClient fetches geocoding data from the Nominatim OSM API.
// The Nominatim usage policy requires a descriptive User-Agent header
// and limits requests to 1 per second.
type NominatimClient struct {
	baseURL  string
	http     HTTPDoer
	cache    firebase.CacheRepository
	throttle chan struct{}
}

// NewNominatimClient constructs a NominatimClient with a 1 req/sec rate limiter.
func NewNominatimClient(baseURL string, http HTTPDoer, cache firebase.CacheRepository) *NominatimClient {
	throttle := make(chan struct{}, 1)
	throttle <- struct{}{} // initially available

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			select {
			case throttle <- struct{}{}:
			default:
			}
		}
	}()

	return &NominatimClient{baseURL: baseURL, http: http, cache: cache, throttle: throttle}
}

// GetCoordinates returns coordinates for the given ISO country code.
// Results are cached for 24 hours. Rate-limited to 1 req/sec per Nominatim policy.
func (c *NominatimClient) GetCoordinates(ctx context.Context, iso string) (*NominatimData, error) {
	key := "nominatim:" + iso

	if cached, ok, err := c.cache.Get(ctx, key); err == nil && ok {
		var data NominatimData
		if json.Unmarshal(cached, &data) == nil {
			return &data, nil
		}
	}

	// Acquire rate-limit token
	select {
	case <-c.throttle:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	params := url.Values{}
	params.Set("country", iso)
	params.Set("format", "json")
	params.Set("limit", "1")

	reqURL := fmt.Sprintf("%s/search?%s", c.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("nominatim: build request: %w", err)
	}
	req.Header.Set("User-Agent", nominatimUserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nominatim: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nominatim: unexpected status %d", resp.StatusCode)
	}

	var results []nominatimResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("nominatim: decode response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("nominatim: no results for %q", iso)
	}

	lat, err := strconv.ParseFloat(results[0].Lat, 64)
	if err != nil {
		return nil, fmt.Errorf("nominatim: parse latitude: %w", err)
	}
	lon, err := strconv.ParseFloat(results[0].Lon, 64)
	if err != nil {
		return nil, fmt.Errorf("nominatim: parse longitude: %w", err)
	}

	data := &NominatimData{Latitude: lat, Longitude: lon}

	if b, err := json.Marshal(data); err == nil {
		_ = c.cache.Set(ctx, key, b, nominatimCacheTTL)
	}

	return data, nil
}
