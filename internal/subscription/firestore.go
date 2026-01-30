package subscription

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// collectionName is the Firestore collection for subscriptions
	collectionName = "subscriptions"
)

// ErrNotFound is returned when a subscription is not found
var ErrNotFound = errors.New("subscription not found")

// FirestoreRepository implements Repository interface using Firestore
type FirestoreRepository struct {
	client *firestore.Client
}

// Ensure FirestoreRepository implements Repository interface
var _ Repository = (*FirestoreRepository)(nil)

// NewFirestoreRepository creates a new FirestoreRepository
//
// Parameters:
//   - client: Firestore client instance
//
// Returns:
//   - FirestoreRepository instance
func NewFirestoreRepository(client *firestore.Client) *FirestoreRepository {
	return &FirestoreRepository{
		client: client,
	}
}

// List returns all subscriptions from Firestore
//
// Parameters:
//   - ctx: Context for cancellation control
//
// Returns:
//   - Slice of all subscriptions (copy to prevent mutation)
//   - Error if Firestore operation fails
func (r *FirestoreRepository) List(ctx context.Context) ([]Subscription, error) {
	docs, err := r.client.Collection(collectionName).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	subscriptions := make([]Subscription, 0, len(docs))
	for _, doc := range docs {
		sub, err := documentToSubscription(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to convert document %s: %w", doc.Ref.ID, err)
		}
		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}

// ListByUserID returns all subscriptions for a specific user
//
// Parameters:
//   - ctx: Context for cancellation control
//   - userID: User ID to filter subscriptions
//
// Returns:
//   - Slice of subscriptions belonging to the user
//   - Error if Firestore operation fails
func (r *FirestoreRepository) ListByUserID(ctx context.Context, userID string) ([]Subscription, error) {
	docs, err := r.client.Collection(collectionName).
		Where("userId", "==", userID).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions by user: %w", err)
	}

	subscriptions := make([]Subscription, 0, len(docs))
	for _, doc := range docs {
		sub, err := documentToSubscription(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to convert document %s: %w", doc.Ref.ID, err)
		}
		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}

// Create creates a new subscription and returns its ID
//
// Parameters:
//   - ctx: Context for cancellation control
//   - sub: Subscription to create
//
// Returns:
//   - ID of the created subscription
//   - Error if Firestore operation fails
func (r *FirestoreRepository) Create(ctx context.Context, sub Subscription) (string, error) {
	data := subscriptionToMap(sub)

	docRef, _, err := r.client.Collection(collectionName).Add(ctx, data)
	if err != nil {
		return "", fmt.Errorf("failed to create subscription: %w", err)
	}

	return docRef.ID, nil
}

// Get retrieves a subscription by ID
//
// Parameters:
//   - ctx: Context for cancellation control
//   - id: Subscription ID to retrieve
//
// Returns:
//   - Pointer to the subscription (nil if not found)
//   - Error if Firestore operation fails (nil for not found)
func (r *FirestoreRepository) Get(ctx context.Context, id string) (*Subscription, error) {
	doc, err := r.client.Collection(collectionName).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	sub, err := documentToSubscription(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to convert document: %w", err)
	}

	return &sub, nil
}

// Update updates an existing subscription
//
// Parameters:
//   - ctx: Context for cancellation control
//   - id: Subscription ID to update
//   - sub: New subscription data
//
// Returns:
//   - Error if subscription not found or Firestore operation fails
func (r *FirestoreRepository) Update(ctx context.Context, id string, sub Subscription) error {
	docRef := r.client.Collection(collectionName).Doc(id)

	// Check if document exists
	_, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ErrNotFound
		}
		return fmt.Errorf("failed to check subscription existence: %w", err)
	}

	data := subscriptionToMap(sub)
	_, err = docRef.Set(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	return nil
}

// Delete removes a subscription by ID
//
// Parameters:
//   - ctx: Context for cancellation control
//   - id: Subscription ID to delete
//
// Returns:
//   - Error if subscription not found or Firestore operation fails
func (r *FirestoreRepository) Delete(ctx context.Context, id string) error {
	docRef := r.client.Collection(collectionName).Doc(id)

	// Check if document exists
	_, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ErrNotFound
		}
		return fmt.Errorf("failed to check subscription existence: %w", err)
	}

	_, err = docRef.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete subscription: %w", err)
	}

	return nil
}

// subscriptionToMap converts a Subscription to a map for Firestore storage
func subscriptionToMap(sub Subscription) map[string]interface{} {
	data := map[string]interface{}{
		"userId": sub.UserID,
		"name":   sub.Name,
		"delivery": map[string]interface{}{
			"type":   sub.Delivery.Type,
			"url":    sub.Delivery.URL,
			"secret": sub.Delivery.Secret,
		},
	}

	if sub.Filter != nil {
		data["filter"] = map[string]interface{}{
			"minScale":    sub.Filter.MinScale,
			"prefectures": sub.Filter.Prefectures,
		}
	}

	return data
}

// documentToSubscription converts a Firestore document to a Subscription
func documentToSubscription(doc *firestore.DocumentSnapshot) (Subscription, error) {
	data := doc.Data()

	sub := Subscription{
		ID: doc.Ref.ID,
	}

	if userID, ok := data["userId"].(string); ok {
		sub.UserID = userID
	}

	if name, ok := data["name"].(string); ok {
		sub.Name = name
	}

	if delivery, ok := data["delivery"].(map[string]interface{}); ok {
		if deliveryType, ok := delivery["type"].(string); ok {
			sub.Delivery.Type = deliveryType
		}
		if url, ok := delivery["url"].(string); ok {
			sub.Delivery.URL = url
		}
		if secret, ok := delivery["secret"].(string); ok {
			sub.Delivery.Secret = secret
		}
	}

	if filter, ok := data["filter"].(map[string]interface{}); ok {
		sub.Filter = &FilterConfig{}
		if minScale, ok := filter["minScale"].(int64); ok {
			sub.Filter.MinScale = int(minScale)
		}
		if prefectures, ok := filter["prefectures"].([]interface{}); ok {
			sub.Filter.Prefectures = make([]string, 0, len(prefectures))
			for _, p := range prefectures {
				if pStr, ok := p.(string); ok {
					sub.Filter.Prefectures = append(sub.Filter.Prefectures, pStr)
				}
			}
		}
	}

	return sub, nil
}
