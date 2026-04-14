package firebase

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"envdash/internal/models"
)

const registrationsCollection = "registrations"

// ErrNotFound is returned when a document does not exist.
var ErrNotFound = errors.New("not found")

// RegistrationRepository defines persistence operations for dashboard configurations.
type RegistrationRepository interface {
	Create(ctx context.Context, r *models.Registration) error
	Get(ctx context.Context, id string) (*models.Registration, error)
	List(ctx context.Context) ([]models.Registration, error)
	Update(ctx context.Context, r *models.Registration) error
	Delete(ctx context.Context, id string) error
}

type registrationRepo struct {
	fs         *firestore.Client
	collection string
}

// NewRegistrationRepo returns a Firestore-backed RegistrationRepository.
func NewRegistrationRepo(fs *firestore.Client) RegistrationRepository {
	return &registrationRepo{fs: fs, collection: registrationsCollection}
}

// Create writes a new registration document. The document ID is taken from reg.ID.
func (r *registrationRepo) Create(ctx context.Context, reg *models.Registration) error {
	_, err := r.fs.Collection(r.collection).Doc(reg.ID).Set(ctx, reg)
	if err != nil {
		return fmt.Errorf("create registration %s: %w", reg.ID, err)
	}
	return nil
}

// Get fetches a single registration by ID. Returns ErrNotFound if no document exists.
func (r *registrationRepo) Get(ctx context.Context, id string) (*models.Registration, error) {
	doc, err := r.fs.Collection(r.collection).Doc(id).Get(ctx)
	if err != nil {
		// Translate the gRPC NotFound code into our sentinel error so callers
		// don't need to import gRPC packages to detect missing documents.
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get registration %s: %w", id, err)
	}

	var reg models.Registration
	if err := doc.DataTo(&reg); err != nil {
		return nil, fmt.Errorf("decode registration %s: %w", id, err)
	}
	return &reg, nil
}

// List returns all registration documents in the collection.
func (r *registrationRepo) List(ctx context.Context) ([]models.Registration, error) {
	docs, err := r.fs.Collection(r.collection).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("list registrations: %w", err)
	}

	// Pre-allocate the slice to the known document count to avoid reallocations.
	regs := make([]models.Registration, 0, len(docs))
	for _, doc := range docs {
		var reg models.Registration
		if err := doc.DataTo(&reg); err != nil {
			return nil, fmt.Errorf("decode registration %s: %w", doc.Ref.ID, err)
		}
		regs = append(regs, reg)
	}
	return regs, nil
}

// Update overwrites an existing registration document with the new value.
func (r *registrationRepo) Update(ctx context.Context, reg *models.Registration) error {
	_, err := r.fs.Collection(r.collection).Doc(reg.ID).Set(ctx, reg)
	if err != nil {
		return fmt.Errorf("update registration %s: %w", reg.ID, err)
	}
	return nil
}

// Delete removes a registration by ID. Returns ErrNotFound if no document exists.
func (r *registrationRepo) Delete(ctx context.Context, id string) error {
	// Firestore's Delete() succeeds silently even if the document doesn't exist,
	// so we must read first to return a meaningful ErrNotFound to the caller.
	_, err := r.fs.Collection(r.collection).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ErrNotFound
		}
		return fmt.Errorf("delete registration %s: %w", id, err)
	}

	_, err = r.fs.Collection(r.collection).Doc(id).Delete(ctx)
	if err != nil {
		return fmt.Errorf("delete registration %s: %w", id, err)
	}
	return nil
}
