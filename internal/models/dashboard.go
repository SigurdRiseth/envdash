package models

// DashboardResponse is the response body for GET /dashboards/{id}.
type DashboardResponse struct {
	Country       string            `json:"country"`
	ISOCode       string            `json:"isoCode"`
	Features      DashboardFeatures `json:"features"`
	LastRetrieval string            `json:"lastRetrieval"` // UTC timestamp in "20060102 15:04" format
}

// DashboardFeatures holds the populated feature values. Fields are pointers
// so they can be omitted (null) when the corresponding feature flag is false
// or when data could not be retrieved.
type DashboardFeatures struct {
	Temperature      *float64           `json:"temperature,omitempty"`
	Precipitation    *float64           `json:"precipitation,omitempty"`
	AirQuality       *AirQualityData    `json:"airQuality,omitempty"`
	Capital          *string            `json:"capital,omitempty"`
	Coordinates      *CoordinatesData   `json:"coordinates,omitempty"`
	Population       *int64             `json:"population,omitempty"`
	Area             *float64           `json:"area,omitempty"`
	TargetCurrencies map[string]float64 `json:"targetCurrencies,omitempty"`
}

// AirQualityData holds aggregated air quality readings and an AQI level label.
type AirQualityData struct {
	PM25  float64 `json:"pm25"`  // µg/m³ mean across nearby stations; -1 when no data is available
	PM10  float64 `json:"pm10"`  // µg/m³ mean across nearby stations; -1 when no data is available
	Level string  `json:"level"` // EPA AQI category derived from PM2.5 (e.g. "Good", "Moderate")
}

// CoordinatesData holds country centroid coordinates.
type CoordinatesData struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
