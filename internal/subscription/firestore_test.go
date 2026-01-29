package subscription

import (
	"testing"
)

func TestNewFirestoreRepository(t *testing.T) {
	t.Run("creates repository with nil client", func(t *testing.T) {
		repo := NewFirestoreRepository(nil)

		if repo == nil {
			t.Fatal("NewFirestoreRepository returned nil")
		}
		if repo.client != nil {
			t.Error("Expected client to be nil")
		}
	})
}

func TestFirestoreRepository_ImplementsRepository(t *testing.T) {
	// Compile-time check that FirestoreRepository implements Repository interface
	var _ Repository = (*FirestoreRepository)(nil)
}

func TestSubscriptionToMap(t *testing.T) {
	t.Run("converts subscription without filter", func(t *testing.T) {
		sub := Subscription{
			ID:   "test-id",
			Name: "Test Subscription",
			Delivery: DeliveryConfig{
				Type:   "webhook",
				URL:    "https://example.com/webhook",
				Secret: "secret123",
			},
		}

		data := subscriptionToMap(sub)

		if data["name"] != "Test Subscription" {
			t.Errorf("Expected name 'Test Subscription', got %v", data["name"])
		}

		delivery, ok := data["delivery"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected delivery to be a map")
		}
		if delivery["type"] != "webhook" {
			t.Errorf("Expected delivery type 'webhook', got %v", delivery["type"])
		}
		if delivery["url"] != "https://example.com/webhook" {
			t.Errorf("Expected delivery url 'https://example.com/webhook', got %v", delivery["url"])
		}
		if delivery["secret"] != "secret123" {
			t.Errorf("Expected delivery secret 'secret123', got %v", delivery["secret"])
		}

		if _, exists := data["filter"]; exists {
			t.Error("Expected filter to not exist for subscription without filter")
		}
	})

	t.Run("converts subscription with filter", func(t *testing.T) {
		sub := Subscription{
			ID:   "test-id",
			Name: "Filtered Subscription",
			Delivery: DeliveryConfig{
				Type:   "webhook",
				URL:    "https://example.com/webhook",
				Secret: "secret123",
			},
			Filter: &FilterConfig{
				MinScale:    3,
				Prefectures: []string{"Tokyo", "Osaka"},
			},
		}

		data := subscriptionToMap(sub)

		filter, ok := data["filter"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected filter to be a map")
		}
		if filter["minScale"] != 3 {
			t.Errorf("Expected minScale 3, got %v", filter["minScale"])
		}
		prefectures, ok := filter["prefectures"].([]string)
		if !ok {
			t.Fatal("Expected prefectures to be a string slice")
		}
		if len(prefectures) != 2 {
			t.Errorf("Expected 2 prefectures, got %d", len(prefectures))
		}
		if prefectures[0] != "Tokyo" {
			t.Errorf("Expected first prefecture 'Tokyo', got %s", prefectures[0])
		}
		if prefectures[1] != "Osaka" {
			t.Errorf("Expected second prefecture 'Osaka', got %s", prefectures[1])
		}
	})

	t.Run("does not include ID in map", func(t *testing.T) {
		sub := Subscription{
			ID:   "test-id",
			Name: "Test",
			Delivery: DeliveryConfig{
				Type: "webhook",
			},
		}

		data := subscriptionToMap(sub)

		if _, exists := data["id"]; exists {
			t.Error("ID should not be included in the map (it's the document ID)")
		}
	})
}

func TestErrNotFound(t *testing.T) {
	t.Run("error message is correct", func(t *testing.T) {
		if ErrNotFound.Error() != "subscription not found" {
			t.Errorf("ErrNotFound message = %q, want %q", ErrNotFound.Error(), "subscription not found")
		}
	})
}

func TestCollectionName(t *testing.T) {
	t.Run("collection name is correct", func(t *testing.T) {
		if collectionName != "subscriptions" {
			t.Errorf("collectionName = %q, want %q", collectionName, "subscriptions")
		}
	})
}
