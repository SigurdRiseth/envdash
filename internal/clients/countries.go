package clients

import (
	"context"
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

	if data, ok := cacheGet[CountryData](ctx, c.cache, key); ok {
		return &data, nil
	}

	url := fmt.Sprintf("%s/alpha/%s?fields=name,capital,latlng,population,area,currencies", c.baseURL, iso)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("countries: build request: %w", err)
	}

	var raw countriesAPIResponse
	if err := fetchJSON(c.http, req, &raw, "countries"); err != nil {
		return nil, err
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

	cacheSet(ctx, c.cache, key, c.cacheTTL, data)

	return data, nil
}
