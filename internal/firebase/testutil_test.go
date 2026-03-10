//go:build integration

package firebase

import (
	"context"
	"os"
	"testing"

	"cloud.google.com/go/firestore"

	"envdash/internal/config"
)

// testClient returns a Firestore client for integration tests.
// Skips the test if FIREBASE_PROJECT_ID is not set in the environment.
// Registers fs.Close() via t.Cleanup so callers don't need to close it manually.
func testClient(t *testing.T) *firestore.Client {
	t.Helper()

	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		t.Skip("FIREBASE_PROJECT_ID not set — skipping integration test")
	}

	// Build a minimal config — only the Firestore fields are needed here.
	// We deliberately avoid config.Load() because that also requires OPENAQ_API_KEY.
	cfg := &config.Config{
		FirebaseProject:   projectID,
		FirebaseCreds:     os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		FirebaseCredsJSON: os.Getenv("FIREBASE_CREDENTIALS_JSON"),
	}

	fs, err := NewFirestoreClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("firestore: %v", err)
	}
	t.Cleanup(func() { fs.Close() })
	return fs
}

// dropCollection deletes all documents in the given collection.
// Used in cleanup callbacks to leave the test database clean.
func dropCollection(t *testing.T, fs *firestore.Client, collection string) {
	t.Helper()
	ctx := context.Background()
	docs, err := fs.Collection(collection).Documents(ctx).GetAll()
	if err != nil {
		t.Logf("dropCollection %q: list failed: %v", collection, err)
		return
	}
	for _, d := range docs {
		if _, err := d.Ref.Delete(ctx); err != nil {
			t.Logf("dropCollection %q: delete %s failed: %v", collection, d.Ref.ID, err)
		}
	}
}
