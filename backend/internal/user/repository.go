package user

import (
	"context"
	"time"
)

// Repository defines the interface for user storage operations
type Repository interface {
	// Create creates a new user (first login)
	//
	// Parameters:
	//   - ctx: Context for cancellation control
	//   - user: User to create
	//
	// Returns:
	//   - ID of the created user document
	//   - Error if Firestore operation fails or UID already exists
	Create(ctx context.Context, user User) (string, error)

	// Get retrieves a user by document ID
	//
	// Parameters:
	//   - ctx: Context for cancellation control
	//   - id: User document ID to retrieve
	//
	// Returns:
	//   - Pointer to the user (nil if not found)
	//   - Error if Firestore operation fails (nil for not found)
	Get(ctx context.Context, id string) (*User, error)

	// GetByUID retrieves a user by Identity Platform UID
	//
	// Parameters:
	//   - ctx: Context for cancellation control
	//   - uid: Identity Platform UID to search for
	//
	// Returns:
	//   - Pointer to the user (nil if not found)
	//   - Error if Firestore operation fails (nil for not found)
	GetByUID(ctx context.Context, uid string) (*User, error)

	// Update updates an existing user
	//
	// Parameters:
	//   - ctx: Context for cancellation control
	//   - id: User document ID to update
	//   - user: New user data
	//
	// Returns:
	//   - Error if user not found or Firestore operation fails
	Update(ctx context.Context, id string, user User) error

	// UpdateLastLogin updates the LastLoginAt field
	//
	// Parameters:
	//   - ctx: Context for cancellation control
	//   - id: User document ID to update
	//   - t: New last login time
	//
	// Returns:
	//   - Error if user not found or Firestore operation fails
	UpdateLastLogin(ctx context.Context, id string, t time.Time) error

	// AddProvider adds a linked provider to a user
	//
	// Parameters:
	//   - ctx: Context for cancellation control
	//   - id: User document ID
	//   - provider: Provider to add
	//
	// Returns:
	//   - Error if user not found, provider already exists, or Firestore operation fails
	AddProvider(ctx context.Context, id string, provider LinkedProvider) error

	// RemoveProvider removes a linked provider from a user
	//
	// Parameters:
	//   - ctx: Context for cancellation control
	//   - id: User document ID
	//   - providerID: Provider ID to remove (e.g., "google.com")
	//
	// Returns:
	//   - Error if user not found, provider not found, or Firestore operation fails
	RemoveProvider(ctx context.Context, id string, providerID string) error

	// GetByStripeCustomerID retrieves a user by Stripe customer ID
	//
	// Parameters:
	//   - ctx: Context for cancellation control
	//   - customerID: Stripe customer ID to search for
	//
	// Returns:
	//   - Pointer to the user (nil if not found)
	//   - Error if Firestore operation fails (nil for not found)
	GetByStripeCustomerID(ctx context.Context, customerID string) (*User, error)
}
