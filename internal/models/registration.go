package models

// Registration represents a persisted dashboard configuration.
type Registration struct {
	ID         string   `json:"id" firestore:"id"`
	Country    string   `json:"country" firestore:"country"`
	ISOCode    string   `json:"isoCode" firestore:"isoCode"`
	Features   Features `json:"features" firestore:"features"`
	LastChange string   `json:"lastChange" firestore:"lastChange"` // UTC timestamp in "20060102 15:04" format
}

// Features holds the feature flags and settings for a dashboard configuration.
type Features struct {
	Temperature      bool     `json:"temperature" firestore:"temperature"`
	Precipitation    bool     `json:"precipitation" firestore:"precipitation"`
	AirQuality       bool     `json:"airQuality" firestore:"airQuality"`
	Capital          bool     `json:"capital" firestore:"capital"`
	Coordinates      bool     `json:"coordinates" firestore:"coordinates"`
	Population       bool     `json:"population" firestore:"population"`
	Area             bool     `json:"area" firestore:"area"`
	TargetCurrencies []string `json:"targetCurrencies" firestore:"targetCurrencies"`
}

// RegistrationRequest is the body accepted by POST and PUT /registrations/.
// ID and LastChange are server-generated and must not be included.
type RegistrationRequest struct {
	Country  string   `json:"country"`
	ISOCode  string   `json:"isoCode"`
	Features Features `json:"features"`
}

// RegistrationCreateResponse is returned by POST /registrations/.
// LastChange is the server-assigned creation timestamp in "20060102 15:04" format.
type RegistrationCreateResponse struct {
	ID         string `json:"id"`
	LastChange string `json:"lastChange"`
}


// Validate returns a *ValidationError if the request is missing required fields.
// At least one of Country or ISOCode must be non-empty.
func (r RegistrationRequest) Validate() error {
	if r.Country == "" && r.ISOCode == "" {
		return &ValidationError{Message: "at least one of 'country' or 'isoCode' is required"}
	}
	return nil
}
