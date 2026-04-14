package models

// AQILevel returns the EPA AQI level label for a given PM2.5 concentration (µg/m³).
// Breakpoints from https://aqs.epa.gov/aqsweb/documents/codetables/aqi_breakpoints.html
func AQILevel(pm25 float64) string {
	switch {
	case pm25 < 0:
		// OpenAQ signals "no data available" with -1; propagate as Unknown.
		return "Unknown"
	case pm25 <= 12.0:
		return "Good"
	case pm25 <= 35.4:
		return "Moderate"
	case pm25 <= 55.4:
		return "Unhealthy for Sensitive Groups"
	case pm25 <= 150.4:
		return "Unhealthy"
	case pm25 <= 250.4:
		return "Very Unhealthy"
	default:
		// Any PM2.5 reading above 250.4 µg/m³ falls in the Hazardous category.
		return "Hazardous"
	}
}
