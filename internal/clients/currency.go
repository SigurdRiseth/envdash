package clients

import (
	"context"
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

// NewCurrencyClient constructs a CurrencyClient. If cacheTTL is 0 it
// defaults to 1 hour.
func NewCurrencyClient(baseURL string, http HTTPDoer, cache firebase.CacheRepository, cacheTTL time.Duration) *CurrencyClient {
	if cacheTTL == 0 {
		cacheTTL = currencyCacheTTL
	}
	return &CurrencyClient{baseURL: baseURL, http: http, cache: cache, cacheTTL: cacheTTL}
}

// GetRates returns exchange rates from base to each currency in targets.
// base and all keys in targets are normalised to upper-case. The full rate
// map for base is cached for 1 hour; only the requested subset is returned.
func (c *CurrencyClient) GetRates(ctx context.Context, base string, targets []string) (map[string]float64, error) {
	base = strings.ToUpper(base)
	key := "currency:" + base

	if allRates, ok := cacheGet[map[string]float64](ctx, c.cache, key); ok {
		return filterRates(allRates, targets), nil
	}

	reqURL := fmt.Sprintf("%s/%s", c.baseURL, base)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("currency: build request: %w", err)
	}

	var wrapper struct {
		Rates map[string]float64 `json:"rates"`
	}
	if err := fetchJSON(c.http, req, &wrapper, "currency"); err != nil {
		return nil, err
	}

	cacheSet(ctx, c.cache, key, c.cacheTTL, wrapper.Rates)

	return filterRates(wrapper.Rates, targets), nil
}

// filterRates returns a subset of all containing only the keys in targets.
// Keys are upper-cased before lookup. Returns nil if targets is empty.
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
