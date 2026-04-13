package services

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"envdash/internal/firebase"
	"envdash/internal/models"
)

// NotificationService manages webhook registrations.
type NotificationService interface {
	Create(ctx context.Context, req models.NotificationRequest) (*models.Notification, error)
	Get(ctx context.Context, id string) (*models.Notification, error)
	List(ctx context.Context) ([]models.Notification, error)
	Delete(ctx context.Context, id string) error
	Patch(ctx context.Context, id string, patch map[string]interface{}) (*models.Notification, error)
}

type notificationService struct {
	notifs firebase.NotificationRepository
}

// NewNotificationService constructs a NotificationService.
func NewNotificationService(notifs firebase.NotificationRepository) NotificationService {
	return &notificationService{notifs: notifs}
}

func (s *notificationService) Create(ctx context.Context, req models.NotificationRequest) (*models.Notification, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	n := &models.Notification{
		ID:        generateID(),
		URL:       req.URL,
		Country:   strings.ToUpper(req.Country),
		Event:     req.Event,
		Threshold: req.Threshold,
	}

	if err := s.notifs.Create(ctx, n); err != nil {
		return nil, fmt.Errorf("create notification: %w", err)
	}
	return n, nil
}

func (s *notificationService) Get(ctx context.Context, id string) (*models.Notification, error) {
	return s.notifs.Get(ctx, id)
}

func (s *notificationService) List(ctx context.Context) ([]models.Notification, error) {
	return s.notifs.List(ctx)
}

func (s *notificationService) Delete(ctx context.Context, id string) error {
	return s.notifs.Delete(ctx, id)
}

// Patch applies a partial update to a notification. Only the keys present in
// patch are changed; all other fields remain as stored. Recognised keys are
// "url", "country", "event", and "threshold". Unrecognised keys are silently
// ignored. If the resulting event is THRESHOLD but no threshold is set,
// validation fails with a *models.ValidationError.
func (s *notificationService) Patch(ctx context.Context, id string, patch map[string]interface{}) (*models.Notification, error) {
	existing, err := s.notifs.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if v, ok := patch["url"].(string); ok {
		if _, err := url.ParseRequestURI(v); err != nil {
			return nil, &models.ValidationError{Message: "invalid url"}
		}
		existing.URL = v
	}
	if v, ok := patch["country"].(string); ok {
		existing.Country = strings.ToUpper(v)
	}
	if v, ok := patch["event"].(string); ok {
		if !models.ValidEvents[v] {
			return nil, &models.ValidationError{Message: fmt.Sprintf("invalid event %q", v)}
		}
		existing.Event = v
	}
	if v, ok := patch["threshold"]; ok {
		if v == nil {
			existing.Threshold = nil
		} else if tm, ok := v.(map[string]interface{}); ok {
			t, err := parseThresholdMap(tm)
			if err != nil {
				return nil, err
			}
			existing.Threshold = t
		}
	}

	if existing.Event == models.EventThreshold && existing.Threshold == nil {
		return nil, &models.ValidationError{Message: "'threshold' is required for THRESHOLD event"}
	}

	if err := s.notifs.Create(ctx, existing); err != nil { // Set (overwrite)
		return nil, fmt.Errorf("patch notification: %w", err)
	}
	return existing, nil
}

// parseThresholdMap converts the raw map[string]interface{} representation of a
// threshold (as decoded from a PATCH JSON body) into a *models.Threshold.
// Field and Operator are validated against ValidThresholdFields and
// ValidThresholdOperators; a *models.ValidationError is returned on failure.
// Value defaults to 0 if the key is absent or not a float64.
func parseThresholdMap(m map[string]interface{}) (*models.Threshold, error) {
	t := &models.Threshold{}
	if v, ok := m["field"].(string); ok {
		t.Field = v
	}
	if v, ok := m["operator"].(string); ok {
		t.Operator = v
	}
	if v, ok := m["value"].(float64); ok {
		t.Value = v
	}
	if !models.ValidThresholdFields[t.Field] {
		return nil, &models.ValidationError{Message: fmt.Sprintf("invalid threshold field %q", t.Field)}
	}
	if !models.ValidThresholdOperators[t.Operator] {
		return nil, &models.ValidationError{Message: fmt.Sprintf("invalid threshold operator %q", t.Operator)}
	}
	return t, nil
}
