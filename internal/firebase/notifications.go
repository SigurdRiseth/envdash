package firebase

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"envdash/internal/models"
)

const notificationsCollection = "notifications"

// NotificationRepository defines persistence operations for webhook registrations.
type NotificationRepository interface {
	Create(ctx context.Context, n *models.Notification) error
	Get(ctx context.Context, id string) (*models.Notification, error)
	List(ctx context.Context) ([]models.Notification, error)
	Delete(ctx context.Context, id string) error
	// ListMatching returns notifications that match the given ISO country code and event.
	// A notification with an empty Country field matches any country.
	ListMatching(ctx context.Context, isoCode, event string) ([]models.Notification, error)
	Count(ctx context.Context) (int, error)
}

type notificationRepo struct {
	fs *firestore.Client
}

// NewNotificationRepo returns a Firestore-backed NotificationRepository.
func NewNotificationRepo(fs *firestore.Client) NotificationRepository {
	return &notificationRepo{fs: fs}
}

func (r *notificationRepo) Create(ctx context.Context, n *models.Notification) error {
	_, err := r.fs.Collection(notificationsCollection).Doc(n.ID).Set(ctx, n)
	if err != nil {
		return fmt.Errorf("create notification %s: %w", n.ID, err)
	}
	return nil
}

func (r *notificationRepo) Get(ctx context.Context, id string) (*models.Notification, error) {
	doc, err := r.fs.Collection(notificationsCollection).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get notification %s: %w", id, err)
	}

	var n models.Notification
	if err := doc.DataTo(&n); err != nil {
		return nil, fmt.Errorf("decode notification %s: %w", id, err)
	}
	return &n, nil
}

func (r *notificationRepo) List(ctx context.Context) ([]models.Notification, error) {
	docs, err := r.fs.Collection(notificationsCollection).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}

	ns := make([]models.Notification, 0, len(docs))
	for _, doc := range docs {
		var n models.Notification
		if err := doc.DataTo(&n); err != nil {
			return nil, fmt.Errorf("decode notification %s: %w", doc.Ref.ID, err)
		}
		ns = append(ns, n)
	}
	return ns, nil
}

func (r *notificationRepo) Delete(ctx context.Context, id string) error {
	_, err := r.fs.Collection(notificationsCollection).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ErrNotFound
		}
		return fmt.Errorf("delete notification %s: %w", id, err)
	}

	_, err = r.fs.Collection(notificationsCollection).Doc(id).Delete(ctx)
	if err != nil {
		return fmt.Errorf("delete notification %s: %w", id, err)
	}
	return nil
}

// ListMatching fetches all notifications and filters in memory. Firestore
// does not support OR queries on multiple fields cleanly, and the collection
// is expected to remain small, so client-side filtering is acceptable.
func (r *notificationRepo) ListMatching(ctx context.Context, isoCode, event string) ([]models.Notification, error) {
	all, err := r.List(ctx)
	if err != nil {
		return nil, err
	}

	var matched []models.Notification
	for _, n := range all {
		if n.Event != event {
			continue
		}
		if n.Country == "" || n.Country == isoCode {
			matched = append(matched, n)
		}
	}
	return matched, nil
}

func (r *notificationRepo) Count(ctx context.Context) (int, error) {
	docs, err := r.fs.Collection(notificationsCollection).Documents(ctx).GetAll()
	if err != nil {
		return 0, fmt.Errorf("count notifications: %w", err)
	}
	return len(docs), nil
}
