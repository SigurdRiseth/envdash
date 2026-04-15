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
	// Reject invalid requests (missing URL, unknown event, missing/invalid threshold conditions).
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Build the notification document. Country is upper-cased so ListMatching
	// comparisons are case-insensitive; an empty Country matches all countries.
	notif := &models.Notification{
		ID:         generateID(),
		URL:        req.URL,
		Country:    strings.ToUpper(req.Country),
		Event:      req.Event,
		Thresholds: req.Thresholds,
	}

	if err := s.notifs.Create(ctx, notif); err != nil {
		return nil, fmt.Errorf("create notification: %w", err)
	}
	return notif, nil
}

// Get returns a single notification by ID. Returns firebase.ErrNotFound if no
// notification with the given ID exists.
func (s *notificationService) Get(ctx context.Context, id string) (*models.Notification, error) {
	return s.notifs.Get(ctx, id)
}

// List returns all persisted webhook notifications.
func (s *notificationService) List(ctx context.Context) ([]models.Notification, error) {
	return s.notifs.List(ctx)
}

// Delete permanently removes a notification by ID. Returns firebase.ErrNotFound
// if no notification with the given ID exists.
func (s *notificationService) Delete(ctx context.Context, id string) error {
	return s.notifs.Delete(ctx, id)
}

// Patch applies a partial update to a notification. Only the keys present in
// patch are changed; all other fields remain as stored. Recognised keys are
// "url", "country", "event", and "threshold". Unrecognised keys are silently
// ignored. If the resulting event is THRESHOLD but no threshold conditions are
// set, validation fails with a *models.ValidationError.
func (s *notificationService) Patch(ctx context.Context, id string, patch map[string]interface{}) (*models.Notification, error) {
	// Load the current state; only the fields present in patch will be changed.
	existing, err := s.notifs.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate and apply the URL if provided.
	if newURL, ok := patch["url"].(string); ok {
		if _, err := url.ParseRequestURI(newURL); err != nil {
			return nil, &models.ValidationError{Message: "invalid url"}
		}
		existing.URL = newURL
	}
	if newCountry, ok := patch["country"].(string); ok {
		existing.Country = strings.ToUpper(newCountry)
	}

	// Validate the event string against the allow-list before applying it.
	if newEvent, ok := patch["event"].(string); ok {
		if !models.ValidEvents[newEvent] {
			return nil, &models.ValidationError{Message: fmt.Sprintf("invalid event %q", newEvent)}
		}
		existing.Event = newEvent
	}

	// "threshold" can be set to null (clear all conditions) or replaced with a
	// new list of condition objects.
	if rawThreshold, ok := patch["threshold"]; ok {
		if rawThreshold == nil {
			existing.Thresholds = nil
		} else if rawList, ok := rawThreshold.([]interface{}); ok {
			thresholds, err := parseThresholdList(rawList)
			if err != nil {
				return nil, err
			}
			existing.Thresholds = thresholds
		}
	}

	// Post-patch consistency check: THRESHOLD events always require at least one condition.
	if existing.Event == models.EventThreshold && len(existing.Thresholds) == 0 {
		return nil, &models.ValidationError{Message: "'threshold' is required for THRESHOLD event"}
	}

	// Use Create (Set) to overwrite the existing Firestore document in-place.
	if err := s.notifs.Create(ctx, existing); err != nil { // Set (overwrite)
		return nil, fmt.Errorf("patch notification: %w", err)
	}
	return existing, nil
}

// parseThresholdList converts a raw JSON-decoded []interface{} into a validated
// []models.Threshold. Each element must be a condition object with a valid field
// and operator. Returns a *models.ValidationError if the list is empty or any
// condition is invalid.
func parseThresholdList(rawList []interface{}) ([]models.Threshold, error) {
	if len(rawList) == 0 {
		return nil, &models.ValidationError{Message: "'threshold' must contain at least one condition"}
	}

	thresholds := make([]models.Threshold, 0, len(rawList))
	for _, item := range rawList {
		conditionMap, ok := item.(map[string]interface{})
		if !ok {
			return nil, &models.ValidationError{Message: "each threshold condition must be an object with field, operator, and value"}
		}
		condition, err := parseThresholdCondition(conditionMap)
		if err != nil {
			return nil, err
		}
		thresholds = append(thresholds, *condition)
	}
	return thresholds, nil
}

// parseThresholdCondition converts the raw map[string]interface{} representation
// of a single threshold condition (as decoded from a JSON body) into a
// *models.Threshold. Field and Operator are validated against ValidThresholdFields
// and ValidThresholdOperators. Value defaults to 0 if absent or not a float64.
// JSON numbers always decode as float64 in Go's encoding/json.
func parseThresholdCondition(raw map[string]interface{}) (*models.Threshold, error) {
	condition := &models.Threshold{}
	if field, ok := raw["field"].(string); ok {
		condition.Field = field
	}
	if operator, ok := raw["operator"].(string); ok {
		condition.Operator = operator
	}
	if value, ok := raw["value"].(float64); ok {
		condition.Value = value
	}

	if !models.ValidThresholdFields[condition.Field] {
		return nil, &models.ValidationError{Message: fmt.Sprintf("invalid threshold field %q; must be pm25, pm10, temperature, or precipitation", condition.Field)}
	}
	if !models.ValidThresholdOperators[condition.Operator] {
		return nil, &models.ValidationError{Message: fmt.Sprintf("invalid threshold operator %q; must be >, <, >=, or <=", condition.Operator)}
	}
	return condition, nil
}
