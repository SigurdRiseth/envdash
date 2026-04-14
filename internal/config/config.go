package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration read from environment variables.
type Config struct {
	Port             string
	FirebaseProject  string
	FirebaseCreds    string // path to JSON file (GOOGLE_APPLICATION_CREDENTIALS)
	FirebaseCredsJSON string // inline JSON (FIREBASE_CREDENTIALS_JSON)
	OpenAQKey        string

	CountriesBaseURL string
	MeteoBaseURL     string
	OpenAQBaseURL    string
	NominatimBaseURL string
	CurrencyBaseURL  string

	CacheTTLHours    int // 0 means use per-type defaults
	CachePurgeHours  int
}

// Load reads configuration from environment variables.
// Returns an error if any required variable is missing.
func Load() (*Config, error) {
	// Each field falls back to a sensible default when the env var is absent.
	// API base URLs default to the course-provided instances; override in .env
	// to point at local stubs during development or testing.
	cfg := &Config{
		Port:              getEnv("SERVER_PORT", "8080"),
		FirebaseProject:   os.Getenv("FIREBASE_PROJECT_ID"),
		FirebaseCreds:     os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		FirebaseCredsJSON: os.Getenv("FIREBASE_CREDENTIALS_JSON"),
		OpenAQKey:         os.Getenv("OPENAQ_API_KEY"),

		CountriesBaseURL: getEnv("COUNTRIES_API_URL", "http://129.241.150.113:8080/v3.1"),
		MeteoBaseURL:     getEnv("METEO_API_URL", "https://api.open-meteo.com/v1"),
		OpenAQBaseURL:    getEnv("OPENAQ_API_URL", "https://api.openaq.org/v3"),
		NominatimBaseURL: getEnv("NOMINATIM_API_URL", "https://nominatim.openstreetmap.org"),
		CurrencyBaseURL:  getEnv("CURRENCY_API_URL", "http://129.241.150.113:9090/currency"),

		// CachePurgeHours defaults to 1; set to 0 to disable background purging.
		CachePurgeHours: getEnvInt("CACHE_PURGE_INTERVAL_HOURS", 1),
		// CacheTTLHours defaults to 0, which tells each client to use its own built-in TTL.
		CacheTTLHours: getEnvInt("CACHE_TTL_HOURS", 0),
	}

	// These two values have no safe default and must be explicitly configured.
	if cfg.FirebaseProject == "" {
		return nil, fmt.Errorf("FIREBASE_PROJECT_ID is required")
	}
	if cfg.OpenAQKey == "" {
		return nil, fmt.Errorf("OPENAQ_API_KEY is required")
	}

	return cfg, nil
}

// getEnv returns the value of the environment variable named key,
// or fallback if the variable is unset or empty.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvInt returns the integer value of the environment variable named key,
// or fallback if the variable is unset, empty, or not a valid integer.
func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
