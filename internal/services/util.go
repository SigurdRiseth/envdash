package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// generateID returns a cryptographically random 16-character hex string.
func generateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand unavailable: %v", err))
	}
	return hex.EncodeToString(b)
}

// timestamp returns the current UTC time in the project-standard format.
func timestamp() string {
	return time.Now().UTC().Format("20060102 15:04")
}
