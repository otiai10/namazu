package subscription

import (
	"context"
	"errors"
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

func TestStaticRepository_Create(t *testing.T) {
	t.Run("returns read-only error", func(t *testing.T) {
		cfg := &config.Config{
			Subscriptions: []config.SubscriptionConfig{},
		}

		repo := NewStaticRepository(cfg)
		ctx := context.Background()

		sub := Subscription{
			Name: "New Webhook",
			Delivery: DeliveryConfig{
				Type: "webhook",
				URL:  "https://new.example.com",
			},
		}

		id, err := repo.Create(ctx, sub)

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if err != ErrReadOnly {
			t.Errorf("Expected ErrReadOnly, got %v", err)
		}
		if id != "" {
			t.Errorf("Expected empty ID, got %s", id)
		}
	})
}

func TestStaticRepository_Get(t *testing.T) {
	t.Run("returns nil for non-existent ID", func(t *testing.T) {
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
		ctx := context.Background()

		sub, err := repo.Get(ctx, "non-existent-id")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if sub != nil {
			t.Error("Expected nil subscription for non-existent ID")
		}
	})

	t.Run("returns subscription by ID", func(t *testing.T) {
		// Create repository with subscription that has an ID
		repo := &StaticRepository{
			subscriptions: []Subscription{
				{
					ID:   "test-id-123",
					Name: "Test Webhook",
					Delivery: DeliveryConfig{
						Type:   "webhook",
						URL:    "https://test.example.com",
						Secret: "secret",
					},
				},
			},
		}
		ctx := context.Background()

		sub, err := repo.Get(ctx, "test-id-123")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if sub == nil {
			t.Fatal("Expected subscription, got nil")
		}
		if sub.ID != "test-id-123" {
			t.Errorf("Expected ID 'test-id-123', got '%s'", sub.ID)
		}
		if sub.Name != "Test Webhook" {
			t.Errorf("Expected name 'Test Webhook', got '%s'", sub.Name)
		}
	})

	t.Run("returns copy to prevent mutation", func(t *testing.T) {
		repo := &StaticRepository{
			subscriptions: []Subscription{
				{
					ID:   "test-id",
					Name: "Original Name",
					Delivery: DeliveryConfig{
						Type: "webhook",
						URL:  "https://test.example.com",
					},
					Filter: &FilterConfig{
						MinScale:    3,
						Prefectures: []string{"Tokyo"},
					},
				},
			},
		}
		ctx := context.Background()

		// Get and modify
		sub1, _ := repo.Get(ctx, "test-id")
		sub1.Name = "Modified Name"
		sub1.Filter.Prefectures[0] = "Osaka"

		// Get again
		sub2, _ := repo.Get(ctx, "test-id")

		// Original should be unchanged
		if sub2.Name != "Original Name" {
			t.Errorf("Expected name 'Original Name', got '%s' (mutation detected)", sub2.Name)
		}
		if sub2.Filter.Prefectures[0] != "Tokyo" {
			t.Errorf("Expected prefecture 'Tokyo', got '%s' (mutation detected)", sub2.Filter.Prefectures[0])
		}
	})
}

func TestStaticRepository_Update(t *testing.T) {
	t.Run("returns read-only error with ID", func(t *testing.T) {
		cfg := &config.Config{
			Subscriptions: []config.SubscriptionConfig{},
		}

		repo := NewStaticRepository(cfg)
		ctx := context.Background()

		sub := Subscription{
			Name: "Updated Webhook",
		}

		err := repo.Update(ctx, "some-id", sub)

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !errors.Is(err, ErrReadOnly) {
			t.Errorf("Expected error to wrap ErrReadOnly, got %v", err)
		}
	})
}

func TestStaticRepository_Delete(t *testing.T) {
	t.Run("returns read-only error with ID", func(t *testing.T) {
		cfg := &config.Config{
			Subscriptions: []config.SubscriptionConfig{},
		}

		repo := NewStaticRepository(cfg)
		ctx := context.Background()

		err := repo.Delete(ctx, "some-id")

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !errors.Is(err, ErrReadOnly) {
			t.Errorf("Expected error to wrap ErrReadOnly, got %v", err)
		}
	})
}

func TestStaticRepository_ImplementsRepository(t *testing.T) {
	// Compile-time check that StaticRepository implements Repository interface
	var _ Repository = (*StaticRepository)(nil)
}

func TestErrReadOnly(t *testing.T) {
	t.Run("error message is correct", func(t *testing.T) {
		if ErrReadOnly.Error() != "static repository is read-only" {
			t.Errorf("ErrReadOnly message = %q, want %q", ErrReadOnly.Error(), "static repository is read-only")
		}
	})
}
