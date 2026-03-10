//go:build integration

package firebase

import (
	"context"
	"errors"
	"testing"

	"envdash/internal/models"
)

// testRegistrationRepo creates a repo pointing at "test_registrations".
func testRegistrationRepo(t *testing.T) RegistrationRepository {
	t.Helper()
	fs := testClient(t)
	t.Cleanup(func() { dropCollection(t, fs, "test_registrations") })
	return &registrationRepo{fs: fs, collection: "test_registrations"}
}

func TestRegistrationRepo_CreateAndGet(t *testing.T) {
	repo := testRegistrationRepo(t)
	ctx := context.Background()

	reg := &models.Registration{
		ID:         "integ-test-001",
		Country:    "Norway",
		ISOCode:    "NO",
		LastChange: "20250301 09:00",
	}

	if err := repo.Create(ctx, reg); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, reg.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Country != reg.Country {
		t.Errorf("country = %q, want %q", got.Country, reg.Country)
	}
	if got.ISOCode != reg.ISOCode {
		t.Errorf("isoCode = %q, want %q", got.ISOCode, reg.ISOCode)
	}
}

func TestRegistrationRepo_Get_NotFound(t *testing.T) {
	repo := testRegistrationRepo(t)
	ctx := context.Background()

	_, err := repo.Get(ctx, "does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRegistrationRepo_List(t *testing.T) {
	repo := testRegistrationRepo(t)
	ctx := context.Background()

	regs := []*models.Registration{
		{ID: "integ-list-001", Country: "Norway", ISOCode: "NO", LastChange: "20250301 09:00"},
		{ID: "integ-list-002", Country: "Sweden", ISOCode: "SE", LastChange: "20250301 09:00"},
	}
	for _, r := range regs {
		if err := repo.Create(ctx, r); err != nil {
			t.Fatalf("Create %s: %v", r.ID, err)
		}
	}

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) < 2 {
		t.Errorf("list length = %d, want at least 2", len(list))
	}
}

func TestRegistrationRepo_Update(t *testing.T) {
	repo := testRegistrationRepo(t)
	ctx := context.Background()

	reg := &models.Registration{
		ID:         "integ-update-001",
		Country:    "Norway",
		ISOCode:    "NO",
		LastChange: "20250301 09:00",
	}
	if err := repo.Create(ctx, reg); err != nil {
		t.Fatalf("Create: %v", err)
	}

	reg.Country = "Kingdom of Norway"
	reg.LastChange = "20250301 10:00"
	if err := repo.Update(ctx, reg); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.Get(ctx, reg.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Country != "Kingdom of Norway" {
		t.Errorf("country = %q, want %q", got.Country, "Kingdom of Norway")
	}
}

func TestRegistrationRepo_Delete(t *testing.T) {
	repo := testRegistrationRepo(t)
	ctx := context.Background()

	reg := &models.Registration{
		ID:         "integ-delete-001",
		Country:    "Norway",
		ISOCode:    "NO",
		LastChange: "20250301 09:00",
	}
	if err := repo.Create(ctx, reg); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, reg.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.Get(ctx, reg.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestRegistrationRepo_Delete_NotFound(t *testing.T) {
	repo := testRegistrationRepo(t)
	ctx := context.Background()

	err := repo.Delete(ctx, "does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
