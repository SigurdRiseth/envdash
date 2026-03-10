package models

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
