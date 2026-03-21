package handlers_test

import (
	"context"
	"errors"

	"envdash/internal/firebase"
	"envdash/internal/models"
	"envdash/internal/services"
)

// errNotFound is returned by mocks to simulate a missing document.
var errNotFound = firebase.ErrNotFound

// ---- Mock RegistrationService ----

type mockRegistrationService struct {
	regs        map[string]*models.Registration
	createError error
}

func newMockRegService() *mockRegistrationService {
	return &mockRegistrationService{regs: make(map[string]*models.Registration)}
}

func (m *mockRegistrationService) Create(_ context.Context, req models.RegistrationRequest) (*models.Registration, error) {
	if m.createError != nil {
		return nil, m.createError
	}
	if req.Country == "" && req.ISOCode == "" {
		return nil, &models.ValidationError{Message: "at least one of 'country' or 'isoCode' is required"}
	}
	reg := &models.Registration{
		ID:         "test-id-001",
		Country:    req.Country,
		ISOCode:    req.ISOCode,
		Features:   req.Features,
		LastChange: "20250301 09:15",
	}
	m.regs[reg.ID] = reg
	return reg, nil
}

func (m *mockRegistrationService) Get(_ context.Context, id string) (*models.Registration, error) {
	reg, ok := m.regs[id]
	if !ok {
		return nil, errNotFound
	}
	return reg, nil
}

func (m *mockRegistrationService) List(_ context.Context) ([]models.Registration, error) {
	list := make([]models.Registration, 0, len(m.regs))
	for _, r := range m.regs {
		list = append(list, *r)
	}
	return list, nil
}

func (m *mockRegistrationService) Update(_ context.Context, id string, req models.RegistrationRequest) (*models.Registration, error) {
	reg, ok := m.regs[id]
	if !ok {
		return nil, errNotFound
	}
	reg.Country = req.Country
	reg.ISOCode = req.ISOCode
	reg.Features = req.Features
	reg.LastChange = "20250301 10:00"
	return reg, nil
}

func (m *mockRegistrationService) Patch(_ context.Context, id string, patch map[string]interface{}) (*models.Registration, error) {
	reg, ok := m.regs[id]
	if !ok {
		return nil, errNotFound
	}
	if v, ok := patch["country"].(string); ok {
		reg.Country = v
	}
	return reg, nil
}

func (m *mockRegistrationService) Delete(_ context.Context, id string) error {
	if _, ok := m.regs[id]; !ok {
		return errNotFound
	}
	delete(m.regs, id)
	return nil
}

// ---- Mock DashboardService ----

type mockDashboardService struct {
	dashboards map[string]*models.DashboardResponse
}

func newMockDashService() *mockDashboardService {
	return &mockDashboardService{dashboards: make(map[string]*models.DashboardResponse)}
}

func (m *mockDashboardService) Get(_ context.Context, id string) (*models.DashboardResponse, error) {
	d, ok := m.dashboards[id]
	if !ok {
		return nil, errNotFound
	}
	return d, nil
}

// ---- Mock NotificationService ----

type mockNotificationService struct {
	notifs map[string]*models.Notification
}

func newMockNotifService() *mockNotificationService {
	return &mockNotificationService{notifs: make(map[string]*models.Notification)}
}

func (m *mockNotificationService) Create(_ context.Context, req models.NotificationRequest) (*models.Notification, error) {
	if req.URL == "" {
		return nil, &models.ValidationError{Message: "'url' is required"}
	}
	if !models.ValidEvents[req.Event] {
		return nil, &models.ValidationError{Message: "invalid event"}
	}
	if req.Event == models.EventThreshold && req.Threshold == nil {
		return nil, &models.ValidationError{Message: "'threshold' is required for THRESHOLD event"}
	}
	n := &models.Notification{
		ID:        "notif-id-001",
		URL:       req.URL,
		Country:   req.Country,
		Event:     req.Event,
		Threshold: req.Threshold,
	}
	m.notifs[n.ID] = n
	return n, nil
}

func (m *mockNotificationService) Get(_ context.Context, id string) (*models.Notification, error) {
	n, ok := m.notifs[id]
	if !ok {
		return nil, errNotFound
	}
	return n, nil
}

func (m *mockNotificationService) List(_ context.Context) ([]models.Notification, error) {
	list := make([]models.Notification, 0, len(m.notifs))
	for _, n := range m.notifs {
		list = append(list, *n)
	}
	return list, nil
}

func (m *mockNotificationService) Delete(_ context.Context, id string) error {
	if _, ok := m.notifs[id]; !ok {
		return errNotFound
	}
	delete(m.notifs, id)
	return nil
}

func (m *mockNotificationService) Patch(_ context.Context, id string, patch map[string]interface{}) (*models.Notification, error) {
	n, ok := m.notifs[id]
	if !ok {
		return nil, errNotFound
	}
	if v, ok := patch["url"].(string); ok {
		n.URL = v
	}
	return n, nil
}

// ---- Mock StatusService ----

type mockStatusService struct {
	response services.StatusResponse
}

func (m *mockStatusService) Get(_ context.Context) services.StatusResponse {
	return m.response
}

// ---- Mock AuthService ----

type mockAuthService struct {
	keys      map[string]bool
	createErr error
	deleteErr error
	existsErr error
}

func newMockAuthService() *mockAuthService {
	return &mockAuthService{keys: make(map[string]bool)}
}

func (m *mockAuthService) Register(_ context.Context) (string, error) {
	if m.createErr != nil {
		return "", m.createErr
	}
	key := "sk-envdash-testkey1234567890ab"
	m.keys[key] = true
	return key, nil
}

func (m *mockAuthService) Revoke(_ context.Context, key string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if !m.keys[key] {
		return errNotFound
	}
	delete(m.keys, key)
	return nil
}

func (m *mockAuthService) Validate(_ context.Context, key string) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	return m.keys[key], nil
}

// ---- helpers ----

func ptr[T any](v T) *T { return &v }

// errInternal is a non-validation, non-notFound error for testing 500 paths.
var errInternal = errors.New("internal error")
