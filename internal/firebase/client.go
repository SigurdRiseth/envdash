package firebase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"

	"envdash/internal/config"
)

// NewFirestoreClient initialises a Firestore client from the given config.
// Credentials are sourced from GOOGLE_APPLICATION_CREDENTIALS (file path) or
// FIREBASE_CREDENTIALS_JSON (inline JSON string).
func NewFirestoreClient(ctx context.Context, cfg *config.Config) (*firestore.Client, error) {
	var opts []option.ClientOption

	// Determine the credential source. Inline JSON takes precedence over a
	// file path, which takes precedence over Application Default Credentials.
	switch {
	case cfg.FirebaseCredsJSON != "":
		// Validate the JSON before handing it to the SDK so errors are caught early.
		var raw json.RawMessage
		if err := json.Unmarshal([]byte(cfg.FirebaseCredsJSON), &raw); err != nil {
			return nil, fmt.Errorf("invalid FIREBASE_CREDENTIALS_JSON: %w", err)
		}
		opts = append(opts, option.WithCredentialsJSON([]byte(cfg.FirebaseCredsJSON)))

	case cfg.FirebaseCreds != "":
		// Verify the file exists before passing it to the SDK to surface a clear error.
		if _, err := os.Stat(cfg.FirebaseCreds); err != nil {
			return nil, fmt.Errorf("credentials file not found: %s", cfg.FirebaseCreds)
		}
		opts = append(opts, option.WithCredentialsFile(cfg.FirebaseCreds))

	default:
		// Fall back to Application Default Credentials (e.g. gcloud auth application-default login).
	}

	app, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: cfg.FirebaseProject,
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("initialise Firebase app: %w", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialise Firestore client: %w", err)
	}

	return client, nil
}
