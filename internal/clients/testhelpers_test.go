package clients_test

import (
	"context"
	"time"
)

// noopCache is a stub CacheRepository that always reports a cache miss and
// discards all writes. Use it in tests that don't need caching behaviour.
type noopCache struct{}

func (n *noopCache) Get(_ context.Context, _ string) ([]byte, bool, error) { return nil, false, nil }
func (n *noopCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error { return nil }
func (n *noopCache) Purge(_ context.Context) (int, error)                             { return 0, nil }

// inMemoryCache is a stub CacheRepository backed by an in-memory map.
// Use it in tests that need to verify cache hit/miss behaviour without
// a real Firestore instance.
type inMemoryCache struct{ data map[string][]byte }

func newInMemoryCache() *inMemoryCache { return &inMemoryCache{data: make(map[string][]byte)} }

func (c *inMemoryCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	v, ok := c.data[key]
	return v, ok, nil
}
func (c *inMemoryCache) Set(_ context.Context, key string, data []byte, _ time.Duration) error {
	c.data[key] = data
	return nil
}
func (c *inMemoryCache) Purge(_ context.Context) (int, error) { return 0, nil }
