package subscription

import (
	"context"

	"github.com/ayanel/namazu/internal/config"
)

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
