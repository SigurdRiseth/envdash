package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"envdash/internal/firebase"
)

const countriesCacheTTL = 24 * time.Hour

// CountryData holds the fields extracted from the REST Countries API.
type CountryData struct {
	Name         string
	Capital      string
	Latitude     float64
	Longitude    float64
	Population   int64
	Area         float64
	BaseCurrency string // ISO 4217 code, e.g. "NOK"
}

// countriesAPIResponse maps the fields we need from the REST Countries API.
type countriesAPIResponse struct {
	Name struct {
		Common string `json:"common"`
	} `json:"name"`
	Capital  []string    `json:"capital"`
	Latlng   []float64   `json:"latlng"`
	Population int64     `json:"population"`
	Area       float64   `json:"area"`
	Currencies map[string]struct {
		Name   string `json:"name"`
		Symbol string `json:"symbol"`
	} `json:"currencies"`
}

// CountriesClient fetches country data from the REST Countries API.
type CountriesClient struct {
	baseURL  string
	http     HTTPDoer
	cache    firebase.CacheRepository
	cacheTTL time.Duration
}

// NewCountriesClient constructs a CountriesClient.
func NewCountriesClient(baseURL string, http HTTPDoer, cache firebase.CacheRepository, cacheTTL time.Duration) *CountriesClient {
	if cacheTTL == 0 {
		cacheTTL = countriesCacheTTL
	}
	return &CountriesClient{baseURL: baseURL, http: http, cache: cache, cacheTTL: cacheTTL}
}

// GetByISO fetches country data for the given ISO 3166-1 alpha-2 code.
// Results are cached in Firestore for 24 hours.
func (c *CountriesClient) GetByISO(ctx context.Context, iso string) (*CountryData, error) {
	iso = strings.ToUpper(iso)
	key := "countries:" + iso

	if cached, ok, err := c.cache.Get(ctx, key); err == nil && ok {
		var data CountryData
		if json.Unmarshal(cached, &data) == nil {
			return &data, nil
		}
	}

	url := fmt.Sprintf("%s/alpha/%s?fields=name,capital,latlng,population,area,currencies", c.baseURL, iso)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("countries: build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("countries: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("countries: country %q not found", iso)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("countries: unexpected status %d", resp.StatusCode)
	}

	var raw countriesAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("countries: decode response: %w", err)
	}

	data := &CountryData{
		Name:       raw.Name.Common,
		Population: raw.Population,
		Area:       raw.Area,
	}

	if len(raw.Capital) > 0 {
		data.Capital = raw.Capital[0]
	}
	if len(raw.Latlng) >= 2 {
		data.Latitude = raw.Latlng[0]
		data.Longitude = raw.Latlng[1]
	}
	for code := range raw.Currencies {
		data.BaseCurrency = code
		break // use first (only) currency
	}

	if b, err := json.Marshal(data); err == nil {
		_ = c.cache.Set(ctx, key, b, c.cacheTTL)
	}

	return data, nil
}
