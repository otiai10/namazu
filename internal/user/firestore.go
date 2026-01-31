package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// collectionName is the Firestore collection for users
	collectionName = "users"
)

// Error definitions
var (
	// ErrNotFound is returned when a user is not found
	ErrNotFound = errors.New("user not found")

	// ErrDuplicateUID is returned when trying to create a user with an existing UID
	ErrDuplicateUID = errors.New("user with this UID already exists")

	// ErrProviderExists is returned when trying to add a provider that already exists
	ErrProviderExists = errors.New("provider already linked to user")

	// ErrProviderNotFound is returned when trying to remove a provider that doesn't exist
	ErrProviderNotFound = errors.New("provider not found for user")
)

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

// Create creates a new user and returns its document ID
//
// Parameters:
//   - ctx: Context for cancellation control
//   - user: User to create
//
// Returns:
//   - ID of the created user document
//   - Error if Firestore operation fails or UID already exists
func (r *FirestoreRepository) Create(ctx context.Context, user User) (string, error) {
	// Check if user with same UID already exists
	existingUser, err := r.GetByUID(ctx, user.UID)
	if err != nil {
		return "", fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return "", ErrDuplicateUID
	}

	data := userToMap(user)

	docRef, _, err := r.client.Collection(collectionName).Add(ctx, data)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	return docRef.ID, nil
}

// Get retrieves a user by document ID
//
// Parameters:
//   - ctx: Context for cancellation control
//   - id: User document ID to retrieve
//
// Returns:
//   - Pointer to the user (nil if not found)
//   - Error if Firestore operation fails (nil for not found)
func (r *FirestoreRepository) Get(ctx context.Context, id string) (*User, error) {
	doc, err := r.client.Collection(collectionName).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user, err := documentToUser(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to convert document: %w", err)
	}

	return &user, nil
}

// GetByUID retrieves a user by Identity Platform UID
//
// Parameters:
//   - ctx: Context for cancellation control
//   - uid: Identity Platform UID to search for
//
// Returns:
//   - Pointer to the user (nil if not found)
//   - Error if Firestore operation fails (nil for not found)
func (r *FirestoreRepository) GetByUID(ctx context.Context, uid string) (*User, error) {
	docs, err := r.client.Collection(collectionName).
		Where("uid", "==", uid).
		Limit(1).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to query user by UID: %w", err)
	}

	if len(docs) == 0 {
		return nil, nil
	}

	user, err := documentToUser(docs[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert document: %w", err)
	}

	return &user, nil
}

// Update updates an existing user
//
// Parameters:
//   - ctx: Context for cancellation control
//   - id: User document ID to update
//   - user: New user data
//
// Returns:
//   - Error if user not found or Firestore operation fails
func (r *FirestoreRepository) Update(ctx context.Context, id string, user User) error {
	docRef := r.client.Collection(collectionName).Doc(id)

	// Check if document exists
	_, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ErrNotFound
		}
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	data := userToMap(user)
	_, err = docRef.Set(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdateLastLogin updates the LastLoginAt field
//
// Parameters:
//   - ctx: Context for cancellation control
//   - id: User document ID to update
//   - t: New last login time
//
// Returns:
//   - Error if user not found or Firestore operation fails
func (r *FirestoreRepository) UpdateLastLogin(ctx context.Context, id string, t time.Time) error {
	docRef := r.client.Collection(collectionName).Doc(id)

	// Check if document exists
	_, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ErrNotFound
		}
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	_, err = docRef.Update(ctx, []firestore.Update{
		{Path: "lastLoginAt", Value: t},
		{Path: "updatedAt", Value: time.Now().UTC()},
	})
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// AddProvider adds a linked provider to a user
//
// Parameters:
//   - ctx: Context for cancellation control
//   - id: User document ID
//   - provider: Provider to add
//
// Returns:
//   - Error if user not found, provider already exists, or Firestore operation fails
func (r *FirestoreRepository) AddProvider(ctx context.Context, id string, provider LinkedProvider) error {
	docRef := r.client.Collection(collectionName).Doc(id)

	// Get current user to check existing providers
	doc, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ErrNotFound
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	user, err := documentToUser(doc)
	if err != nil {
		return fmt.Errorf("failed to convert document: %w", err)
	}

	// Check if provider already exists
	if findProviderIndex(user.Providers, provider.ProviderID) >= 0 {
		return ErrProviderExists
	}

	// Add provider using array union (immutable operation)
	providerMap := providerToMap(provider)
	_, err = docRef.Update(ctx, []firestore.Update{
		{Path: "providers", Value: firestore.ArrayUnion(providerMap)},
		{Path: "updatedAt", Value: time.Now().UTC()},
	})
	if err != nil {
		return fmt.Errorf("failed to add provider: %w", err)
	}

	return nil
}

// RemoveProvider removes a linked provider from a user
//
// Parameters:
//   - ctx: Context for cancellation control
//   - id: User document ID
//   - providerID: Provider ID to remove (e.g., "google.com")
//
// Returns:
//   - Error if user not found, provider not found, or Firestore operation fails
func (r *FirestoreRepository) RemoveProvider(ctx context.Context, id string, providerID string) error {
	docRef := r.client.Collection(collectionName).Doc(id)

	// Get current user to find the provider
	doc, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ErrNotFound
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	user, err := documentToUser(doc)
	if err != nil {
		return fmt.Errorf("failed to convert document: %w", err)
	}

	// Find provider index
	idx := findProviderIndex(user.Providers, providerID)
	if idx < 0 {
		return ErrProviderNotFound
	}

	// Create new providers slice without the removed provider (immutable)
	newProviders := make([]map[string]any, 0, len(user.Providers)-1)
	for i, p := range user.Providers {
		if i != idx {
			newProviders = append(newProviders, providerToMap(p))
		}
	}

	_, err = docRef.Update(ctx, []firestore.Update{
		{Path: "providers", Value: newProviders},
		{Path: "updatedAt", Value: time.Now().UTC()},
	})
	if err != nil {
		return fmt.Errorf("failed to remove provider: %w", err)
	}

	return nil
}

// userToMap converts a User to a map for Firestore storage
func userToMap(user User) map[string]any {
	providers := make([]map[string]any, len(user.Providers))
	for i, p := range user.Providers {
		providers[i] = providerToMap(p)
	}

	data := map[string]any{
		"uid":         user.UID,
		"email":       user.Email,
		"displayName": user.DisplayName,
		"pictureUrl":  user.PictureURL,
		"plan":        user.Plan,
		"providers":   providers,
		"createdAt":   user.CreatedAt,
		"updatedAt":   user.UpdatedAt,
		"lastLoginAt": user.LastLoginAt,
	}

	// Include Stripe fields if set
	if user.StripeCustomerID != "" {
		data["stripeCustomerId"] = user.StripeCustomerID
	}
	if user.SubscriptionID != "" {
		data["subscriptionId"] = user.SubscriptionID
	}
	if user.SubscriptionStatus != "" {
		data["subscriptionStatus"] = user.SubscriptionStatus
	}
	if !user.SubscriptionEndsAt.IsZero() {
		data["subscriptionEndsAt"] = user.SubscriptionEndsAt
	}

	return data
}

// providerToMap converts a LinkedProvider to a map for Firestore storage
func providerToMap(provider LinkedProvider) map[string]any {
	return map[string]any{
		"providerId":  provider.ProviderID,
		"subject":     provider.Subject,
		"email":       provider.Email,
		"displayName": provider.DisplayName,
		"linkedAt":    provider.LinkedAt,
	}
}

// documentToUser converts a Firestore document to a User
func documentToUser(doc *firestore.DocumentSnapshot) (User, error) {
	data := doc.Data()

	user := User{
		ID: doc.Ref.ID,
	}

	if uid, ok := data["uid"].(string); ok {
		user.UID = uid
	}
	if email, ok := data["email"].(string); ok {
		user.Email = email
	}
	if displayName, ok := data["displayName"].(string); ok {
		user.DisplayName = displayName
	}
	if pictureURL, ok := data["pictureUrl"].(string); ok {
		user.PictureURL = pictureURL
	}
	if plan, ok := data["plan"].(string); ok {
		user.Plan = plan
	}
	if createdAt, ok := data["createdAt"].(time.Time); ok {
		user.CreatedAt = createdAt
	}
	if updatedAt, ok := data["updatedAt"].(time.Time); ok {
		user.UpdatedAt = updatedAt
	}
	if lastLoginAt, ok := data["lastLoginAt"].(time.Time); ok {
		user.LastLoginAt = lastLoginAt
	}

	// Parse Stripe fields
	if stripeCustomerID, ok := data["stripeCustomerId"].(string); ok {
		user.StripeCustomerID = stripeCustomerID
	}
	if subscriptionID, ok := data["subscriptionId"].(string); ok {
		user.SubscriptionID = subscriptionID
	}
	if subscriptionStatus, ok := data["subscriptionStatus"].(string); ok {
		user.SubscriptionStatus = subscriptionStatus
	}
	if subscriptionEndsAt, ok := data["subscriptionEndsAt"].(time.Time); ok {
		user.SubscriptionEndsAt = subscriptionEndsAt
	}

	// Parse providers
	if providers, ok := data["providers"].([]any); ok {
		user.Providers = make([]LinkedProvider, 0, len(providers))
		for _, p := range providers {
			if providerMap, ok := p.(map[string]any); ok {
				provider := mapToProvider(providerMap)
				user.Providers = append(user.Providers, provider)
			}
		}
	}

	return user, nil
}

// mapToProvider converts a map to a LinkedProvider
func mapToProvider(data map[string]any) LinkedProvider {
	provider := LinkedProvider{}

	if providerID, ok := data["providerId"].(string); ok {
		provider.ProviderID = providerID
	}
	if subject, ok := data["subject"].(string); ok {
		provider.Subject = subject
	}
	if email, ok := data["email"].(string); ok {
		provider.Email = email
	}
	if displayName, ok := data["displayName"].(string); ok {
		provider.DisplayName = displayName
	}
	if linkedAt, ok := data["linkedAt"].(time.Time); ok {
		provider.LinkedAt = linkedAt
	}

	return provider
}

// findProviderIndex finds the index of a provider by its ID
// Returns -1 if not found
func findProviderIndex(providers []LinkedProvider, providerID string) int {
	for i, p := range providers {
		if p.ProviderID == providerID {
			return i
		}
	}
	return -1
}

// GetByStripeCustomerID retrieves a user by Stripe customer ID
//
// Parameters:
//   - ctx: Context for cancellation control
//   - customerID: Stripe customer ID to search for
//
// Returns:
//   - Pointer to the user (nil if not found)
//   - Error if Firestore operation fails (nil for not found)
func (r *FirestoreRepository) GetByStripeCustomerID(ctx context.Context, customerID string) (*User, error) {
	docs, err := r.client.Collection(collectionName).
		Where("stripeCustomerId", "==", customerID).
		Limit(1).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to query user by Stripe customer ID: %w", err)
	}

	if len(docs) == 0 {
		return nil, nil
	}

	user, err := documentToUser(docs[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert document: %w", err)
	}

	return &user, nil
}
