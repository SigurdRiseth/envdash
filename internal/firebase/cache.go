package firebase

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const cacheCollection = "cache"

// CacheRepository defines operations for reading and writing cached API responses.
type CacheRepository interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, data []byte, ttl time.Duration) error
	// Purge deletes all cache entries whose TTL has expired.
	// Returns the number of deleted entries.
	Purge(ctx context.Context) (int, error)
}

type cacheDoc struct {
	Data      string    `firestore:"data"`
	ExpiresAt time.Time `firestore:"expiresAt"`
}

type cacheRepo struct {
	fs         *firestore.Client
	collection string
}

// NewCacheRepo returns a Firestore-backed CacheRepository.
func NewCacheRepo(fs *firestore.Client) CacheRepository {
	return &cacheRepo{fs: fs, collection: cacheCollection}
}

// Get retrieves a cached value by key. Returns (data, true, nil) on hit,
// (nil, false, nil) on miss or expiry, and (nil, false, err) on error.
func (r *cacheRepo) Get(ctx context.Context, key string) ([]byte, bool, error) {
	doc, err := r.fs.Collection(r.collection).Doc(sanitiseKey(key)).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("cache get %q: %w", key, err)
	}

	var entry cacheDoc
	if err := doc.DataTo(&entry); err != nil {
		return nil, false, fmt.Errorf("cache decode %q: %w", key, err)
	}

	if time.Now().After(entry.ExpiresAt) {
		// Expired — treat as miss; let Purge handle cleanup
		return nil, false, nil
	}

	return []byte(entry.Data), true, nil
}

// Set stores a value in the cache with the given TTL.
func (r *cacheRepo) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	entry := cacheDoc{
		Data:      string(data),
		ExpiresAt: time.Now().Add(ttl),
	}
	_, err := r.fs.Collection(r.collection).Doc(sanitiseKey(key)).Set(ctx, entry)
	if err != nil {
		return fmt.Errorf("cache set %q: %w", key, err)
	}
	return nil
}

// Purge deletes all expired cache entries. Returns the number of deleted entries.
func (r *cacheRepo) Purge(ctx context.Context) (int, error) {
	docs, err := r.fs.Collection(r.collection).
		Where("expiresAt", "<", time.Now()).
		Documents(ctx).GetAll()
	if err != nil {
		return 0, fmt.Errorf("cache purge query: %w", err)
	}

	batch := r.fs.Batch()
	for _, doc := range docs {
		batch.Delete(doc.Ref)
	}

	if len(docs) > 0 {
		if _, err := batch.Commit(ctx); err != nil {
			return 0, fmt.Errorf("cache purge commit: %w", err)
		}
	}

	return len(docs), nil
}

// sanitiseKey replaces characters that are invalid in Firestore document IDs.
// Specifically, '/' is not allowed in document IDs (it is treated as a path separator).
func sanitiseKey(key string) string {
	result := make([]byte, len(key))
	for i := range key {
		if key[i] == '/' {
			result[i] = '_'
		} else {
			result[i] = key[i]
		}
	}
	return string(result)
}
