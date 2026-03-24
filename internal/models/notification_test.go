package models_test

import (
	"testing"

	"envdash/internal/models"
)

func TestNotificationRequest_Validate(t *testing.T) {
	validURL := "https://example.com/hook"

	tests := []struct {
		name    string
		req     models.NotificationRequest
		wantErr bool
	}{
		{
			name:    "valid_non_threshold",
			req:     models.NotificationRequest{URL: validURL, Country: "NO", Event: "REGISTER"},
			wantErr: false,
		},
		{
			name: "valid_threshold",
			req: models.NotificationRequest{
				URL:     validURL,
				Country: "NO",
				Event:   "THRESHOLD",
				Threshold: &models.Threshold{
					Field:    "pm25",
					Operator: ">",
					Value:    50,
				},
			},
			wantErr: false,
		},
		{
			name:    "missing_url",
			req:     models.NotificationRequest{Country: "NO", Event: "REGISTER"},
			wantErr: true,
		},
		{
			name:    "invalid_url",
			req:     models.NotificationRequest{URL: "not-a-url", Country: "NO", Event: "REGISTER"},
			wantErr: true,
		},
		{
			name:    "invalid_event",
			req:     models.NotificationRequest{URL: validURL, Country: "NO", Event: "UNKNOWN"},
			wantErr: true,
		},
		{
			name:    "threshold_event_missing_threshold",
			req:     models.NotificationRequest{URL: validURL, Country: "NO", Event: "THRESHOLD"},
			wantErr: true,
		},
		{
			name: "threshold_invalid_field",
			req: models.NotificationRequest{
				URL:     validURL,
				Country: "NO",
				Event:   "THRESHOLD",
				Threshold: &models.Threshold{
					Field:    "invalid_field",
					Operator: ">",
					Value:    50,
				},
			},
			wantErr: true,
		},
		{
			name: "threshold_invalid_operator",
			req: models.NotificationRequest{
				URL:     validURL,
				Country: "NO",
				Event:   "THRESHOLD",
				Threshold: &models.Threshold{
					Field:    "pm25",
					Operator: "!=",
					Value:    50,
				},
			},
			wantErr: true,
		},
		{
			name: "threshold_operator_gte",
			req: models.NotificationRequest{
				URL:     validURL,
				Country: "NO",
				Event:   "THRESHOLD",
				Threshold: &models.Threshold{
					Field:    "temperature",
					Operator: ">=",
					Value:    30,
				},
			},
			wantErr: false,
		},
		{
			name: "threshold_operator_lte",
			req: models.NotificationRequest{
				URL:     validURL,
				Country: "NO",
				Event:   "THRESHOLD",
				Threshold: &models.Threshold{
					Field:    "precipitation",
					Operator: "<=",
					Value:    5,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				if _, ok := err.(*models.ValidationError); !ok {
					t.Errorf("Validate() returned non-ValidationError: %T", err)
				}
			}
		})
	}
}
