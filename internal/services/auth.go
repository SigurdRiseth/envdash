package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"envdash/internal/firebase"
)

// AuthService manages API key lifecycle.
type AuthService interface {
	Register(ctx context.Context) (string, error)
	Revoke(ctx context.Context, key string) error
	Validate(ctx context.Context, key string) (bool, error)
}

type authService struct {
	keys firebase.APIKeyRepository
}

// NewAuthService constructs an AuthService backed by Firestore.
func NewAuthService(keys firebase.APIKeyRepository) AuthService {
	return &authService{keys: keys}
}

func (s *authService) Register(ctx context.Context) (string, error) {
	key := generateAPIKey()
	if err := s.keys.Create(ctx, key); err != nil {
		return "", fmt.Errorf("register api key: %w", err)
	}
	return key, nil
}

func (s *authService) Revoke(ctx context.Context, key string) error {
	return s.keys.Delete(ctx, key)
}

func (s *authService) Validate(ctx context.Context, key string) (bool, error) {
	return s.keys.Exists(ctx, key)
}

// generateAPIKey returns a key in the format sk-envdash-{32 hex chars}.
func generateAPIKey() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand unavailable: %v", err))
	}
	return "sk-envdash-" + hex.EncodeToString(b)
}
