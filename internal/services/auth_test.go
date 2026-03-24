package services_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"envdash/internal/firebase"
	"envdash/internal/services"
)

// ---- stub APIKeyRepository ----

type stubAPIKeyRepo struct {
	keys      map[string]bool
	createErr error
	deleteErr error
}

func newStubAPIKeyRepo() *stubAPIKeyRepo {
	return &stubAPIKeyRepo{keys: make(map[string]bool)}
}

func (r *stubAPIKeyRepo) Create(_ context.Context, key string) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.keys[key] = true
	return nil
}

func (r *stubAPIKeyRepo) Exists(_ context.Context, key string) (bool, error) {
	return r.keys[key], nil
}

func (r *stubAPIKeyRepo) Delete(_ context.Context, key string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	if !r.keys[key] {
		return firebase.ErrNotFound
	}
	delete(r.keys, key)
	return nil
}

// ---- helpers ----

func newTestAuthService() (services.AuthService, *stubAPIKeyRepo) {
	repo := newStubAPIKeyRepo()
	return services.NewAuthService(repo), repo
}

// ---- tests ----

func TestAuthService_Register_ReturnsKeyWithPrefix(t *testing.T) {
	svc, _ := newTestAuthService()
	key, err := svc.Register(context.Background())
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if !strings.HasPrefix(key, "sk-envdash-") {
		t.Errorf("key %q does not start with sk-envdash-", key)
	}
}

func TestAuthService_Register_KeyStoredInRepo(t *testing.T) {
	svc, repo := newTestAuthService()
	key, err := svc.Register(context.Background())
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if !repo.keys[key] {
		t.Errorf("key %q not stored in repository", key)
	}
}

func TestAuthService_Register_UniqueKeys(t *testing.T) {
	svc, _ := newTestAuthService()
	k1, _ := svc.Register(context.Background())
	k2, _ := svc.Register(context.Background())
	if k1 == k2 {
		t.Error("expected unique keys, got duplicates")
	}
}

func TestAuthService_Register_PropagatesRepoError(t *testing.T) {
	repo := newStubAPIKeyRepo()
	repo.createErr = errors.New("firestore unavailable")
	svc := services.NewAuthService(repo)

	_, err := svc.Register(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAuthService_Validate_ExistingKey(t *testing.T) {
	svc, repo := newTestAuthService()
	repo.keys["sk-envdash-abc123"] = true

	valid, err := svc.Validate(context.Background(), "sk-envdash-abc123")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !valid {
		t.Error("expected valid=true for existing key")
	}
}

func TestAuthService_Validate_MissingKey(t *testing.T) {
	svc, _ := newTestAuthService()

	valid, err := svc.Validate(context.Background(), "sk-envdash-doesnotexist")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if valid {
		t.Error("expected valid=false for missing key")
	}
}

func TestAuthService_Revoke_Success(t *testing.T) {
	svc, repo := newTestAuthService()
	key, _ := svc.Register(context.Background())

	if err := svc.Revoke(context.Background(), key); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if repo.keys[key] {
		t.Error("key should be removed from repo after revoke")
	}
}

func TestAuthService_Revoke_NotFound(t *testing.T) {
	svc, _ := newTestAuthService()

	err := svc.Revoke(context.Background(), "sk-envdash-doesnotexist")
	if !errors.Is(err, firebase.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestAuthService_Validate_AfterRevoke(t *testing.T) {
	svc, _ := newTestAuthService()
	key, _ := svc.Register(context.Background())
	_ = svc.Revoke(context.Background(), key)

	valid, err := svc.Validate(context.Background(), key)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if valid {
		t.Error("key should be invalid after revoke")
	}
}
