//go:build integration

package firebase

import (
	"context"
	"testing"
	"time"
)

// testCacheRepo creates a repo pointing at "test_cache".
func testCacheRepo(t *testing.T) CacheRepository {
	t.Helper()
	fs := testClient(t)
	t.Cleanup(func() { dropCollection(t, fs, "test_cache") })
	return &cacheRepo{fs: fs, collection: "test_cache"}
}

func TestCacheRepo_SetAndGet(t *testing.T) {
	repo := testCacheRepo(t)
	ctx := context.Background()

	key := "countries:NO"
	data := []byte(`{"name":"Norway"}`)

	if err := repo.Set(ctx, key, data, time.Hour); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, ok, err := repo.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit, got miss")
	}
	if string(got) != string(data) {
		t.Errorf("data = %q, want %q", got, data)
	}
}

func TestCacheRepo_Get_Miss(t *testing.T) {
	repo := testCacheRepo(t)
	ctx := context.Background()

	_, ok, err := repo.Get(ctx, "does-not-exist")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ok {
		t.Error("expected cache miss, got hit")
	}
}

func TestCacheRepo_Get_Expired(t *testing.T) {
	repo := testCacheRepo(t)
	ctx := context.Background()

	key := "expired:key"
	data := []byte(`{"stale":"data"}`)

	// Store with a TTL that has already expired (negative duration).
	if err := repo.Set(ctx, key, data, -time.Second); err != nil {
		t.Fatalf("Set: %v", err)
	}

	_, ok, err := repo.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ok {
		t.Error("expected cache miss for expired entry, got hit")
	}
}

func TestCacheRepo_Purge(t *testing.T) {
	repo := testCacheRepo(t)
	ctx := context.Background()

	if err := repo.Set(ctx, "fresh:key", []byte(`{}`), time.Hour); err != nil {
		t.Fatalf("Set fresh: %v", err)
	}
	if err := repo.Set(ctx, "stale:key", []byte(`{}`), -time.Second); err != nil {
		t.Fatalf("Set stale: %v", err)
	}

	n, err := repo.Purge(ctx)
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if n < 1 {
		t.Errorf("purged %d entries, expected at least 1", n)
	}

	// Fresh entry should survive.
	_, ok, err := repo.Get(ctx, "fresh:key")
	if err != nil {
		t.Fatalf("Get after purge: %v", err)
	}
	if !ok {
		t.Error("fresh entry should survive purge")
	}
}

func TestCacheRepo_KeySanitisation(t *testing.T) {
	repo := testCacheRepo(t)
	ctx := context.Background()

	// Keys with '/' are sanitised to '_' for Firestore document IDs.
	key := "meteo:62.0/10.0"
	data := []byte(`{"temp":5}`)

	if err := repo.Set(ctx, key, data, time.Hour); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, ok, err := repo.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit for sanitised key")
	}
	if string(got) != string(data) {
		t.Errorf("data = %q, want %q", got, data)
	}
}
