package billing

import (
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"
)

// Webhook event type constants
const (
	EventCheckoutSessionCompleted = "checkout.session.completed"
	EventSubscriptionUpdated      = "customer.subscription.updated"
	EventSubscriptionDeleted      = "customer.subscription.deleted"
)

// SubscriptionInfo contains parsed subscription information from webhook events
type SubscriptionInfo struct {
	SubscriptionID      string
	CustomerID          string
	Status              string
	PeriodEnd           time.Time
	CanceledAtPeriodEnd bool
}

// VerifyWebhookSignature verifies the Stripe webhook signature and returns the event
//
// Parameters:
//   - payload: Raw request body
//   - signature: Stripe-Signature header value
//   - secret: Webhook signing secret
//
// Returns:
//   - Stripe event if signature is valid
//   - Error if verification fails
func VerifyWebhookSignature(payload []byte, signature, secret string) (stripe.Event, error) {
	event, err := webhook.ConstructEvent(payload, signature, secret)
	if err != nil {
		return stripe.Event{}, fmt.Errorf("webhook signature verification failed: %w", err)
	}

	return event, nil
}

// ParseCheckoutSessionCompleted extracts customer and subscription IDs from checkout.session.completed event
//
// Parameters:
//   - session: Stripe CheckoutSession object
//
// Returns:
//   - customerID: Stripe customer ID
//   - subscriptionID: Stripe subscription ID
func ParseCheckoutSessionCompleted(session *stripe.CheckoutSession) (customerID, subscriptionID string) {
	if session.Customer != nil {
		customerID = session.Customer.ID
	}
	if session.Subscription != nil {
		subscriptionID = session.Subscription.ID
	}
	return customerID, subscriptionID
}

// ParseSubscriptionUpdate extracts subscription details from subscription update/delete events
//
// Parameters:
//   - sub: Stripe Subscription object
//
// Returns:
//   - SubscriptionInfo containing parsed subscription details
func ParseSubscriptionUpdate(sub *stripe.Subscription) SubscriptionInfo {
	info := SubscriptionInfo{
		SubscriptionID:      sub.ID,
		Status:              mapSubscriptionStatus(sub.Status),
		PeriodEnd:           time.Unix(sub.CurrentPeriodEnd, 0),
		CanceledAtPeriodEnd: sub.CancelAtPeriodEnd,
	}

	if sub.Customer != nil {
		info.CustomerID = sub.Customer.ID
	}

	return info
}

// mapSubscriptionStatus maps Stripe subscription status to our internal status string
func mapSubscriptionStatus(status stripe.SubscriptionStatus) string {
	switch status {
	case stripe.SubscriptionStatusActive:
		return "active"
	case stripe.SubscriptionStatusCanceled:
		return "canceled"
	case stripe.SubscriptionStatusPastDue:
		return "past_due"
	case stripe.SubscriptionStatusTrialing:
		return "trialing"
	case stripe.SubscriptionStatusIncomplete:
		return "incomplete"
	case stripe.SubscriptionStatusIncompleteExpired:
		return "incomplete_expired"
	case stripe.SubscriptionStatusUnpaid:
		return "unpaid"
	case stripe.SubscriptionStatusPaused:
		return "paused"
	default:
		return string(status)
	}
}
