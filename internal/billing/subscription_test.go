package billing

import (
	"context"
	"testing"
)

func TestBuildCheckoutSessionParams(t *testing.T) {
	t.Run("builds params with all required fields", func(t *testing.T) {
		params := buildCheckoutSessionParams(
			"cus_123",
			"price_456",
			"https://example.com/success?session_id={CHECKOUT_SESSION_ID}",
			"https://example.com/cancel",
		)

		if params.Customer == nil || *params.Customer != "cus_123" {
			t.Errorf("Expected customer 'cus_123', got %v", params.Customer)
		}
		if params.SuccessURL == nil || *params.SuccessURL != "https://example.com/success?session_id={CHECKOUT_SESSION_ID}" {
			t.Errorf("Expected success URL, got %v", params.SuccessURL)
		}
		if params.CancelURL == nil || *params.CancelURL != "https://example.com/cancel" {
			t.Errorf("Expected cancel URL, got %v", params.CancelURL)
		}
		if params.Mode == nil || *params.Mode != "subscription" {
			t.Errorf("Expected mode 'subscription', got %v", params.Mode)
		}
		if len(params.LineItems) != 1 {
			t.Fatalf("Expected 1 line item, got %d", len(params.LineItems))
		}
		if params.LineItems[0].Price == nil || *params.LineItems[0].Price != "price_456" {
			t.Errorf("Expected price 'price_456', got %v", params.LineItems[0].Price)
		}
		if params.LineItems[0].Quantity == nil || *params.LineItems[0].Quantity != 1 {
			t.Errorf("Expected quantity 1, got %v", params.LineItems[0].Quantity)
		}
	})
}

func TestBuildPortalSessionParams(t *testing.T) {
	t.Run("builds params with customer and return URL", func(t *testing.T) {
		params := buildPortalSessionParams(
			"cus_123",
			"https://example.com/billing",
		)

		if params.Customer == nil || *params.Customer != "cus_123" {
			t.Errorf("Expected customer 'cus_123', got %v", params.Customer)
		}
		if params.ReturnURL == nil || *params.ReturnURL != "https://example.com/billing" {
			t.Errorf("Expected return URL, got %v", params.ReturnURL)
		}
	})
}

func TestCreateCheckoutSession_RequiresStripeConnection(t *testing.T) {
	t.Run("returns error without valid Stripe connection", func(t *testing.T) {
		client := NewClient("invalid_key")

		_, err := client.CreateCheckoutSession(
			context.Background(),
			"cus_123",
			"price_456",
			"https://example.com/success",
			"https://example.com/cancel",
		)
		if err == nil {
			t.Error("Expected error when calling Stripe with invalid key")
		}
	})
}

func TestCreatePortalSession_RequiresStripeConnection(t *testing.T) {
	t.Run("returns error without valid Stripe connection", func(t *testing.T) {
		client := NewClient("invalid_key")

		_, err := client.CreatePortalSession(
			context.Background(),
			"cus_123",
			"https://example.com/billing",
		)
		if err == nil {
			t.Error("Expected error when calling Stripe with invalid key")
		}
	})
}

func TestGetSubscription_RequiresStripeConnection(t *testing.T) {
	t.Run("returns error without valid Stripe connection", func(t *testing.T) {
		client := NewClient("invalid_key")

		_, err := client.GetSubscription(context.Background(), "sub_123")
		if err == nil {
			t.Error("Expected error when calling Stripe with invalid key")
		}
	})
}
