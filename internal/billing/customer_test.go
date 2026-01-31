package billing

import (
	"context"
	"testing"

	"github.com/ayanel/namazu/internal/user"
)

func TestBuildCustomerParams(t *testing.T) {
	t.Run("builds params from user", func(t *testing.T) {
		u := &user.User{
			ID:          "doc-123",
			UID:         "uid-456",
			Email:       "test@example.com",
			DisplayName: "Test User",
		}

		params := buildCustomerParams(u)

		if params.Email == nil || *params.Email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got %v", params.Email)
		}
		if params.Name == nil || *params.Name != "Test User" {
			t.Errorf("Expected name 'Test User', got %v", params.Name)
		}
		if params.Metadata == nil {
			t.Fatal("Expected metadata to be set")
		}
		if params.Metadata["user_id"] != "doc-123" {
			t.Errorf("Expected user_id 'doc-123', got %s", params.Metadata["user_id"])
		}
		if params.Metadata["uid"] != "uid-456" {
			t.Errorf("Expected uid 'uid-456', got %s", params.Metadata["uid"])
		}
	})

	t.Run("handles empty display name", func(t *testing.T) {
		u := &user.User{
			ID:    "doc-123",
			UID:   "uid-456",
			Email: "test@example.com",
		}

		params := buildCustomerParams(u)

		if params.Name == nil || *params.Name != "" {
			t.Errorf("Expected empty name, got %v", params.Name)
		}
	})
}

func TestGetOrCreateCustomer_RequiresStripeConnection(t *testing.T) {
	// This test validates error handling when Stripe is not properly configured
	t.Run("returns error without valid Stripe connection", func(t *testing.T) {
		client := NewClient("invalid_key")

		u := &user.User{
			ID:          "doc-123",
			UID:         "uid-456",
			Email:       "test@example.com",
			DisplayName: "Test User",
		}

		_, err := client.GetOrCreateCustomer(context.Background(), u)
		if err == nil {
			t.Error("Expected error when calling Stripe with invalid key")
		}
	})
}

func TestGetOrCreateCustomer_ExistingCustomerID(t *testing.T) {
	t.Run("returns existing customer ID if already set", func(t *testing.T) {
		client := NewClient("sk_test_123")

		u := &user.User{
			ID:               "doc-123",
			UID:              "uid-456",
			Email:            "test@example.com",
			StripeCustomerID: "cus_existing123",
		}

		customerID, err := client.GetOrCreateCustomer(context.Background(), u)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if customerID != "cus_existing123" {
			t.Errorf("Expected existing customer ID, got %s", customerID)
		}
	})
}
