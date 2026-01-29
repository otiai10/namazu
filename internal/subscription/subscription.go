package subscription

import (
	"context"
)

// Subscription represents a notification subscription
type Subscription struct {
	Name     string
	Delivery DeliveryConfig
	Filter   *FilterConfig
}

// DeliveryConfig represents how to deliver notifications
type DeliveryConfig struct {
	Type   string // "webhook" | "email" | "slack"
	URL    string
	Secret string
}

// FilterConfig represents event filtering conditions
type FilterConfig struct {
	MinScale    int
	Prefectures []string
}

// Repository defines the interface for subscription storage
type Repository interface {
	// List returns all active subscriptions
	List(ctx context.Context) ([]Subscription, error)
}
