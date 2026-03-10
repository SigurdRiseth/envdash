package clients_test

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"envdash/internal/clients"
)

func TestMeteoClient_GetForecast(t *testing.T) {
	body := `{
		"hourly": {
			"temperature_2m": [1.0, 2.0, 3.0],
			"precipitation":  [0.1, 0.2, 0.3]
		}
	}`

	tests := []struct {
		name       string
		respStatus int
		respBody   string
		wantErr    bool
		wantTemp   float64
		wantPrecip float64
	}{
		{
			name:       "valid response",
			respStatus: http.StatusOK,
			respBody:   body,
			wantErr:    false,
			wantTemp:   2.0,
			wantPrecip: 0.2,
		},
		{
			name:       "server error",
			respStatus: http.StatusBadGateway,
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

			c := clients.NewMeteoClient(srv.URL, http.DefaultClient, &noopCache{}, 0)
			data, err := c.GetForecast(context.Background(), 62.0, 10.0)

			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr {
				const eps = 1e-9
				if math.Abs(data.Temperature-tt.wantTemp) > eps {
					t.Errorf("temperature = %f, want %f", data.Temperature, tt.wantTemp)
				}
				if math.Abs(data.Precipitation-tt.wantPrecip) > eps {
					t.Errorf("precipitation = %f, want %f", data.Precipitation, tt.wantPrecip)
				}
			}
		})
	}
}
