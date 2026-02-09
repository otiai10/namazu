package billing

import (
	"testing"
	"time"

	"github.com/stripe/stripe-go/v78"
)

func TestVerifyWebhookSignature(t *testing.T) {
	t.Run("returns error for invalid signature", func(t *testing.T) {
		payload := []byte(`{"type":"test"}`)
		signature := "invalid_signature"
		secret := "whsec_test123"

		_, err := VerifyWebhookSignature(payload, signature, secret)
		if err == nil {
			t.Error("Expected error for invalid signature")
		}
	})

	t.Run("returns error for empty payload", func(t *testing.T) {
		payload := []byte{}
		signature := "t=1234567890,v1=signature"
		secret := "whsec_test123"

		_, err := VerifyWebhookSignature(payload, signature, secret)
		if err == nil {
			t.Error("Expected error for empty payload")
		}
	})

	t.Run("returns error for empty signature", func(t *testing.T) {
		payload := []byte(`{"type":"test"}`)
		signature := ""
		secret := "whsec_test123"

		_, err := VerifyWebhookSignature(payload, signature, secret)
		if err == nil {
			t.Error("Expected error for empty signature")
		}
	})
}

func TestParseCheckoutSessionCompleted(t *testing.T) {
	t.Run("extracts customer and subscription IDs", func(t *testing.T) {
		session := &stripe.CheckoutSession{
			ID:           "cs_123",
			Customer:     &stripe.Customer{ID: "cus_456"},
			Subscription: &stripe.Subscription{ID: "sub_789"},
			Mode:         stripe.CheckoutSessionModeSubscription,
		}

		customerID, subscriptionID := ParseCheckoutSessionCompleted(session)

		if customerID != "cus_456" {
			t.Errorf("Expected customer ID 'cus_456', got %s", customerID)
		}
		if subscriptionID != "sub_789" {
			t.Errorf("Expected subscription ID 'sub_789', got %s", subscriptionID)
		}
	})

	t.Run("handles nil customer", func(t *testing.T) {
		session := &stripe.CheckoutSession{
			ID:           "cs_123",
			Customer:     nil,
			Subscription: &stripe.Subscription{ID: "sub_789"},
		}

		customerID, subscriptionID := ParseCheckoutSessionCompleted(session)

		if customerID != "" {
			t.Errorf("Expected empty customer ID, got %s", customerID)
		}
		if subscriptionID != "sub_789" {
			t.Errorf("Expected subscription ID 'sub_789', got %s", subscriptionID)
		}
	})

	t.Run("handles nil subscription", func(t *testing.T) {
		session := &stripe.CheckoutSession{
			ID:           "cs_123",
			Customer:     &stripe.Customer{ID: "cus_456"},
			Subscription: nil,
		}

		customerID, subscriptionID := ParseCheckoutSessionCompleted(session)

		if customerID != "cus_456" {
			t.Errorf("Expected customer ID 'cus_456', got %s", customerID)
		}
		if subscriptionID != "" {
			t.Errorf("Expected empty subscription ID, got %s", subscriptionID)
		}
	})
}

func TestParseSubscriptionUpdate(t *testing.T) {
	t.Run("extracts subscription details", func(t *testing.T) {
		now := time.Now().Unix()
		sub := &stripe.Subscription{
			ID:                "sub_123",
			Customer:          &stripe.Customer{ID: "cus_456"},
			Status:            stripe.SubscriptionStatusActive,
			CurrentPeriodEnd:  now,
			CancelAtPeriodEnd: false,
		}

		info := ParseSubscriptionUpdate(sub)

		if info.SubscriptionID != "sub_123" {
			t.Errorf("Expected subscription ID 'sub_123', got %s", info.SubscriptionID)
		}
		if info.CustomerID != "cus_456" {
			t.Errorf("Expected customer ID 'cus_456', got %s", info.CustomerID)
		}
		if info.Status != "active" {
			t.Errorf("Expected status 'active', got %s", info.Status)
		}
		if info.PeriodEnd.Unix() != now {
			t.Errorf("Expected period end %d, got %d", now, info.PeriodEnd.Unix())
		}
		if info.CanceledAtPeriodEnd {
			t.Error("Expected CanceledAtPeriodEnd to be false")
		}
	})

	t.Run("handles canceled status", func(t *testing.T) {
		sub := &stripe.Subscription{
			ID:                "sub_123",
			Customer:          &stripe.Customer{ID: "cus_456"},
			Status:            stripe.SubscriptionStatusCanceled,
			CurrentPeriodEnd:  time.Now().Unix(),
			CancelAtPeriodEnd: true,
		}

		info := ParseSubscriptionUpdate(sub)

		if info.Status != "canceled" {
			t.Errorf("Expected status 'canceled', got %s", info.Status)
		}
		if !info.CanceledAtPeriodEnd {
			t.Error("Expected CanceledAtPeriodEnd to be true")
		}
	})

	t.Run("handles past_due status", func(t *testing.T) {
		sub := &stripe.Subscription{
			ID:               "sub_123",
			Customer:         &stripe.Customer{ID: "cus_456"},
			Status:           stripe.SubscriptionStatusPastDue,
			CurrentPeriodEnd: time.Now().Unix(),
		}

		info := ParseSubscriptionUpdate(sub)

		if info.Status != "past_due" {
			t.Errorf("Expected status 'past_due', got %s", info.Status)
		}
	})

	t.Run("handles nil customer", func(t *testing.T) {
		sub := &stripe.Subscription{
			ID:               "sub_123",
			Customer:         nil,
			Status:           stripe.SubscriptionStatusActive,
			CurrentPeriodEnd: time.Now().Unix(),
		}

		info := ParseSubscriptionUpdate(sub)

		if info.CustomerID != "" {
			t.Errorf("Expected empty customer ID, got %s", info.CustomerID)
		}
	})
}

func TestMapSubscriptionStatus(t *testing.T) {
	tests := []struct {
		input    stripe.SubscriptionStatus
		expected string
	}{
		{stripe.SubscriptionStatusActive, "active"},
		{stripe.SubscriptionStatusCanceled, "canceled"},
		{stripe.SubscriptionStatusPastDue, "past_due"},
		{stripe.SubscriptionStatusTrialing, "trialing"},
		{stripe.SubscriptionStatusIncomplete, "incomplete"},
		{stripe.SubscriptionStatusIncompleteExpired, "incomplete_expired"},
		{stripe.SubscriptionStatusUnpaid, "unpaid"},
		{stripe.SubscriptionStatusPaused, "paused"},
	}

	for _, tc := range tests {
		t.Run(string(tc.input), func(t *testing.T) {
			result := mapSubscriptionStatus(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestWebhookEventTypes(t *testing.T) {
	t.Run("event type constants are correct", func(t *testing.T) {
		if EventCheckoutSessionCompleted != "checkout.session.completed" {
			t.Errorf("Unexpected EventCheckoutSessionCompleted: %s", EventCheckoutSessionCompleted)
		}
		if EventSubscriptionUpdated != "customer.subscription.updated" {
			t.Errorf("Unexpected EventSubscriptionUpdated: %s", EventSubscriptionUpdated)
		}
		if EventSubscriptionDeleted != "customer.subscription.deleted" {
			t.Errorf("Unexpected EventSubscriptionDeleted: %s", EventSubscriptionDeleted)
		}
	})
}
