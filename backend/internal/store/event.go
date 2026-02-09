package store

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/otiai10/namazu/internal/source"
)

// EventRecord represents an event stored in Firestore
type EventRecord struct {
	ID            string    `firestore:"-"`
	Type          string    `firestore:"type"`
	Source        string    `firestore:"source"`
	Severity      int       `firestore:"severity"`
	AffectedAreas []string  `firestore:"affectedAreas"`
	OccurredAt    time.Time `firestore:"occurredAt"`
	ReceivedAt    time.Time `firestore:"receivedAt"`
	RawJSON       string    `firestore:"rawJson"`
	CreatedAt     time.Time `firestore:"createdAt"`
}

// EventRepository defines the interface for event storage operations
type EventRepository interface {
	// Create stores a new event and returns its ID
	Create(ctx context.Context, event EventRecord) (string, error)

	// Get retrieves an event by ID
	Get(ctx context.Context, id string) (*EventRecord, error)

	// List retrieves events ordered by occurredAt descending with pagination
	List(ctx context.Context, limit int, startAfter *time.Time) ([]EventRecord, error)
}

// FirestoreEventRepository implements EventRepository using Firestore
type FirestoreEventRepository struct {
	client     *firestore.Client
	collection string
}

// Compile-time interface check
var _ EventRepository = (*FirestoreEventRepository)(nil)

// NewFirestoreEventRepository creates a new FirestoreEventRepository
func NewFirestoreEventRepository(client *firestore.Client) *FirestoreEventRepository {
	return &FirestoreEventRepository{
		client:     client,
		collection: "events",
	}
}

// Create stores a new event in Firestore
func (r *FirestoreEventRepository) Create(ctx context.Context, event EventRecord) (string, error) {
	if r.client == nil {
		return "", fmt.Errorf("firestore client is nil")
	}

	// Set CreatedAt if not already set
	record := createEventRecord(event)

	var docRef *firestore.DocumentRef
	if event.ID != "" {
		// Use provided ID as document ID
		docRef = r.client.Collection(r.collection).Doc(event.ID)
		_, err := docRef.Set(ctx, record)
		if err != nil {
			return "", fmt.Errorf("failed to create event: %w", err)
		}
		return event.ID, nil
	}

	// Auto-generate document ID
	docRef, _, err := r.client.Collection(r.collection).Add(ctx, record)
	if err != nil {
		return "", fmt.Errorf("failed to create event: %w", err)
	}

	return docRef.ID, nil
}

// createEventRecord creates a new EventRecord with CreatedAt set
func createEventRecord(event EventRecord) EventRecord {
	return EventRecord{
		ID:            event.ID,
		Type:          event.Type,
		Source:        event.Source,
		Severity:      event.Severity,
		AffectedAreas: copyStringSlice(event.AffectedAreas),
		OccurredAt:    event.OccurredAt,
		ReceivedAt:    event.ReceivedAt,
		RawJSON:       event.RawJSON,
		CreatedAt:     time.Now(),
	}
}

// copyStringSlice creates a defensive copy of a string slice
func copyStringSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	result := make([]string, len(s))
	copy(result, s)
	return result
}

// Get retrieves an event by ID from Firestore
func (r *FirestoreEventRepository) Get(ctx context.Context, id string) (*EventRecord, error) {
	if r.client == nil {
		return nil, fmt.Errorf("firestore client is nil")
	}

	if id == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	docSnap, err := r.client.Collection(r.collection).Doc(id).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	var record EventRecord
	if err := docSnap.DataTo(&record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	record.ID = docSnap.Ref.ID

	return &record, nil
}

// List retrieves events ordered by occurredAt descending with pagination
func (r *FirestoreEventRepository) List(ctx context.Context, limit int, startAfter *time.Time) ([]EventRecord, error) {
	if r.client == nil {
		return nil, fmt.Errorf("firestore client is nil")
	}

	if limit <= 0 {
		limit = 10 // Default limit
	}

	query := r.client.Collection(r.collection).
		OrderBy("occurredAt", firestore.Desc).
		Limit(limit)

	if startAfter != nil {
		query = query.StartAfter(*startAfter)
	}

	iter := query.Documents(ctx)
	defer iter.Stop()

	records := make([]EventRecord, 0, limit)
	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate events: %w", err)
		}

		var record EventRecord
		if err := docSnap.DataTo(&record); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event: %w", err)
		}
		record.ID = docSnap.Ref.ID
		records = append(records, record)
	}

	return records, nil
}

// EventFromSource converts a source.Event to EventRecord
func EventFromSource(event source.Event) EventRecord {
	return EventRecord{
		ID:            event.GetID(),
		Type:          string(event.GetType()),
		Source:        event.GetSource(),
		Severity:      event.GetSeverity(),
		AffectedAreas: copyStringSlice(event.GetAffectedAreas()),
		OccurredAt:    event.GetOccurredAt(),
		ReceivedAt:    event.GetReceivedAt(),
		RawJSON:       event.GetRawJSON(),
	}
}
