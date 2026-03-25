package firebase

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const apiKeysCollection = "api_keys"

// APIKeyRepository defines persistence operations for API keys.
type APIKeyRepository interface {
	Create(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Delete(ctx context.Context, key string) error
}

type apiKeyRepo struct {
	fs         *firestore.Client
	collection string
}

// NewAPIKeyRepo returns a Firestore-backed APIKeyRepository.
func NewAPIKeyRepo(fs *firestore.Client) APIKeyRepository {
	return &apiKeyRepo{fs: fs, collection: apiKeysCollection}
}

func (r *apiKeyRepo) Create(ctx context.Context, key string) error {
	_, err := r.fs.Collection(r.collection).Doc(key).Set(ctx, map[string]interface{}{
		"key":       key,
		"createdAt": time.Now().UTC().Format("20060102 15:04"),
	})
	if err != nil {
		return fmt.Errorf("create api key: %w", err)
	}
	return nil
}

func (r *apiKeyRepo) Exists(ctx context.Context, key string) (bool, error) {
	doc, err := r.fs.Collection(r.collection).Doc(key).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, fmt.Errorf("check api key: %w", err)
	}
	return doc.Exists(), nil
}

func (r *apiKeyRepo) Delete(ctx context.Context, key string) error {
	doc, err := r.fs.Collection(r.collection).Doc(key).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ErrNotFound
		}
		return fmt.Errorf("delete api key: %w", err)
	}
	if !doc.Exists() {
		return ErrNotFound
	}
	if _, err := r.fs.Collection(r.collection).Doc(key).Delete(ctx); err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}
	return nil
}
