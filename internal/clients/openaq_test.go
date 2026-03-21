package clients_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"envdash/internal/clients"
)

func TestOpenAQClient_GetAirQuality(t *testing.T) {
	tests := []struct {
		name       string
		respBody   string
		respStatus int
		wantErr    bool
		wantPM25   float64
		wantLevel  string
	}{
		{
			name:       "valid response with stations",
			respStatus: http.StatusOK,
			respBody: `{
				"results": [
					{
						"id": 1,
						"latestMeasurements": [
							{"parameter":{"name":"pm25"},"value":10.0},
							{"parameter":{"name":"pm10"},"value":20.0}
						]
					},
					{
						"id": 2,
						"latestMeasurements": [
							{"parameter":{"name":"pm25"},"value":14.0}
						]
					}
				]
			}`,
			wantErr:   false,
			wantPM25:  12.0,   // mean of 10, 14
			wantLevel: "Good", // 12 µg/m³ is on the boundary of Good
		},
		{
			name:       "no stations within range",
			respStatus: http.StatusOK,
			respBody:   `{"results":[]}`,
			wantErr:    false,
			wantPM25:   -1,
			wantLevel:  "Unknown",
		},
		{
			name:       "server error",
			respStatus: http.StatusInternalServerError,
			respBody:   `{}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify API key header is set
				if r.Header.Get("X-API-Key") == "" {
					t.Error("expected X-API-Key header to be set")
				}
				w.WriteHeader(tt.respStatus)
				w.Write([]byte(tt.respBody))
			}))
			defer srv.Close()

			c := clients.NewOpenAQClient(srv.URL, "test-key", http.DefaultClient, &noopCache{}, 0)
			data, err := c.GetAirQuality(context.Background(), 62.0, 10.0)

			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr {
				if data.PM25 != tt.wantPM25 {
					t.Errorf("pm25 = %f, want %f", data.PM25, tt.wantPM25)
				}
				if data.Level != tt.wantLevel {
					t.Errorf("level = %q, want %q", data.Level, tt.wantLevel)
				}
			}
		})
	}
}

func TestOpenAQClient_CacheHit(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results":[{"id":1,"latestMeasurements":[{"parameter":{"name":"pm25"},"value":10.0}]}]}`))
	}))
	defer srv.Close()

	cache := newInMemoryCache()
	c := clients.NewOpenAQClient(srv.URL, "test-key", http.DefaultClient, cache, 0)

	// First call — hits server and populates cache
	if _, err := c.GetAirQuality(context.Background(), 62.0, 10.0); err != nil {
		t.Fatalf("first call: %v", err)
	}
	// Second call — should hit cache
	if _, err := c.GetAirQuality(context.Background(), 62.0, 10.0); err != nil {
		t.Fatalf("second call: %v", err)
	}

	if calls != 1 {
		t.Errorf("expected 1 server call (cache hit on 2nd), got %d", calls)
	}
}
