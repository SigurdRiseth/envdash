package models

import (
	"fmt"
	"net/url"
)

// Event types for webhook notifications.
const (
	EventRegister  = "REGISTER"
	EventChange    = "CHANGE"
	EventDelete    = "DELETE"
	EventInvoke    = "INVOKE"
	EventThreshold = "THRESHOLD"
)

// ValidEvents is the set of accepted event type strings.
var ValidEvents = map[string]bool{
	EventRegister:  true,
	EventChange:    true,
	EventDelete:    true,
	EventInvoke:    true,
	EventThreshold: true,
}

// Valid threshold fields and operators.
var (
	ValidThresholdFields    = map[string]bool{"pm25": true, "pm10": true, "temperature": true, "precipitation": true}
	ValidThresholdOperators = map[string]bool{">": true, "<": true, ">=": true, "<=": true}
)

// Notification represents a persisted webhook registration.
type Notification struct {
	ID        string     `json:"id" firestore:"id"`
	URL       string     `json:"url" firestore:"url"`
	Country   string     `json:"country" firestore:"country"`
	Event     string     `json:"event" firestore:"event"`
	Threshold *Threshold `json:"threshold,omitempty" firestore:"threshold,omitempty"`
}

// Threshold defines the conditions under which a THRESHOLD webhook fires.
type Threshold struct {
	Field    string  `json:"field" firestore:"field"`
	Operator string  `json:"operator" firestore:"operator"` // >, <, >=, <=
	Value    float64 `json:"value" firestore:"value"`
}

// NotificationRequest is the body accepted by POST /notifications/.
type NotificationRequest struct {
	URL       string     `json:"url"`
	Country   string     `json:"country"`
	Event     string     `json:"event"`
	Threshold *Threshold `json:"threshold,omitempty"`
}

// NotificationCreateResponse is returned by POST /notifications/.
type NotificationCreateResponse struct {
	ID string `json:"id"`
}

// Validate returns a *ValidationError if the request contains invalid or missing fields.
// URL must be present and parseable, Event must be a recognised event type, and a
// Threshold (with valid Field and Operator) is required when Event is THRESHOLD.
func (n NotificationRequest) Validate() error {
	if n.URL == "" {
		return &ValidationError{Message: "'url' is required"}
	}
	if _, err := url.ParseRequestURI(n.URL); err != nil {
		return &ValidationError{Message: "invalid url"}
	}
	if !ValidEvents[n.Event] {
		return &ValidationError{Message: fmt.Sprintf("invalid event %q; must be one of REGISTER, CHANGE, DELETE, INVOKE, THRESHOLD", n.Event)}
	}
	if n.Event == EventThreshold {
		if n.Threshold == nil {
			return &ValidationError{Message: "'threshold' is required for THRESHOLD event"}
		}
		if !ValidThresholdFields[n.Threshold.Field] {
			return &ValidationError{Message: fmt.Sprintf("invalid threshold field %q; must be pm25, pm10, temperature, or precipitation", n.Threshold.Field)}
		}
		if !ValidThresholdOperators[n.Threshold.Operator] {
			return &ValidationError{Message: fmt.Sprintf("invalid threshold operator %q; must be >, <, >=, or <=", n.Threshold.Operator)}
		}
	}
	return nil
}

// WebhookPayload is the JSON body POSTed to a registered webhook URL when an event fires.
type WebhookPayload struct {
	ID      string            `json:"id"`
	Country string            `json:"country"`
	Event   string            `json:"event"`
	Time    string            `json:"time"`
	Details *ThresholdDetails `json:"details,omitempty"`
}

// ThresholdDetails is included in webhook payloads for THRESHOLD events.
type ThresholdDetails struct {
	Field         string  `json:"field"`
	Operator      string  `json:"operator"`
	Threshold     float64 `json:"threshold"`
	MeasuredValue float64 `json:"measuredValue"`
}
