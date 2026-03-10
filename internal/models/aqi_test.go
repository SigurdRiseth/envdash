package models_test

import (
	"testing"

	"envdash/internal/models"
)

func TestAQILevel(t *testing.T) {
	tests := []struct {
		pm25  float64
		level string
	}{
		{-1, "Unknown"},
		{0, "Good"},
		{12.0, "Good"},
		{12.1, "Moderate"},
		{35.4, "Moderate"},
		{35.5, "Unhealthy for Sensitive Groups"},
		{55.4, "Unhealthy for Sensitive Groups"},
		{55.5, "Unhealthy"},
		{150.4, "Unhealthy"},
		{150.5, "Very Unhealthy"},
		{250.4, "Very Unhealthy"},
		{250.5, "Hazardous"},
		{999.9, "Hazardous"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := models.AQILevel(tt.pm25)
			if got != tt.level {
				t.Errorf("AQILevel(%.1f) = %q, want %q", tt.pm25, got, tt.level)
			}
		})
	}
}
