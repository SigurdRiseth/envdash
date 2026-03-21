package models_test

import (
	"testing"

	"envdash/internal/models"
)

func TestRegistrationRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     models.RegistrationRequest
		wantErr bool
	}{
		{
			name:    "valid_with_both_fields",
			req:     models.RegistrationRequest{Country: "Norway", ISOCode: "NO"},
			wantErr: false,
		},
		{
			name:    "valid_country_only",
			req:     models.RegistrationRequest{Country: "Norway"},
			wantErr: false,
		},
		{
			name:    "valid_isocode_only",
			req:     models.RegistrationRequest{ISOCode: "NO"},
			wantErr: false,
		},
		{
			name:    "invalid_both_empty",
			req:     models.RegistrationRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				var ve *models.ValidationError
				if ok := isValidationError(err, &ve); !ok {
					t.Errorf("Validate() returned non-ValidationError: %T", err)
				}
			}
		})
	}
}

// isValidationError checks whether err is a *ValidationError and sets target.
func isValidationError(err error, target **models.ValidationError) bool {
	ve, ok := err.(*models.ValidationError)
	if ok {
		*target = ve
	}
	return ok
}
