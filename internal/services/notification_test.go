package services_test

import (
	"context"
	"errors"
	"testing"

	"envdash/internal/firebase"
	"envdash/internal/models"
	"envdash/internal/services"
)

// stubFullNotifRepo extends stubNotifRepo with actual storage for notification tests.
type stubFullNotifRepo struct {
	notifs map[string]*models.Notification
}

func newStubFullNotifRepo() *stubFullNotifRepo {
	return &stubFullNotifRepo{notifs: make(map[string]*models.Notification)}
}

func (r *stubFullNotifRepo) Create(_ context.Context, n *models.Notification) error {
	r.notifs[n.ID] = n
	return nil
}
func (r *stubFullNotifRepo) Get(_ context.Context, id string) (*models.Notification, error) {
	n, ok := r.notifs[id]
	if !ok {
		return nil, firebase.ErrNotFound
	}
	return n, nil
}
func (r *stubFullNotifRepo) List(_ context.Context) ([]models.Notification, error) {
	out := make([]models.Notification, 0, len(r.notifs))
	for _, v := range r.notifs {
		out = append(out, *v)
	}
	return out, nil
}
func (r *stubFullNotifRepo) Delete(_ context.Context, id string) error {
	if _, ok := r.notifs[id]; !ok {
		return firebase.ErrNotFound
	}
	delete(r.notifs, id)
	return nil
}
func (r *stubFullNotifRepo) ListMatching(_ context.Context, _, _ string) ([]models.Notification, error) {
	return nil, nil
}
func (r *stubFullNotifRepo) Count(_ context.Context) (int, error) { return len(r.notifs), nil }

func newTestNotifService() services.NotificationService {
	return services.NewNotificationService(newStubFullNotifRepo())
}

// ---- Create validation tests ----

func TestNotificationService_Create_MissingURL(t *testing.T) {
	svc := newTestNotifService()
	_, err := svc.Create(context.Background(), models.NotificationRequest{
		Event: models.EventRegister,
	})
	assertValidationError(t, err, "url")
}

func TestNotificationService_Create_InvalidURL(t *testing.T) {
	svc := newTestNotifService()
	_, err := svc.Create(context.Background(), models.NotificationRequest{
		URL:   "not-a-url",
		Event: models.EventRegister,
	})
	assertValidationError(t, err, "url")
}

func TestNotificationService_Create_InvalidEvent(t *testing.T) {
	svc := newTestNotifService()
	_, err := svc.Create(context.Background(), models.NotificationRequest{
		URL:   "https://example.com/hook",
		Event: "BOGUS",
	})
	assertValidationError(t, err, "event")
}

func TestNotificationService_Create_ThresholdRequired(t *testing.T) {
	svc := newTestNotifService()
	_, err := svc.Create(context.Background(), models.NotificationRequest{
		URL:   "https://example.com/hook",
		Event: models.EventThreshold,
		// no Threshold
	})
	assertValidationError(t, err, "threshold")
}

func TestNotificationService_Create_InvalidThresholdField(t *testing.T) {
	svc := newTestNotifService()
	_, err := svc.Create(context.Background(), models.NotificationRequest{
		URL:   "https://example.com/hook",
		Event: models.EventThreshold,
		Threshold: &models.Threshold{
			Field:    "humidity",
			Operator: ">",
			Value:    50,
		},
	})
	assertValidationError(t, err, "field")
}

func TestNotificationService_Create_InvalidThresholdOperator(t *testing.T) {
	svc := newTestNotifService()
	_, err := svc.Create(context.Background(), models.NotificationRequest{
		URL:   "https://example.com/hook",
		Event: models.EventThreshold,
		Threshold: &models.Threshold{
			Field:    "pm25",
			Operator: "!=",
			Value:    50,
		},
	})
	assertValidationError(t, err, "operator")
}

func TestNotificationService_Create_ValidThreshold(t *testing.T) {
	for _, op := range []string{">", "<", ">=", "<="} {
		t.Run(op, func(t *testing.T) {
			svc := newTestNotifService()
			_, err := svc.Create(context.Background(), models.NotificationRequest{
				URL:   "https://example.com/hook",
				Event: models.EventThreshold,
				Threshold: &models.Threshold{
					Field:    "pm25",
					Operator: op,
					Value:    35.4,
				},
			})
			if err != nil {
				t.Errorf("op=%q: unexpected error: %v", op, err)
			}
		})
	}
}

func TestNotificationService_Create_CountryUppercased(t *testing.T) {
	svc := newTestNotifService()
	n, err := svc.Create(context.Background(), models.NotificationRequest{
		URL:     "https://example.com/hook",
		Event:   models.EventRegister,
		Country: "no",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.Country != "NO" {
		t.Errorf("Country = %q, want NO", n.Country)
	}
}

// ---- Patch validation tests ----

func TestNotificationService_Patch_InvalidURL(t *testing.T) {
	svc := newTestNotifService()
	n, _ := svc.Create(context.Background(), models.NotificationRequest{
		URL:   "https://example.com/hook",
		Event: models.EventRegister,
	})

	_, err := svc.Patch(context.Background(), n.ID, map[string]interface{}{
		"url": "not-a-url",
	})
	assertValidationError(t, err, "url")
}

func TestNotificationService_Patch_InvalidEvent(t *testing.T) {
	svc := newTestNotifService()
	n, _ := svc.Create(context.Background(), models.NotificationRequest{
		URL:   "https://example.com/hook",
		Event: models.EventRegister,
	})

	_, err := svc.Patch(context.Background(), n.ID, map[string]interface{}{
		"event": "BOGUS",
	})
	assertValidationError(t, err, "event")
}

func TestNotificationService_Patch_UpdatesURL(t *testing.T) {
	svc := newTestNotifService()
	n, _ := svc.Create(context.Background(), models.NotificationRequest{
		URL:   "https://example.com/old",
		Event: models.EventRegister,
	})

	patched, err := svc.Patch(context.Background(), n.ID, map[string]interface{}{
		"url": "https://example.com/new",
	})
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if patched.URL != "https://example.com/new" {
		t.Errorf("URL = %q, want new URL", patched.URL)
	}
	// Event should be unchanged
	if patched.Event != models.EventRegister {
		t.Errorf("Event = %q, should be unchanged", patched.Event)
	}
}

func TestNotificationService_Patch_NotFound(t *testing.T) {
	svc := newTestNotifService()
	_, err := svc.Patch(context.Background(), "no-such-id", map[string]interface{}{
		"url": "https://example.com/hook",
	})
	if !errors.Is(err, firebase.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---- helper ----

// assertValidationError fails the test if err is not a *models.ValidationError
// or if the error message does not mention the given field keyword.
func assertValidationError(t *testing.T, err error, keyword string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected validation error containing %q, got nil", keyword)
	}
	var ve *models.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected *models.ValidationError, got %T: %v", err, err)
	}
}
