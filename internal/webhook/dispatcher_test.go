package webhook_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"envdash/internal/models"
	"envdash/internal/webhook"
)

func TestDispatcher_Dispatch_DeliverPayload(t *testing.T) {
	var received models.WebhookPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := webhook.NewDispatcher(http.DefaultClient, slog.New(slog.NewTextHandler(io.Discard, nil)))
	payload := models.WebhookPayload{
		ID:      "abc123",
		Country: "NO",
		Event:   models.EventInvoke,
		Time:    "20250301 14:22",
	}
	d.Dispatch(payload, srv.URL)

	// Give the goroutine time to complete
	time.Sleep(100 * time.Millisecond)

	if received.ID != payload.ID {
		t.Errorf("received ID = %q, want %q", received.ID, payload.ID)
	}
	if received.Event != payload.Event {
		t.Errorf("received Event = %q, want %q", received.Event, payload.Event)
	}
}

func TestDispatcher_Dispatch_RetriesOnFailure(t *testing.T) {
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := webhook.NewDispatcher(http.DefaultClient, slog.New(slog.NewTextHandler(io.Discard, nil)))
	d.Dispatch(models.WebhookPayload{ID: "x", Event: models.EventRegister, Time: "t"}, srv.URL)

	// Wait long enough for the retry (retryDelay is 2s in dispatcher)
	time.Sleep(2500 * time.Millisecond)

	if got := callCount.Load(); got != 2 {
		t.Errorf("expected 2 calls (initial + retry), got %d", got)
	}
}

func TestDispatcher_Dispatch_NocrashOnUnreachable(t *testing.T) {
	d := webhook.NewDispatcher(http.DefaultClient, slog.New(slog.NewTextHandler(io.Discard, nil)))
	// Should not panic even when the URL is unreachable
	d.Dispatch(models.WebhookPayload{ID: "y", Event: models.EventDelete, Time: "t"}, "http://127.0.0.1:1")
	time.Sleep(2500 * time.Millisecond) // wait for retry to finish too
}

func TestDispatcher_Dispatch_ThresholdPayload(t *testing.T) {
	var received models.WebhookPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := webhook.NewDispatcher(http.DefaultClient, slog.New(slog.NewTextHandler(io.Discard, nil)))
	payload := models.WebhookPayload{
		ID:      "thr1",
		Country: "NO",
		Event:   models.EventThreshold,
		Time:    "20250301 14:22",
		Details: &models.ThresholdDetails{
			Field:         "pm25",
			Operator:      ">",
			Threshold:     35.0,
			MeasuredValue: 47.3,
		},
	}
	d.Dispatch(payload, srv.URL)
	time.Sleep(100 * time.Millisecond)

	if received.Details == nil {
		t.Fatal("expected Details in threshold payload, got nil")
	}
	if received.Details.MeasuredValue != 47.3 {
		t.Errorf("MeasuredValue = %f, want 47.3", received.Details.MeasuredValue)
	}
}
