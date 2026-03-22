package services_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"envdash/internal/firebase"
	"envdash/internal/models"
	"envdash/internal/services"
	"envdash/internal/webhook"
)

// ---- stub repos ----

type stubRegRepo struct {
	regs      map[string]*models.Registration
	createErr error
}

func newStubRegRepo() *stubRegRepo {
	return &stubRegRepo{regs: make(map[string]*models.Registration)}
}

func (r *stubRegRepo) Create(_ context.Context, reg *models.Registration) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.regs[reg.ID] = reg
	return nil
}
func (r *stubRegRepo) Get(_ context.Context, id string) (*models.Registration, error) {
	reg, ok := r.regs[id]
	if !ok {
		return nil, firebase.ErrNotFound
	}
	return reg, nil
}
func (r *stubRegRepo) List(_ context.Context) ([]models.Registration, error) {
	out := make([]models.Registration, 0, len(r.regs))
	for _, v := range r.regs {
		out = append(out, *v)
	}
	return out, nil
}
func (r *stubRegRepo) Update(_ context.Context, reg *models.Registration) error {
	r.regs[reg.ID] = reg
	return nil
}
func (r *stubRegRepo) Delete(_ context.Context, id string) error {
	if _, ok := r.regs[id]; !ok {
		return firebase.ErrNotFound
	}
	delete(r.regs, id)
	return nil
}

type stubNotifRepo struct{}

func (r *stubNotifRepo) Create(_ context.Context, _ *models.Notification) error  { return nil }
func (r *stubNotifRepo) Get(_ context.Context, id string) (*models.Notification, error) {
	return nil, firebase.ErrNotFound
}
func (r *stubNotifRepo) List(_ context.Context) ([]models.Notification, error) { return nil, nil }
func (r *stubNotifRepo) Delete(_ context.Context, _ string) error              { return nil }
func (r *stubNotifRepo) ListMatching(_ context.Context, _, _ string) ([]models.Notification, error) {
	return nil, nil
}
func (r *stubNotifRepo) Count(_ context.Context) (int, error) { return 0, nil }

// noopHTTPDoer satisfies clients.HTTPDoer; returns 200 OK for any request.
type noopHTTPDoer struct{}

func (n *noopHTTPDoer) Do(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil
}

func newTestRegService() services.RegistrationService {
	return services.NewRegistrationService(
		newStubRegRepo(),
		&stubNotifRepo{},
		webhook.NewDispatcher(&noopHTTPDoer{}),
	)
}

// ---- tests ----

func TestRegistrationService_Create_MissingCountryAndISO(t *testing.T) {
	svc := newTestRegService()
	_, err := svc.Create(context.Background(), models.RegistrationRequest{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var ve *models.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected *ValidationError, got %T: %v", err, err)
	}
}

func TestRegistrationService_Create_ISOCodeUppercased(t *testing.T) {
	svc := newTestRegService()
	reg, err := svc.Create(context.Background(), models.RegistrationRequest{
		ISOCode: "no",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reg.ISOCode != "NO" {
		t.Errorf("ISOCode = %q, want %q", reg.ISOCode, "NO")
	}
}

func TestRegistrationService_Create_CountryOnly(t *testing.T) {
	svc := newTestRegService()
	reg, err := svc.Create(context.Background(), models.RegistrationRequest{
		Country: "Norway",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reg.Country != "Norway" {
		t.Errorf("Country = %q, want %q", reg.Country, "Norway")
	}
	if reg.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestRegistrationService_Create_IDGenerated(t *testing.T) {
	svc := newTestRegService()
	r1, _ := svc.Create(context.Background(), models.RegistrationRequest{ISOCode: "NO"})
	r2, _ := svc.Create(context.Background(), models.RegistrationRequest{ISOCode: "SE"})
	if r1.ID == r2.ID {
		t.Error("expected unique IDs, got duplicates")
	}
}

func TestRegistrationService_Update_Validation(t *testing.T) {
	svc := newTestRegService()
	// Create a registration first
	reg, _ := svc.Create(context.Background(), models.RegistrationRequest{ISOCode: "NO"})

	// Try to update with no country or isoCode
	_, err := svc.Update(context.Background(), reg.ID, models.RegistrationRequest{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var ve *models.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected *ValidationError, got %T", err)
	}
}

func TestRegistrationService_Update_NotFound(t *testing.T) {
	svc := newTestRegService()
	_, err := svc.Update(context.Background(), "no-such-id", models.RegistrationRequest{ISOCode: "NO"})
	if !errors.Is(err, firebase.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRegistrationService_Patch_AppliesFields(t *testing.T) {
	svc := newTestRegService()
	reg, _ := svc.Create(context.Background(), models.RegistrationRequest{
		ISOCode: "NO",
		Country: "Norway",
	})

	patched, err := svc.Patch(context.Background(), reg.ID, map[string]interface{}{
		"country": "Kingdom of Norway",
	})
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if patched.Country != "Kingdom of Norway" {
		t.Errorf("country = %q, want %q", patched.Country, "Kingdom of Norway")
	}
	// ISOCode should be unchanged
	if patched.ISOCode != "NO" {
		t.Errorf("ISOCode = %q, want %q", patched.ISOCode, "NO")
	}
}

func TestRegistrationService_Patch_ISOCodeUppercased(t *testing.T) {
	svc := newTestRegService()
	reg, _ := svc.Create(context.Background(), models.RegistrationRequest{ISOCode: "NO"})

	patched, err := svc.Patch(context.Background(), reg.ID, map[string]interface{}{
		"isoCode": "se",
	})
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if patched.ISOCode != "SE" {
		t.Errorf("ISOCode = %q, want SE", patched.ISOCode)
	}
}

func TestRegistrationService_Patch_Features(t *testing.T) {
	svc := newTestRegService()
	reg, _ := svc.Create(context.Background(), models.RegistrationRequest{
		ISOCode: "NO",
		Features: models.Features{
			Temperature: true,
			AirQuality:  false,
		},
	})

	patched, err := svc.Patch(context.Background(), reg.ID, map[string]interface{}{
		"features": map[string]interface{}{
			"airQuality": true,
		},
	})
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	// airQuality should be toggled on
	if !patched.Features.AirQuality {
		t.Error("airQuality should be true after patch")
	}
	// temperature should be unchanged
	if !patched.Features.Temperature {
		t.Error("temperature should remain true")
	}
}

func TestRegistrationService_Delete_NotFound(t *testing.T) {
	svc := newTestRegService()
	err := svc.Delete(context.Background(), "no-such-id")
	if !errors.Is(err, firebase.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRegistrationService_Delete_Success(t *testing.T) {
	svc := newTestRegService()
	reg, _ := svc.Create(context.Background(), models.RegistrationRequest{ISOCode: "NO"})

	if err := svc.Delete(context.Background(), reg.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := svc.Get(context.Background(), reg.ID)
	if !errors.Is(err, firebase.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}
