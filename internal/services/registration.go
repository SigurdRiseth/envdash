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

func (s *registrationService) Create(ctx context.Context, req models.RegistrationRequest) (*models.Registration, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

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

	s.fireLifecycle(ctx, reg.ISOCode, models.EventRegister)
	return reg, nil
}

func (s *registrationService) Get(ctx context.Context, id string) (*models.Registration, error) {
	return s.regs.Get(ctx, id)
}

func (s *registrationService) List(ctx context.Context) ([]models.Registration, error) {
	return s.regs.List(ctx)
}

func (s *registrationService) Update(ctx context.Context, id string, req models.RegistrationRequest) (*models.Registration, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	existing, err := s.regs.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	existing.Country = req.Country
	existing.ISOCode = strings.ToUpper(req.ISOCode)
	existing.Features = req.Features
	existing.LastChange = timestamp()

	if err := s.regs.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update registration: %w", err)
	}

	s.fireLifecycle(ctx, existing.ISOCode, models.EventChange)
	return existing, nil
}

// Patch applies a partial update. Only the fields present in the patch map are
// changed. Recognised top-level keys: "country", "isoCode", "features".
// For "features", only the sub-fields present in the nested map are changed.
func (s *registrationService) Patch(ctx context.Context, id string, patch map[string]interface{}) (*models.Registration, error) {
	existing, err := s.regs.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if v, ok := patch["country"].(string); ok {
		existing.Country = v
	}
	if v, ok := patch["isoCode"].(string); ok {
		existing.ISOCode = strings.ToUpper(v)
	}
	if fm, ok := patch["features"].(map[string]interface{}); ok {
		applyFeaturePatch(&existing.Features, fm)
	}
	existing.LastChange = timestamp()

	if err := s.regs.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("patch registration: %w", err)
	}

	s.fireLifecycle(ctx, existing.ISOCode, models.EventChange)
	return existing, nil
}

func (s *registrationService) Delete(ctx context.Context, id string) error {
	reg, err := s.regs.Get(ctx, id)
	if err != nil {
		return err
	}

	if err := s.regs.Delete(ctx, id); err != nil {
		return err
	}

	s.fireLifecycle(ctx, reg.ISOCode, models.EventDelete)
	return nil
}

// fireLifecycle dispatches all matching webhook notifications for a lifecycle event.
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
		currencies := make([]string, 0, len(v))
		for _, c := range v {
			if s, ok := c.(string); ok {
				currencies = append(currencies, strings.ToUpper(s))
			}
		}
		f.TargetCurrencies = currencies
	}
}

