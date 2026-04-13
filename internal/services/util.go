package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// osloLocation is the time zone used for all timestamps produced by this
// service ("20060102 15:04" format). The assignment specifies Oslo (CET/CEST).
// Falls back to UTC if the system timezone database is unavailable.
var osloLocation = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Oslo")
	if err != nil {
		return time.UTC // fallback if tz database unavailable
	}
	return loc
}()

// generateID returns a cryptographically random 16-character hex string.
func generateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand unavailable: %v", err))
	}
	return hex.EncodeToString(b)
}

// timestamp returns the current Oslo time in the project-standard format.
func timestamp() string {
	return time.Now().In(osloLocation).Format("20060102 15:04")
}
