package models

import (
	"fmt"
	"net/url"
)

// Event type constants used in webhook registrations and webhook payloads.
// REGISTER, CHANGE, and DELETE fire on registration lifecycle changes.
// INVOKE fires each time a populated dashboard is retrieved.
// THRESHOLD fires when all live measured values satisfy the registered conditions.
const (
	EventRegister  = "REGISTER"
	EventChange    = "CHANGE"
	EventDelete    = "DELETE"
	EventInvoke    = "INVOKE"
	EventThreshold = "THRESHOLD"
)

// ValidEvents is the set of event type strings accepted by POST /notifications/.
// Used for validation in the service layer.
var ValidEvents = map[string]bool{
	EventRegister:  true,
	EventChange:    true,
	EventDelete:    true,
	EventInvoke:    true,
	EventThreshold: true,
}

// ValidThresholdFields is the set of dashboard measurement fields that can be
// monitored by a THRESHOLD webhook: pm25, pm10, temperature, precipitation.
var ValidThresholdFields = map[string]bool{"pm25": true, "pm10": true, "temperature": true, "precipitation": true}

// ValidThresholdOperators is the set of comparison operators accepted in a
// Threshold condition: >, <, >= and <=.
var ValidThresholdOperators = map[string]bool{">": true, "<": true, ">=": true, "<=": true}

// Notification is a persisted webhook registration. It is stored in Firestore
// and survives service restarts. Country is stored in upper-case; an empty
// Country matches events for all countries (wildcard). Thresholds is only
// present when Event is THRESHOLD; all conditions in the list must be satisfied
// simultaneously for the webhook to fire.
type Notification struct {
	ID         string      `json:"id" firestore:"id"`
	URL        string      `json:"url" firestore:"url"`
	Country    string      `json:"country" firestore:"country"`
	Event      string      `json:"event" firestore:"event"`
	Thresholds []Threshold `json:"threshold,omitempty" firestore:"threshold,omitempty"`
}

// Threshold defines a single condition within a THRESHOLD webhook registration.
// Field must be one of ValidThresholdFields; Operator must be one of
// ValidThresholdOperators. The condition is satisfied when the live measured
// value for Field satisfies: measuredValue <Operator> Value.
type Threshold struct {
	Field    string  `json:"field" firestore:"field"`
	Operator string  `json:"operator" firestore:"operator"` // >, <, >=, <=
	Value    float64 `json:"value" firestore:"value"`
}

// NotificationRequest is the body accepted by POST /notifications/.
// URL and Event are required. Country is optional (empty = all countries).
// Thresholds is required when Event is THRESHOLD; at least one condition must
// be provided. All conditions must be satisfied simultaneously for the webhook
// to fire, so multiple conditions on the same field express a range
// (e.g. temperature > 0 AND temperature <= 5).
type NotificationRequest struct {
	URL        string      `json:"url"`
	Country    string      `json:"country"`
	Event      string      `json:"event"`
	Thresholds []Threshold `json:"threshold,omitempty"`
}

// NotificationCreateResponse is the body returned by POST /notifications/ on success.
// Only the server-assigned ID is returned; the full registration can be fetched via GET.
type NotificationCreateResponse struct {
	ID string `json:"id"`
}

// Validate returns a *ValidationError if the request contains invalid or missing fields.
// URL must be present and parseable, Event must be a recognised event type, and at least
// one Threshold condition (each with a valid Field and Operator) is required when Event
// is THRESHOLD.
func (n NotificationRequest) Validate() error {
	// URL is always required and must be a parseable absolute URI.
	if n.URL == "" {
		return &ValidationError{Message: "'url' is required"}
	}
	if _, err := url.ParseRequestURI(n.URL); err != nil {
		return &ValidationError{Message: "invalid url"}
	}

	// Reject any event string that isn't in the predefined allow-list.
	if !ValidEvents[n.Event] {
		return &ValidationError{Message: fmt.Sprintf("invalid event %q; must be one of REGISTER, CHANGE, DELETE, INVOKE, THRESHOLD", n.Event)}
	}

	// THRESHOLD events require at least one condition. Each condition must
	// reference a valid field and use a recognised comparison operator.
	if n.Event == EventThreshold {
		if len(n.Thresholds) == 0 {
			return &ValidationError{Message: "'threshold' is required for THRESHOLD event and must contain at least one condition"}
		}
		for _, condition := range n.Thresholds {
			if !ValidThresholdFields[condition.Field] {
				return &ValidationError{Message: fmt.Sprintf("invalid threshold field %q; must be pm25, pm10, temperature, or precipitation", condition.Field)}
			}
			if !ValidThresholdOperators[condition.Operator] {
				return &ValidationError{Message: fmt.Sprintf("invalid threshold operator %q; must be >, <, >=, or <=", condition.Operator)}
			}
		}
	}
	return nil
}

// WebhookPayload is the JSON body POSTed to a registered webhook URL when an
// event fires. Details is only populated for THRESHOLD events; it is omitted
// from the payload for lifecycle events (REGISTER, CHANGE, DELETE, INVOKE).
type WebhookPayload struct {
	ID      string            `json:"id"`
	Country string            `json:"country"`
	Event   string            `json:"event"`
	Time    string            `json:"time"`
	Details *ThresholdDetails `json:"details,omitempty"`
}

// ThresholdDetails is the "details" object included in THRESHOLD webhook payloads.
// Conditions lists each registered condition alongside the live measured value that
// was evaluated, giving the receiver full context for why the webhook fired.
type ThresholdDetails struct {
	Conditions []ThresholdConditionDetail `json:"conditions"`
}

// ThresholdConditionDetail records a single evaluated threshold condition within
// a THRESHOLD webhook payload. Threshold is the configured limit; MeasuredValue
// is the live reading that satisfied the condition.
type ThresholdConditionDetail struct {
	Field         string  `json:"field"`
	Operator      string  `json:"operator"`
	Threshold     float64 `json:"threshold"`
	MeasuredValue float64 `json:"measuredValue"`
}
