package subscription

import (
	"context"
	"errors"
	"fmt"

	"github.com/otiai10/namazu/internal/config"
)

// ErrReadOnly is returned when attempting write operations on StaticRepository
var ErrReadOnly = errors.New("static repository is read-only")

// StaticRepository is a repository that loads subscriptions from config.
// Used for Phase 1 (YAML-based configuration).
type StaticRepository struct {
	subscriptions []Subscription
}

// NewStaticRepository creates a new StaticRepository from config.
// It converts config.SubscriptionConfig to subscription.Subscription.
//
// Parameters:
//   - cfg: Application configuration containing subscriptions
//
// Returns:
//   - StaticRepository instance with subscriptions loaded from config
func NewStaticRepository(cfg *config.Config) *StaticRepository {
	subs := make([]Subscription, len(cfg.Subscriptions))
	for i, sub := range cfg.Subscriptions {
		subs[i] = Subscription{
			Name: sub.Name,
			Delivery: DeliveryConfig{
				Type:   sub.Delivery.Type,
				URL:    sub.Delivery.URL,
				Secret: sub.Delivery.Secret,
			},
		}
		if sub.Filter != nil {
			subs[i].Filter = &FilterConfig{
				MinScale:    sub.Filter.MinScale,
				Prefectures: sub.Filter.Prefectures,
			}
		}
	}
	return &StaticRepository{subscriptions: subs}
}

// Ensure StaticRepository implements Repository interface
var _ Repository = (*StaticRepository)(nil)

// List returns all subscriptions (static, loaded at startup).
// Returns a copy to prevent external mutation.
//
// Parameters:
//   - ctx: Context for cancellation control (not used in static repository)
//
// Returns:
//   - Copy of all subscriptions
//   - Error (always nil for static repository)
func (r *StaticRepository) List(ctx context.Context) ([]Subscription, error) {
	// Return a copy to prevent mutation
	result := make([]Subscription, len(r.subscriptions))
	copy(result, r.subscriptions)
	return result, nil
}

// Create is not supported for StaticRepository (read-only).
//
// Parameters:
//   - ctx: Context for cancellation control (not used)
//   - sub: Subscription to create (not used)
//
// Returns:
//   - Empty string
//   - ErrReadOnly error
func (r *StaticRepository) Create(ctx context.Context, sub Subscription) (string, error) {
	return "", ErrReadOnly
}

// Get retrieves a subscription by ID from the static list.
//
// Parameters:
//   - ctx: Context for cancellation control (not used)
//   - id: Subscription ID to retrieve
//
// Returns:
//   - Pointer to the subscription (nil if not found)
//   - Error (always nil for static repository)
func (r *StaticRepository) Get(ctx context.Context, id string) (*Subscription, error) {
	for _, sub := range r.subscriptions {
		if sub.ID == id {
			// Return a copy to prevent mutation
			result := Subscription{
				ID:     sub.ID,
				UserID: sub.UserID,
				Name:   sub.Name,
				Delivery: DeliveryConfig{
					Type:   sub.Delivery.Type,
					URL:    sub.Delivery.URL,
					Secret: sub.Delivery.Secret,
				},
			}
			if sub.Filter != nil {
				prefectures := make([]string, len(sub.Filter.Prefectures))
				copy(prefectures, sub.Filter.Prefectures)
				result.Filter = &FilterConfig{
					MinScale:    sub.Filter.MinScale,
					Prefectures: prefectures,
				}
			}
			return &result, nil
		}
	}
	return nil, nil
}

// Update is not supported for StaticRepository (read-only).
//
// Parameters:
//   - ctx: Context for cancellation control (not used)
//   - id: Subscription ID to update (not used)
//   - sub: New subscription data (not used)
//
// Returns:
//   - ErrReadOnly error
func (r *StaticRepository) Update(ctx context.Context, id string, sub Subscription) error {
	return fmt.Errorf("%w: cannot update subscription %s", ErrReadOnly, id)
}

// Delete is not supported for StaticRepository (read-only).
//
// Parameters:
//   - ctx: Context for cancellation control (not used)
//   - id: Subscription ID to delete (not used)
//
// Returns:
//   - ErrReadOnly error
func (r *StaticRepository) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("%w: cannot delete subscription %s", ErrReadOnly, id)
}

// ListByUserID returns an empty slice for StaticRepository.
// Static repository doesn't support user-scoped queries as subscriptions
// are loaded from configuration files without user association.
//
// Parameters:
//   - ctx: Context for cancellation control (not used)
//   - userID: User ID to filter (not used)
//
// Returns:
//   - Empty slice of subscriptions
//   - nil error (always succeeds)
func (r *StaticRepository) ListByUserID(ctx context.Context, userID string) ([]Subscription, error) {
	return []Subscription{}, nil
}
