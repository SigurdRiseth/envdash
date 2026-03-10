package clients_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"envdash/internal/clients"
)

func TestCountriesClient_GetByISO(t *testing.T) {
	tests := []struct {
		name       string
		iso        string
		respStatus int
		respBody   string
		wantErr    bool
		wantName   string
		wantCap    string
	}{
		{
			name:       "valid country",
			iso:        "NO",
			respStatus: http.StatusOK,
			respBody: `{
				"name":{"common":"Norway"},
				"capital":["Oslo"],
				"latlng":[62.0,10.0],
				"population":5379475,
				"area":323802,
				"currencies":{"NOK":{"name":"Norwegian krone","symbol":"kr"}}
			}`,
			wantErr:  false,
			wantName: "Norway",
			wantCap:  "Oslo",
		},
		{
			name:       "not found",
			iso:        "XX",
			respStatus: http.StatusNotFound,
			respBody:   `{"status":404,"message":"Not Found"}`,
			wantErr:    true,
		},
		{
			name:       "server error",
			iso:        "NO",
			respStatus: http.StatusInternalServerError,
			respBody:   `{}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.respStatus)
				w.Write([]byte(tt.respBody))
			}))
			defer srv.Close()

			c := clients.NewCountriesClient(srv.URL, http.DefaultClient, &noopCache{})

			data, err := c.GetByISO(context.Background(), tt.iso)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr {
				if data.Name != tt.wantName {
					t.Errorf("name = %q, want %q", data.Name, tt.wantName)
				}
				if data.Capital != tt.wantCap {
					t.Errorf("capital = %q, want %q", data.Capital, tt.wantCap)
				}
			}
		})
	}
}

func TestCountriesClient_CacheHit(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name":{"common":"Norway"},"capital":["Oslo"],"latlng":[62,10],"population":5000000,"area":323802,"currencies":{"NOK":{}}}`))
	}))
	defer srv.Close()

	cache := newInMemoryCache()
	c := clients.NewCountriesClient(srv.URL, http.DefaultClient, cache)

	// First call — hits server and populates cache
	if _, err := c.GetByISO(context.Background(), "NO"); err != nil {
		t.Fatalf("first call: %v", err)
	}
	// Second call — should hit cache
	if _, err := c.GetByISO(context.Background(), "NO"); err != nil {
		t.Fatalf("second call: %v", err)
	}

	if calls != 1 {
		t.Errorf("expected 1 server call (cache hit on 2nd), got %d", calls)
	}
}

// ---- stub cache helpers ----

type noopCache struct{}

func (n *noopCache) Get(_ context.Context, _ string) ([]byte, bool, error) { return nil, false, nil }
func (n *noopCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error { return nil }
func (n *noopCache) Purge(_ context.Context) (int, error)                             { return 0, nil }

type inMemoryCache struct{ data map[string][]byte }

func newInMemoryCache() *inMemoryCache { return &inMemoryCache{data: make(map[string][]byte)} }

func (c *inMemoryCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	v, ok := c.data[key]
	return v, ok, nil
}
func (c *inMemoryCache) Set(_ context.Context, key string, data []byte, _ time.Duration) error {
	c.data[key] = data
	return nil
}
func (c *inMemoryCache) Purge(_ context.Context) (int, error) { return 0, nil }
