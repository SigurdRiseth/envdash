package clients_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"envdash/internal/clients"
)

func TestCurrencyClient_GetRates(t *testing.T) {
	body := `{"EUR":0.087701,"USD":0.095184,"SEK":0.978272,"GBP":0.075}`

	tests := []struct {
		name       string
		base       string
		targets    []string
		respStatus int
		respBody   string
		wantErr    bool
		wantLen    int
	}{
		{
			name:       "filters to requested currencies",
			base:       "NOK",
			targets:    []string{"EUR", "USD"},
			respStatus: http.StatusOK,
			respBody:   body,
			wantErr:    false,
			wantLen:    2,
		},
		{
			name:       "not found",
			base:       "XYZ",
			targets:    []string{"EUR"},
			respStatus: http.StatusNotFound,
			respBody:   `{}`,
			wantErr:    true,
		},
		{
			name:       "empty targets",
			base:       "NOK",
			targets:    nil,
			respStatus: http.StatusOK,
			respBody:   body,
			wantErr:    false,
			wantLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.respStatus)
				w.Write([]byte(tt.respBody))
			}))
			defer srv.Close()

			c := clients.NewCurrencyClient(srv.URL, http.DefaultClient, &noopCache{}, 0)
			rates, err := c.GetRates(context.Background(), tt.base, tt.targets)

			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr && len(rates) != tt.wantLen {
				t.Errorf("len(rates) = %d, want %d", len(rates), tt.wantLen)
			}
		})
	}
}

func TestCurrencyClient_CacheHit(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"EUR":0.087701,"USD":0.095184}`))
	}))
	defer srv.Close()

	cache := newInMemoryCache()
	c := clients.NewCurrencyClient(srv.URL, http.DefaultClient, cache, 0)

	// First call — hits server and populates cache
	if _, err := c.GetRates(context.Background(), "NOK", []string{"EUR"}); err != nil {
		t.Fatalf("first call: %v", err)
	}
	// Second call — should hit cache
	if _, err := c.GetRates(context.Background(), "NOK", []string{"EUR"}); err != nil {
		t.Fatalf("second call: %v", err)
	}

	if calls != 1 {
		t.Errorf("expected 1 server call (cache hit on 2nd), got %d", calls)
	}
}
