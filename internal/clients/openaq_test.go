package clients_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"envdash/internal/clients"
)

// newOpenAQStub builds a stub server that handles both the /locations and
// /locations/{id}/latest endpoints used by GetAirQuality.
// locationsBody is served for requests to /locations, latestBody for /latest.
func newOpenAQStub(t *testing.T, locationsStatus int, locationsBody, latestBody string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") == "" {
			t.Error("expected X-API-Key header to be set")
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/latest") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(latestBody))
		} else {
			w.WriteHeader(locationsStatus)
			w.Write([]byte(locationsBody))
		}
	}))
}

func TestOpenAQClient_GetAirQuality(t *testing.T) {
	tests := []struct {
		name            string
		locationsStatus int
		locationsBody   string
		latestBody      string
		wantErr         bool
		wantPM25        float64
		wantPM10        float64
		wantLevel       string
	}{
		{
			name:            "single station with pm25 and pm10",
			locationsStatus: http.StatusOK,
			locationsBody: `{"results":[{"id":1,"sensors":[` +
				`{"id":101,"parameter":{"name":"pm25"}},` +
				`{"id":102,"parameter":{"name":"pm10"}}]}]}`,
			latestBody: `{"results":[{"sensorsId":101,"value":10.0},{"sensorsId":102,"value":20.0}]}`,
			wantPM25:   10.0,
			wantPM10:   20.0,
			wantLevel:  "Good",
		},
		{
			name:            "two stations pm25 averaged",
			locationsStatus: http.StatusOK,
			// Two locations; the stub serves the same latestBody for both, so
			// sensor 101 (pm25=10) comes from location 1 and sensor 201 (pm25=14)
			// comes from location 2. mean(10, 14) = 12.0.
			locationsBody: `{"results":[` +
				`{"id":1,"sensors":[{"id":101,"parameter":{"name":"pm25"}}]},` +
				`{"id":2,"sensors":[{"id":201,"parameter":{"name":"pm25"}}]}]}`,
			latestBody: `{"results":[{"sensorsId":101,"value":10.0},{"sensorsId":201,"value":14.0}]}`,
			wantPM25:   12.0, // mean of 10.0 (loc 1) and 14.0 (loc 2)
			wantPM10:   -1,
			wantLevel:  "Good", // 12 µg/m³ is on the Good/Moderate boundary
		},
		{
			name:            "no stations within range",
			locationsStatus: http.StatusOK,
			locationsBody:   `{"results":[]}`,
			latestBody:      ``,
			wantPM25:        -1,
			wantPM10:        -1,
			wantLevel:       "Unknown",
		},
		{
			name:            "stations found but no pm sensors",
			locationsStatus: http.StatusOK,
			locationsBody:   `{"results":[{"id":1,"sensors":[{"id":101,"parameter":{"name":"o3"}}]}]}`,
			latestBody:      ``,
			wantPM25:        -1,
			wantPM10:        -1,
			wantLevel:       "Unknown",
		},
		{
			name:            "locations server error",
			locationsStatus: http.StatusInternalServerError,
			locationsBody:   `{}`,
			latestBody:      ``,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newOpenAQStub(t, tt.locationsStatus, tt.locationsBody, tt.latestBody)
			defer srv.Close()

			c := clients.NewOpenAQClient(srv.URL, "test-key", http.DefaultClient, &noopCache{}, 0)
			data, err := c.GetAirQuality(context.Background(), 59.9, 10.7)

			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if tt.wantErr {
				return
			}
			if data.PM25 != tt.wantPM25 {
				t.Errorf("pm25 = %f, want %f", data.PM25, tt.wantPM25)
			}
			if data.PM10 != tt.wantPM10 {
				t.Errorf("pm10 = %f, want %f", data.PM10, tt.wantPM10)
			}
			if data.Level != tt.wantLevel {
				t.Errorf("level = %q, want %q", data.Level, tt.wantLevel)
			}
		})
	}
}

func TestOpenAQClient_CacheHit(t *testing.T) {
	calls := 0
	locationsBody := `{"results":[{"id":1,"sensors":[{"id":101,"parameter":{"name":"pm25"}}]}]}`
	latestBody := `{"results":[{"sensorsId":101,"value":10.0}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/latest") {
			w.Write([]byte(latestBody))
		} else {
			w.Write([]byte(locationsBody))
		}
	}))
	defer srv.Close()

	cache := newInMemoryCache()
	c := clients.NewOpenAQClient(srv.URL, "test-key", http.DefaultClient, cache, 0)

	// First call: hits /locations (1) + /locations/1/latest (1) = 2 requests
	if _, err := c.GetAirQuality(context.Background(), 59.9, 10.7); err != nil {
		t.Fatalf("first call: %v", err)
	}
	firstCalls := calls

	// Second call: result is cached — no new server requests
	if _, err := c.GetAirQuality(context.Background(), 59.9, 10.7); err != nil {
		t.Fatalf("second call: %v", err)
	}

	if calls != firstCalls {
		t.Errorf("expected no additional server calls on cache hit, got %d extra", calls-firstCalls)
	}
}
