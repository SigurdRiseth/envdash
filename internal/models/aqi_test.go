package models_test

import (
	"testing"

	"envdash/internal/models"
)

func TestAQILevel(t *testing.T) {
	tests := []struct {
		name  string
		pm25  float64
		level string
	}{
		{"negative/unknown", -1, "Unknown"},
		{"zero/good", 0, "Good"},
		{"boundary_good_upper", 12.0, "Good"},
		{"boundary_moderate_lower", 12.1, "Moderate"},
		{"boundary_moderate_upper", 35.4, "Moderate"},
		{"boundary_sensitive_lower", 35.5, "Unhealthy for Sensitive Groups"},
		{"boundary_sensitive_upper", 55.4, "Unhealthy for Sensitive Groups"},
		{"boundary_unhealthy_lower", 55.5, "Unhealthy"},
		{"boundary_unhealthy_upper", 150.4, "Unhealthy"},
		{"boundary_very_unhealthy_lower", 150.5, "Very Unhealthy"},
		{"boundary_very_unhealthy_upper", 250.4, "Very Unhealthy"},
		{"boundary_hazardous_lower", 250.5, "Hazardous"},
		{"high/hazardous", 999.9, "Hazardous"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := models.AQILevel(tt.pm25)
			if got != tt.level {
				t.Errorf("AQILevel(%.1f) = %q, want %q", tt.pm25, got, tt.level)
			}
		})
	}
}
