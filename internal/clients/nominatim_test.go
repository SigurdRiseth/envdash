package clients_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"envdash/internal/clients"
)

func TestNominatimClient_GetCoordinates(t *testing.T) {
	tests := []struct {
		name       string
		iso        string
		respStatus int
		respBody   string
		wantErr    bool
		wantLat    float64
		wantLon    float64
	}{
		{
			name:       "valid result",
			iso:        "NO",
			respStatus: http.StatusOK,
			respBody:   `[{"lat":"62.0","lon":"10.0"}]`,
			wantErr:    false,
			wantLat:    62.0,
			wantLon:    10.0,
		},
		{
			name:       "empty results",
			iso:        "XX",
			respStatus: http.StatusOK,
			respBody:   `[]`,
			wantErr:    true,
		},
		{
			name:       "server error",
			iso:        "NO",
			respStatus: http.StatusInternalServerError,
			respBody:   ``,
			wantErr:    true,
		},
		{
			name:       "malformed lat",
			iso:        "NO",
			respStatus: http.StatusOK,
			respBody:   `[{"lat":"not-a-number","lon":"10.0"}]`,
			wantErr:    true,
		},
		{
			name:       "malformed lon",
			iso:        "NO",
			respStatus: http.StatusOK,
			respBody:   `[{"lat":"62.0","lon":"not-a-number"}]`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify User-Agent header is set
				if r.Header.Get("User-Agent") == "" {
					t.Error("expected User-Agent header")
				}
				// Verify query parameters
				if r.URL.Query().Get("country") == "" {
					t.Error("expected 'country' query param")
				}
				if r.URL.Query().Get("format") != "json" {
					t.Errorf("format = %q, want json", r.URL.Query().Get("format"))
				}
				w.WriteHeader(tt.respStatus)
				w.Write([]byte(tt.respBody))
			}))
			defer srv.Close()

			c := clients.NewNominatimClient(srv.URL, http.DefaultClient, &noopCache{}, 0)
			data, err := c.GetCoordinates(context.Background(), tt.iso)

			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr {
				if data.Latitude != tt.wantLat {
					t.Errorf("latitude = %v, want %v", data.Latitude, tt.wantLat)
				}
				if data.Longitude != tt.wantLon {
					t.Errorf("longitude = %v, want %v", data.Longitude, tt.wantLon)
				}
			}
		})
	}
}

func TestNominatimClient_GetCityCoordinates(t *testing.T) {
	tests := []struct {
		name       string
		city       string
		iso        string
		respStatus int
		respBody   string
		wantErr    bool
		wantLat    float64
		wantLon    float64
	}{
		{
			name:       "valid capital result",
			city:       "Oslo",
			iso:        "NO",
			respStatus: http.StatusOK,
			respBody:   `[{"lat":"59.9138688","lon":"10.7522454"}]`,
			wantErr:    false,
			wantLat:    59.9138688,
			wantLon:    10.7522454,
		},
		{
			name:       "empty results",
			city:       "Unknown City",
			iso:        "XX",
			respStatus: http.StatusOK,
			respBody:   `[]`,
			wantErr:    true,
		},
		{
			name:       "server error",
			city:       "Oslo",
			iso:        "NO",
			respStatus: http.StatusInternalServerError,
			respBody:   ``,
			wantErr:    true,
		},
		{
			name:       "malformed lat",
			city:       "Oslo",
			iso:        "NO",
			respStatus: http.StatusOK,
			respBody:   `[{"lat":"not-a-number","lon":"10.7"}]`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("User-Agent") == "" {
					t.Error("expected User-Agent header")
				}
				q := r.URL.Query()
				if q.Get("q") == "" {
					t.Error("expected 'q' query param with city name")
				}
				if q.Get("countrycodes") == "" {
					t.Error("expected 'countrycodes' query param")
				}
				if q.Get("format") != "json" {
					t.Errorf("format = %q, want json", q.Get("format"))
				}
				w.WriteHeader(tt.respStatus)
				w.Write([]byte(tt.respBody))
			}))
			defer srv.Close()

			c := clients.NewNominatimClient(srv.URL, http.DefaultClient, &noopCache{}, 0)
			data, err := c.GetCityCoordinates(context.Background(), tt.city, tt.iso)

			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr {
				if data.Latitude != tt.wantLat {
					t.Errorf("latitude = %v, want %v", data.Latitude, tt.wantLat)
				}
				if data.Longitude != tt.wantLon {
					t.Errorf("longitude = %v, want %v", data.Longitude, tt.wantLon)
				}
			}
		})
	}
}

func TestNominatimClient_GetCityCoordinates_CacheHit(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"lat":"59.9138688","lon":"10.7522454"}]`))
	}))
	defer srv.Close()

	cache := newInMemoryCache()
	c := clients.NewNominatimClient(srv.URL, http.DefaultClient, cache, 0)

	if _, err := c.GetCityCoordinates(context.Background(), "Oslo", "NO"); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := c.GetCityCoordinates(context.Background(), "Oslo", "NO"); err != nil {
		t.Fatalf("second call: %v", err)
	}

	if calls != 1 {
		t.Errorf("expected 1 server call (cache hit on 2nd), got %d", calls)
	}
}

func TestNominatimClient_CacheHit(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"lat":"62.0","lon":"10.0"}]`))
	}))
	defer srv.Close()

	cache := newInMemoryCache()
	c := clients.NewNominatimClient(srv.URL, http.DefaultClient, cache, 0)

	// First call — hits server and populates cache
	if _, err := c.GetCoordinates(context.Background(), "NO"); err != nil {
		t.Fatalf("first call: %v", err)
	}
	// Second call — should hit cache, not server
	if _, err := c.GetCoordinates(context.Background(), "NO"); err != nil {
		t.Fatalf("second call: %v", err)
	}

	if calls != 1 {
		t.Errorf("expected 1 server call (cache hit on 2nd), got %d", calls)
	}
}
