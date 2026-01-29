package subscription

import (
	"context"
	"testing"

	"github.com/ayanel/namazu/internal/config"
)

func TestNewStaticRepository(t *testing.T) {
	t.Run("creates repository from config", func(t *testing.T) {
		cfg := &config.Config{
			Subscriptions: []config.SubscriptionConfig{
				{
					Name: "Webhook 1",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://webhook1.example.com",
						Secret: "secret1",
					},
				},
				{
					Name: "Webhook 2",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://webhook2.example.com",
						Secret: "secret2",
					},
				},
			},
		}

		repo := NewStaticRepository(cfg)

		if repo == nil {
			t.Fatal("NewStaticRepository returned nil")
		}
		if len(repo.subscriptions) != 2 {
			t.Errorf("Expected 2 subscriptions, got %d", len(repo.subscriptions))
		}
	})

	t.Run("converts config fields correctly", func(t *testing.T) {
		cfg := &config.Config{
			Subscriptions: []config.SubscriptionConfig{
				{
					Name: "Test Webhook",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://test.example.com",
						Secret: "test-secret",
					},
				},
			},
		}

		repo := NewStaticRepository(cfg)
		sub := repo.subscriptions[0]

		if sub.Name != "Test Webhook" {
			t.Errorf("Expected name 'Test Webhook', got '%s'", sub.Name)
		}
		if sub.Delivery.Type != "webhook" {
			t.Errorf("Expected delivery type 'webhook', got '%s'", sub.Delivery.Type)
		}
		if sub.Delivery.URL != "https://test.example.com" {
			t.Errorf("Expected URL 'https://test.example.com', got '%s'", sub.Delivery.URL)
		}
		if sub.Delivery.Secret != "test-secret" {
			t.Errorf("Expected secret 'test-secret', got '%s'", sub.Delivery.Secret)
		}
	})

	t.Run("handles filter config when present", func(t *testing.T) {
		cfg := &config.Config{
			Subscriptions: []config.SubscriptionConfig{
				{
					Name: "Filtered Webhook",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://filtered.example.com",
						Secret: "secret",
					},
					Filter: &config.FilterConfig{
						MinScale:    3,
						Prefectures: []string{"Tokyo", "Osaka"},
					},
				},
			},
		}

		repo := NewStaticRepository(cfg)
		sub := repo.subscriptions[0]

		if sub.Filter == nil {
			t.Fatal("Expected filter to be set, got nil")
		}
		if sub.Filter.MinScale != 3 {
			t.Errorf("Expected MinScale 3, got %d", sub.Filter.MinScale)
		}
		if len(sub.Filter.Prefectures) != 2 {
			t.Errorf("Expected 2 prefectures, got %d", len(sub.Filter.Prefectures))
		}
		if sub.Filter.Prefectures[0] != "Tokyo" {
			t.Errorf("Expected first prefecture 'Tokyo', got '%s'", sub.Filter.Prefectures[0])
		}
	})

	t.Run("handles filter config when absent", func(t *testing.T) {
		cfg := &config.Config{
			Subscriptions: []config.SubscriptionConfig{
				{
					Name: "Unfiltered Webhook",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://unfiltered.example.com",
						Secret: "secret",
					},
					Filter: nil,
				},
			},
		}

		repo := NewStaticRepository(cfg)
		sub := repo.subscriptions[0]

		if sub.Filter != nil {
			t.Error("Expected filter to be nil, got non-nil value")
		}
	})

	t.Run("handles empty subscriptions", func(t *testing.T) {
		cfg := &config.Config{
			Subscriptions: []config.SubscriptionConfig{},
		}

		repo := NewStaticRepository(cfg)

		if repo == nil {
			t.Fatal("NewStaticRepository returned nil")
		}
		if len(repo.subscriptions) != 0 {
			t.Errorf("Expected 0 subscriptions, got %d", len(repo.subscriptions))
		}
	})
}

func TestStaticRepository_List(t *testing.T) {
	t.Run("returns all subscriptions", func(t *testing.T) {
		cfg := &config.Config{
			Subscriptions: []config.SubscriptionConfig{
				{
					Name: "Webhook 1",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://webhook1.example.com",
						Secret: "secret1",
					},
				},
				{
					Name: "Webhook 2",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://webhook2.example.com",
						Secret: "secret2",
					},
				},
			},
		}

		repo := NewStaticRepository(cfg)
		ctx := context.Background()

		subs, err := repo.List(ctx)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(subs) != 2 {
			t.Errorf("Expected 2 subscriptions, got %d", len(subs))
		}
	})

	t.Run("returns copy to prevent mutation", func(t *testing.T) {
		cfg := &config.Config{
			Subscriptions: []config.SubscriptionConfig{
				{
					Name: "Original Webhook",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://original.example.com",
						Secret: "original-secret",
					},
				},
			},
		}

		repo := NewStaticRepository(cfg)
		ctx := context.Background()

		// Get list and modify it
		subs1, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		subs1[0].Name = "Modified Webhook"

		// Get list again
		subs2, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Original should be unchanged
		if subs2[0].Name != "Original Webhook" {
			t.Errorf("Expected name 'Original Webhook', got '%s' (mutation detected)", subs2[0].Name)
		}
	})

	t.Run("returns empty slice for empty subscriptions", func(t *testing.T) {
		cfg := &config.Config{
			Subscriptions: []config.SubscriptionConfig{},
		}

		repo := NewStaticRepository(cfg)
		ctx := context.Background()

		subs, err := repo.List(ctx)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(subs) != 0 {
			t.Errorf("Expected 0 subscriptions, got %d", len(subs))
		}
	})

	t.Run("context can be nil", func(t *testing.T) {
		cfg := &config.Config{
			Subscriptions: []config.SubscriptionConfig{
				{
					Name: "Webhook 1",
					Delivery: config.DeliveryConfig{
						Type:   "webhook",
						URL:    "https://webhook1.example.com",
						Secret: "secret1",
					},
				},
			},
		}

		repo := NewStaticRepository(cfg)

		// Should work even with nil context (static doesn't use it)
		subs, err := repo.List(context.Background())

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(subs) != 1 {
			t.Errorf("Expected 1 subscription, got %d", len(subs))
		}
	})
}
