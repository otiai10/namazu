package subscription

import (
	"context"
)

// Subscription represents a notification subscription
type Subscription struct {
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name"`
	Delivery DeliveryConfig `json:"delivery"`
	Filter   *FilterConfig  `json:"filter,omitempty"`
}

// DeliveryConfig represents how to deliver notifications
type DeliveryConfig struct {
	Type   string `json:"type"` // "webhook" | "email" | "slack"
	URL    string `json:"url,omitempty"`
	Secret string `json:"secret,omitempty"`
}

// FilterConfig represents event filtering conditions
type FilterConfig struct {
	MinScale    int      `json:"min_scale,omitempty"`
	Prefectures []string `json:"prefectures,omitempty"`
}

// Repository defines the interface for subscription storage
type Repository interface {
	// List returns all active subscriptions
	List(ctx context.Context) ([]Subscription, error)

	// Create creates a new subscription and returns its ID
	Create(ctx context.Context, sub Subscription) (string, error)

	// Get retrieves a subscription by ID
	// Returns nil and no error if not found
	Get(ctx context.Context, id string) (*Subscription, error)

	// Update updates an existing subscription
	// Returns error if subscription does not exist
	Update(ctx context.Context, id string, sub Subscription) error

	// Delete removes a subscription by ID
	// Returns error if subscription does not exist
	Delete(ctx context.Context, id string) error
}
