package clients

import (
	"context"
	"encoding/json"
	"time"

	"envdash/internal/firebase"
)

// mean returns the arithmetic mean of vals, or 0 if vals is empty.
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

// cacheGet attempts to retrieve and unmarshal a cached value for key.
// Returns the value and true on a cache hit; returns the zero value and false
// on a miss, a read error, or an unmarshal failure.
func cacheGet[T any](ctx context.Context, cache firebase.CacheRepository, key string) (T, bool) {
	var zero T
	cached, ok, err := cache.Get(ctx, key)
	if err != nil || !ok {
		return zero, false
	}
	var v T
	if json.Unmarshal(cached, &v) != nil {
		return zero, false
	}
	return v, true
}

// cacheSet marshals v to JSON and writes it to cache under key with the given
// TTL. Errors are silently discarded — caching is best-effort.
func cacheSet(ctx context.Context, cache firebase.CacheRepository, key string, ttl time.Duration, v any) {
	if b, err := json.Marshal(v); err == nil {
		_ = cache.Set(ctx, key, b, ttl)
	}
}
