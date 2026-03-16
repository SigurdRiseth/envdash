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

const currencyCacheTTL = 1 * time.Hour

// CurrencyClient fetches exchange rate data from the course currency API.
type CurrencyClient struct {
	baseURL  string
	http     HTTPDoer
	cache    firebase.CacheRepository
	cacheTTL time.Duration
}

// NewCurrencyClient constructs a CurrencyClient.
func NewCurrencyClient(baseURL string, http HTTPDoer, cache firebase.CacheRepository, cacheTTL time.Duration) *CurrencyClient {
	if cacheTTL == 0 {
		cacheTTL = currencyCacheTTL
	}
	return &CurrencyClient{baseURL: baseURL, http: http, cache: cache, cacheTTL: cacheTTL}
}

// GetRates returns exchange rates from the given base currency to each target currency.
// Results are cached for 1 hour.
func (c *CurrencyClient) GetRates(ctx context.Context, base string, targets []string) (map[string]float64, error) {
	base = strings.ToUpper(base)
	key := "currency:" + base

	if cached, ok, err := c.cache.Get(ctx, key); err == nil && ok {
		var allRates map[string]float64
		if json.Unmarshal(cached, &allRates) == nil {
			return filterRates(allRates, targets), nil
		}
	}

	reqURL := fmt.Sprintf("%s/%s", c.baseURL, base)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("currency: build request: %w", err)
	}

	var allRates map[string]float64
	if err := fetchJSON(c.http, req, &allRates, "currency"); err != nil {
		return nil, err
	}

	if b, err := json.Marshal(allRates); err == nil {
		_ = c.cache.Set(ctx, key, b, c.cacheTTL)
	}

	return filterRates(allRates, targets), nil
}

func filterRates(all map[string]float64, targets []string) map[string]float64 {
	if len(targets) == 0 {
		return nil
	}
	result := make(map[string]float64, len(targets))
	for _, t := range targets {
		t = strings.ToUpper(t)
		if rate, ok := all[t]; ok {
			result[t] = rate
		}
	}
	return result
}
