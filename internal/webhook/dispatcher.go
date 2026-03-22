package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"envdash/internal/clients"
	"envdash/internal/models"
)

const (
	dispatchTimeout = 10 * time.Second
	retryDelay      = 2 * time.Second
)

// Dispatcher sends webhook payloads to registered URLs.
type Dispatcher struct {
	http clients.HTTPDoer
}

// NewDispatcher constructs a Dispatcher.
func NewDispatcher(http clients.HTTPDoer) *Dispatcher {
	return &Dispatcher{http: http}
}

// Dispatch sends the payload to url asynchronously. The calling goroutine is
// not blocked. Delivery is attempted once, with a single retry after 2 s on failure.
func (d *Dispatcher) Dispatch(payload models.WebhookPayload, url string) {
	go func() {
		if err := d.send(payload, url); err != nil {
			log.Printf("webhook: delivery failed to %s: %v — retrying", url, err)
			time.Sleep(retryDelay)
			if err := d.send(payload, url); err != nil {
				log.Printf("webhook: retry failed to %s: %v", url, err)
			}
		}
	}()
}

// send marshals payload to JSON and POSTs it to url with a 10-second timeout.
// Returns an error if marshalling, request construction, transport, or a non-2xx
// response status is encountered.
func (d *Dispatcher) send(payload models.WebhookPayload, url string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), dispatchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &DeliveryError{StatusCode: resp.StatusCode, URL: url}
	}
	return nil
}

// DeliveryError is returned when the webhook endpoint responds with a non-2xx status.
type DeliveryError struct {
	StatusCode int
	URL        string
}

// Error implements the error interface.
func (e *DeliveryError) Error() string {
	return "webhook: " + e.URL + " responded with status " + http.StatusText(e.StatusCode)
}
