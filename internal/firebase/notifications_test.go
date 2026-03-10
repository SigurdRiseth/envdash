//go:build integration

package firebase

import (
	"context"
	"errors"
	"testing"

	"envdash/internal/models"
)

// testNotificationRepo creates a repo pointing at "test_notifications".
func testNotificationRepo(t *testing.T) NotificationRepository {
	t.Helper()
	fs := testClient(t)
	t.Cleanup(func() { dropCollection(t, fs, "test_notifications") })
	return &notificationRepo{fs: fs, collection: "test_notifications"}
}

func TestNotificationRepo_CreateAndGet(t *testing.T) {
	repo := testNotificationRepo(t)
	ctx := context.Background()

	n := &models.Notification{
		ID:      "integ-notif-001",
		URL:     "https://example.com/hook",
		Country: "NO",
		Event:   models.EventRegister,
	}
	if err := repo.Create(ctx, n); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, n.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.URL != n.URL {
		t.Errorf("url = %q, want %q", got.URL, n.URL)
	}
	if got.Country != n.Country {
		t.Errorf("country = %q, want %q", got.Country, n.Country)
	}
}

func TestNotificationRepo_Get_NotFound(t *testing.T) {
	repo := testNotificationRepo(t)
	ctx := context.Background()

	_, err := repo.Get(ctx, "does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestNotificationRepo_List(t *testing.T) {
	repo := testNotificationRepo(t)
	ctx := context.Background()

	items := []*models.Notification{
		{ID: "integ-list-n001", URL: "https://a.com/1", Country: "NO", Event: models.EventRegister},
		{ID: "integ-list-n002", URL: "https://a.com/2", Country: "SE", Event: models.EventDelete},
	}
	for _, item := range items {
		if err := repo.Create(ctx, item); err != nil {
			t.Fatalf("Create %s: %v", item.ID, err)
		}
	}

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) < 2 {
		t.Errorf("list length = %d, want at least 2", len(list))
	}
}

func TestNotificationRepo_Delete(t *testing.T) {
	repo := testNotificationRepo(t)
	ctx := context.Background()

	n := &models.Notification{
		ID:      "integ-del-n001",
		URL:     "https://example.com/hook",
		Country: "NO",
		Event:   models.EventRegister,
	}
	if err := repo.Create(ctx, n); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, n.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.Get(ctx, n.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestNotificationRepo_Delete_NotFound(t *testing.T) {
	repo := testNotificationRepo(t)
	ctx := context.Background()

	err := repo.Delete(ctx, "does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestNotificationRepo_ListMatching(t *testing.T) {
	repo := testNotificationRepo(t)
	ctx := context.Background()

	items := []*models.Notification{
		// matches NO + REGISTER
		{ID: "integ-match-001", URL: "https://a.com/1", Country: "NO", Event: models.EventRegister},
		// wildcard (empty country) + REGISTER — should also match NO
		{ID: "integ-match-002", URL: "https://a.com/2", Country: "", Event: models.EventRegister},
		// different event — should NOT match
		{ID: "integ-match-003", URL: "https://a.com/3", Country: "NO", Event: models.EventDelete},
		// different country — should NOT match
		{ID: "integ-match-004", URL: "https://a.com/4", Country: "SE", Event: models.EventRegister},
	}
	for _, item := range items {
		if err := repo.Create(ctx, item); err != nil {
			t.Fatalf("Create %s: %v", item.ID, err)
		}
	}

	matched, err := repo.ListMatching(ctx, "NO", models.EventRegister)
	if err != nil {
		t.Fatalf("ListMatching: %v", err)
	}
	if len(matched) != 2 {
		t.Errorf("matched = %d, want 2 (country=NO and wildcard)", len(matched))
	}
}

func TestNotificationRepo_Count(t *testing.T) {
	repo := testNotificationRepo(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		n := &models.Notification{
			ID:    "integ-count-" + string(rune('a'+i)),
			URL:   "https://example.com",
			Event: models.EventRegister,
		}
		if err := repo.Create(ctx, n); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count < 3 {
		t.Errorf("count = %d, want at least 3", count)
	}
}
