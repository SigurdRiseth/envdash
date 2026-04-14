package services

import (
	"context"
	"fmt"
	"strings"

	"envdash/internal/firebase"
	"envdash/internal/models"
	"envdash/internal/webhook"
)

// RegistrationService manages dashboard configuration lifecycle.
type RegistrationService interface {
	Create(ctx context.Context, req models.RegistrationRequest) (*models.Registration, error)
	Get(ctx context.Context, id string) (*models.Registration, error)
	List(ctx context.Context) ([]models.Registration, error)
	Update(ctx context.Context, id string, req models.RegistrationRequest) (*models.Registration, error)
	Patch(ctx context.Context, id string, patch map[string]interface{}) (*models.Registration, error)
	Delete(ctx context.Context, id string) error
}

type registrationService struct {
	regs       firebase.RegistrationRepository
	notifs     firebase.NotificationRepository
	dispatcher *webhook.Dispatcher
}

// NewRegistrationService constructs a RegistrationService.
func NewRegistrationService(
	regs firebase.RegistrationRepository,
	notifs firebase.NotificationRepository,
	dispatcher *webhook.Dispatcher,
) RegistrationService {
	return &registrationService{regs: regs, notifs: notifs, dispatcher: dispatcher}
}

// Create validates the request, generates a new registration with a random ID,
// persists it to Firestore, and fires REGISTER lifecycle webhooks.
func (s *registrationService) Create(ctx context.Context, req models.RegistrationRequest) (*models.Registration, error) {
	// Reject the request early if required fields are missing.
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Build the persisted registration. The ID and timestamp are server-generated
	// so clients cannot influence them. ISOCode is normalised to upper-case so
	// later lookups are case-insensitive.
	reg := &models.Registration{
		ID:         generateID(),
		Country:    req.Country,
		ISOCode:    strings.ToUpper(req.ISOCode),
		Features:   req.Features,
		LastChange: timestamp(),
	}

	if err := s.regs.Create(ctx, reg); err != nil {
		return nil, fmt.Errorf("create registration: %w", err)
	}

	// Notify any webhooks subscribed to the REGISTER lifecycle event.
	s.fireLifecycle(ctx, reg.ISOCode, models.EventRegister)
	return reg, nil
}

// Get returns a single registration by ID. Returns firebase.ErrNotFound if no
// registration with the given ID exists.
func (s *registrationService) Get(ctx context.Context, id string) (*models.Registration, error) {
	return s.regs.Get(ctx, id)
}

// List returns all persisted dashboard registrations.
func (s *registrationService) List(ctx context.Context) ([]models.Registration, error) {
	return s.regs.List(ctx)
}

// Update fully replaces an existing registration (PUT semantics). All fields
// in req overwrite the stored values. Returns ErrNotFound if the ID doesn't exist.
// Fires CHANGE lifecycle webhooks on success.
func (s *registrationService) Update(ctx context.Context, id string, req models.RegistrationRequest) (*models.Registration, error) {
	// Validate before touching the database to avoid a partial-update scenario.
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Fetch the existing document; returns ErrNotFound if it doesn't exist.
	existing, err := s.regs.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Overwrite all mutable fields (PUT semantics — full replacement).
	existing.Country = req.Country
	existing.ISOCode = strings.ToUpper(req.ISOCode)
	existing.Features = req.Features
	existing.LastChange = timestamp()

	if err := s.regs.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update registration: %w", err)
	}

	// Notify webhooks subscribed to the CHANGE lifecycle event.
	s.fireLifecycle(ctx, existing.ISOCode, models.EventChange)
	return existing, nil
}

// Patch applies a partial update. Only the fields present in the patch map are
// changed. Recognised top-level keys: "country", "isoCode", "features".
// For "features", only the sub-fields present in the nested map are changed.
func (s *registrationService) Patch(ctx context.Context, id string, patch map[string]interface{}) (*models.Registration, error) {
	// Load the current state so we can merge only the provided fields into it.
	existing, err := s.regs.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply top-level scalar fields if present in the patch map.
	if v, ok := patch["country"].(string); ok {
		existing.Country = v
	}
	if v, ok := patch["isoCode"].(string); ok {
		existing.ISOCode = strings.ToUpper(v)
	}

	// Recurse into "features" if present, updating only the supplied sub-fields.
	if fm, ok := patch["features"].(map[string]interface{}); ok {
		applyFeaturePatch(&existing.Features, fm)
	}
	existing.LastChange = timestamp()

	if err := s.regs.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("patch registration: %w", err)
	}

	// Notify webhooks subscribed to the CHANGE lifecycle event.
	s.fireLifecycle(ctx, existing.ISOCode, models.EventChange)
	return existing, nil
}

// Delete permanently removes a registration by ID. Returns ErrNotFound if the
// ID doesn't exist. Fires DELETE lifecycle webhooks on success.
func (s *registrationService) Delete(ctx context.Context, id string) error {
	// Fetch the registration first so we have the ISO code available for
	// webhook dispatch after deletion, and so we can return a proper 404
	// before attempting the delete.
	reg, err := s.regs.Get(ctx, id)
	if err != nil {
		return err
	}

	if err := s.regs.Delete(ctx, id); err != nil {
		return err
	}

	// Notify webhooks subscribed to the DELETE lifecycle event.
	s.fireLifecycle(ctx, reg.ISOCode, models.EventDelete)
	return nil
}

// fireLifecycle looks up all webhook notifications matching isoCode and event,
// then dispatches a payload to each registered URL asynchronously. It is called
// after every mutating operation (Create → REGISTER, Update/Patch → CHANGE,
// Delete → DELETE). Errors from the repository are silently ignored so that a
// failing Firestore read never causes the original operation to appear to fail.
func (s *registrationService) fireLifecycle(ctx context.Context, isoCode, event string) {
	notifs, err := s.notifs.ListMatching(ctx, isoCode, event)
	if err != nil {
		return
	}
	ts := timestamp()
	for _, n := range notifs {
		s.dispatcher.Dispatch(models.WebhookPayload{
			ID:      n.ID,
			Country: isoCode,
			Event:   event,
			Time:    ts,
		}, n.URL)
	}
}

// applyFeaturePatch applies the key-value pairs in m to f, updating only the
// fields that are present. Keys not recognised as feature fields are silently
// ignored. Type mismatches (e.g. a non-bool value for "temperature") are also
// silently ignored — only correctly-typed values are applied. This matches the
// partial-update semantics of PATCH: absent means unchanged.
func applyFeaturePatch(f *models.Features, m map[string]interface{}) {
	if v, ok := m["temperature"].(bool); ok {
		f.Temperature = v
	}
	if v, ok := m["precipitation"].(bool); ok {
		f.Precipitation = v
	}
	if v, ok := m["airQuality"].(bool); ok {
		f.AirQuality = v
	}
	if v, ok := m["capital"].(bool); ok {
		f.Capital = v
	}
	if v, ok := m["coordinates"].(bool); ok {
		f.Coordinates = v
	}
	if v, ok := m["population"].(bool); ok {
		f.Population = v
	}
	if v, ok := m["area"].(bool); ok {
		f.Area = v
	}
	if v, ok := m["targetCurrencies"].([]interface{}); ok {
		// JSON arrays decode as []interface{}, so each element must be type-asserted
		// to string. Non-string elements are silently skipped. All codes are
		// upper-cased to match the currency API's expected format.
		currencies := make([]string, 0, len(v))
		for _, c := range v {
			if s, ok := c.(string); ok {
				currencies = append(currencies, strings.ToUpper(s))
			}
		}
		f.TargetCurrencies = currencies
	}
}

